package tun2socks

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/qjpcpu/p2pbyudp/myp2p/Godeps/_workspace/src/github.com/ethereum/go-ethereum/common/math"
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
	dnsProxy       *net.UDPConn
	dataSource     VpnInputStream
	vpnWriteBack   io.WriteCloser
	localSocksAddr string
	protect        ConnProtect
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, locSocks string) (*Tun2Socks, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 51080,
	})
	if err != nil {
		return nil, err
	}
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}
	if err := rawConn.Control(protect); err != nil {
		return nil, err
	}

	tsc := &Tun2Socks{
		dnsProxy:       conn,
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

		for _, typ := range decodedLayers {
			//log.Println("  Successfully decoded layer type", typ)
			switch typ {
			case layers.LayerTypeDNS:
				t2s.dnsProxyOut(&dns, ip4.DstIP, int(udp.DstPort))
				//log.Println(dns.Questions, ip4.DstIP, int(udp.DstPort))
			case layers.LayerTypeIPv4:
				//log.Println("    IP4 ", ip4.SrcIP, ip4.DstIP)
				break
			case layers.LayerTypeTCP:
				//log.Println("    TCP ", tcp.SrcPort, tcp.DstPort)
				break
			case layers.LayerTypeUDP:
				//log.Println("    UDP ", udp.SrcPort, udp.DstPort)
				break
			}
		}
	}
}

func (t2s *Tun2Socks) Writing() {
}

func (t2s *Tun2Socks) Close() {
}
func makeIpPack(addr net.Addr, buff []byte) []byte {
	return nil
}
func (t2s *Tun2Socks) DnsWaitResponse() {
	buff := make([]byte, math.MaxInt16)
	defer t2s.dnsProxy.Close()
	for {
		n, rAddr, e := t2s.dnsProxy.ReadFromUDP(buff)
		if e != nil {
			log.Println("DNS exit:", e)
			return
		}

		data := makeIpPack(rAddr, buff)
		if _, err := t2s.vpnWriteBack.Write(data); err != nil {
			log.Println("DNS data write to Tun err:", err)
			continue
		}

		var decoded []gopacket.LayerType
		dns := &layers.DNS{}

		p := gopacket.NewDecodingLayerParser(layers.LayerTypeDNS, dns)
		if err := p.DecodeLayers(buff[:n], &decoded); err != nil {
			log.Println("DNS-parser:", err)
			continue
		}

		log.Println("DNS-Response-Success:", gopacket.LayerDump(dns))
	}
}

func (t2s *Tun2Socks) dnsProxyOut(dns *layers.DNS, remoteIp net.IP, remotePort int) {
	fmt.Printf("DNS Query: name:%s->%s:%d",
		dns.Questions[0].Name, remoteIp, remotePort)

	if _, e := t2s.dnsProxy.WriteTo(dns.LayerContents(), &net.UDPAddr{
		IP:   remoteIp,
		Port: remotePort,
	}); e != nil {
		log.Println("DNS Query err:", e)
		return
	}
	log.Println("DNS Query Send Success:")

}
