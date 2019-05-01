package tun2socks

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"time"
)

const (
	MTU = math.MaxInt16
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
	dnsProxy       *DnsProxy
	dataSource     VpnInputStream
	vpnWriteBack   io.WriteCloser
	localSocksAddr string
	protect        ConnProtect
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, locSocks string) (*Tun2Socks, error) {

	proxy, err := NewDnsCache(protect)
	if err != nil {
		return nil, err
	}
	proxy.VpnWriteBack = writer
	tsc := &Tun2Socks{
		dnsProxy:       proxy,
		dataSource:     reader,
		vpnWriteBack:   writer,
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

		var ip4 layers.IPv4
		var tcp layers.TCP
		var udp layers.UDP
		var dns layers.DNS
		var payload gopacket.Payload
		parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ip4, &tcp, &udp, &dns, &payload)
		decodedLayers := make([]gopacket.LayerType, 0, 10)

		if err := parser.DecodeLayers(buf, &decodedLayers); err != nil {
			log.Println("  Error encountered:", err)
			continue
		}
		var i = 0
		for _, typ := range decodedLayers {
			i++
			//log.Println("  Successfully decoded layer type", typ)
			switch typ {
			case layers.LayerTypeDNS:
				log.Println(i, "    DNS ", dns.Questions, ip4.DstIP, int(udp.DstPort))
				t2s.dnsProxy.sendOut(&dns, &ip4, &udp)
			case layers.LayerTypeIPv4:
				log.Println(i, "    IP4 ", ip4.SrcIP, ip4.DstIP)
				break
			case layers.LayerTypeTCP:
				log.Println(i, "    TCP ", tcp.SrcPort, tcp.DstPort)
				break
			case layers.LayerTypeUDP:
				log.Println(i, "    UDP ", udp.SrcPort, udp.DstPort)
				break
			}
		}
	}
}

func (t2s *Tun2Socks) Close() {
}
