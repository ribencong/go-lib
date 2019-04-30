package tun2socks

import (
	"golang.org/x/net/ipv4"
	"io"
	"log"
	"net"
	"time"
)

const (
	MTU = 1500
)

var (
	_, ip1, _ = net.ParseCIDR("10.0.0.0/8")
	_, ip2, _ = net.ParseCIDR("172.16.0.0/12")
	_, ip3, _ = net.ParseCIDR("192.168.0.0/24")
)

func isPrivate(ip net.IP) bool {
	return ip1.Contains(ip) || ip2.Contains(ip) || ip3.Contains(ip)
}

type ConnProtect func(fd uintptr)

type Tun2Socks struct {
	dataSource     VpnInputStream
	localSocksAddr string
	protect        ConnProtect
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, locSocks string) (*Tun2Socks, error) {
	tsc := &Tun2Socks{
		dataSource:     reader,
		localSocksAddr: locSocks,
		protect:        protect,
	}
	return tsc, nil
}

func (t2s *Tun2Socks) Reading() {
	for {
		buf := t2s.dataSource.ReadBuff()
		if len(buf) == 0 {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		log.Printf("(%d)[%02x]", len(buf), buf)

		header, err := ipv4.ParseHeader(buf)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		log.Printf("[%s]->[%s]", header.Src, header.Dst)
	}
}

func (t2s *Tun2Socks) Writing() {
}

func (t2s *Tun2Socks) Close() {
}
