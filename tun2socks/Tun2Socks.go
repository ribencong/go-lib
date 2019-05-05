package tun2socks

import (
	"fmt"
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

		ip4 := &layers.IPv4{}
		tcp := &layers.TCP{}
		udp := &layers.UDP{}
		dns := &layers.DNS{}
		parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, ip4, tcp, udp, dns)
		decodedLayers := make([]gopacket.LayerType, 0, 4)
		if err := parser.DecodeLayers(buf, &decodedLayers); err != nil {
			continue
		}

		for _, typ := range decodedLayers {
			fmt.Println("  Successfully decoded layer type", typ)

			switch typ {
			case layers.LayerTypeDNS:
				if dns.QDCount == 0 {
					continue
				}

				log.Println("	DNS :", dns.ID, string(dns.Questions[0].Name))
				t2s.dnsProxy.sendOut(dns, ip4, udp)

				break
			case layers.LayerTypeTCP:
				break
			case layers.LayerTypeUDP:
				break
			case layers.LayerTypeIPv4:
				if !ip4.DstIP.IsGlobalUnicast() || isPrivate(ip4.DstIP) {
					log.Println("Private data:", ip4.DstIP)
					continue
				}
				if ip4.Flags&0x1 != 0 || ip4.FragOffset != 0 {
					log.Println("Fragment data:", ip4.DstIP)
					continue
				}
				break
			}
		}
	}
}

func (t2s *Tun2Socks) Close() {
}

func WrapIPPacket(srcIp, DstIp net.IP, srcPort, dstPort layers.UDPPort, payload []byte) []byte {

	ip4 := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    srcIp,
		DstIP:    DstIp,
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: srcPort,
		DstPort: dstPort,
	}

	udp.SetNetworkLayerForChecksum(ip4)

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, udp, gopacket.Payload(payload)); err != nil {
		log.Println("Wrap dns to ip packet  err:", err)
		return nil
	}

	return b.Bytes()
}
