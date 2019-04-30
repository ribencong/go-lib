package tun2socks

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
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
	dataSource     io.ReadCloser
	localSocksAddr string
	protect        ConnProtect
}

func New(reader io.ReadCloser, writer io.WriteCloser, protect ConnProtect, locSocks string) (*Tun2Socks, error) {

	tsc := &Tun2Socks{
		dataSource:     reader,
		localSocksAddr: locSocks,
		protect:        protect,
	}
	return tsc, nil
}

func (t2s *Tun2Socks) Reading() {
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var tcp layers.TCP
	var udp layers.UDP
	var dns layers.DNS
	var payload gopacket.Payload

	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &ip6, &tcp, &udp, &dns, &payload)
	decoded := make([]gopacket.LayerType, 0, 10)
	buf := make([]byte, math.MaxInt16)

	for {
		n, err := t2s.dataSource.Read(buf)
		if err != nil {
			log.Printf("error getting packet: %v", err)
			continue
		}

		if err = parser.DecodeLayers(buf[:n], &decoded); err != nil {
			log.Printf("error decoding packet: %v", err)
			continue
		}

		for _, typ := range decoded {
			fmt.Println("  Successfully decoded layer type", typ)
			switch typ {
			case layers.LayerTypeEthernet:
				fmt.Println("    Eth ", eth.SrcMAC, eth.DstMAC)
			case layers.LayerTypeIPv4:
				fmt.Println("    IP4 ", ip4.SrcIP, ip4.DstIP)
			case layers.LayerTypeIPv6:
				fmt.Println("    IP6 ", ip6.SrcIP, ip6.DstIP)
			case layers.LayerTypeTCP:
				fmt.Println("    TCP ", tcp.SrcPort, tcp.DstPort)
			case layers.LayerTypeUDP:
				fmt.Println("    UDP ", udp.SrcPort, udp.DstPort)
			case layers.LayerTypeDNS:
				fmt.Println("    DNS Questions: ", dns.Questions)
			}
		}
	}
}

func (t2s *Tun2Socks) Writing() {
}

func (t2s *Tun2Socks) Close() {
}
