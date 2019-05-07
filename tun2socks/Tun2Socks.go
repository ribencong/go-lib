package tun2socks

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"syscall"
	"time"
)

const (
	MTU            = math.MaxInt16
	LocalProxyPort = 51414
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
	dnsProxy     *DnsProxy
	tcpProxy     *TcpProxy
	dataSource   VpnInputStream
	vpnWriteBack io.WriteCloser
	protect      ConnProtect
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, localIP string) (*Tun2Socks, error) {

	dns, err := NewDnsCache(protect)
	if err != nil {
		return nil, err
	}
	dns.VpnWriteBack = writer
	tcp, err := NewTcpProxy(localIP, protect, writer)
	if err != nil {
		return nil, err
	}
	tsc := &Tun2Socks{
		dnsProxy:     dns,
		tcpProxy:     tcp,
		dataSource:   reader,
		vpnWriteBack: writer,
		protect:      protect,
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
		err := parser.DecodeLayers(buf, &decodedLayers)
		for _, typ := range decodedLayers {
			log.Println("  Successfully decoded layer type", typ)
			switch typ {
			case layers.LayerTypeDNS:
				if dns.QDCount == 0 {
					continue
				}
				t2s.dnsProxy.sendOut(dns, ip4, udp)

				break
			case layers.LayerTypeTCP:
				//log.Printf("before:%02x", buf)
				t2s.tcpProxy.ReceivePacket(ip4, tcp)
				break
			case layers.LayerTypeUDP:
				break
			case layers.LayerTypeIPv4:
				if !ip4.DstIP.IsGlobalUnicast() || isPrivate(ip4.DstIP) {
					log.Println("Warning-------->Private data:", ip4.DstIP)
					continue
				}
				if ip4.Flags&0x1 != 0 || ip4.FragOffset != 0 {
					log.Println("Warning-------->Fragment data:", ip4.DstIP)
					continue
				}
				break
			}
		}
		if err != nil {
			log.Println("  Error encountered:", err)
			continue
		}
	}
}

func (t2s *Tun2Socks) Close() {
}

func WrapIPPacketForUdp(srcIp, DstIp net.IP, srcPort, dstPort layers.UDPPort, payload []byte) []byte {

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

	if err := udp.SetNetworkLayerForChecksum(ip4); err != nil {
		log.Println("Udp Wrap ip packet check sum err:", err)
		return nil
	}

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
func WrapIPPacketForTcp(srcIp, DstIp net.IP, srcPort, dstPort layers.TCPPort, payload []byte) []byte {
	ip4 := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    srcIp,
		DstIP:    DstIp,
		Protocol: layers.IPProtocolTCP,
	}

	tcp := &layers.TCP{
		SrcPort: srcPort,
		DstPort: dstPort,
	}

	if err := tcp.SetNetworkLayerForChecksum(ip4); err != nil {
		log.Println("Tcp Wrap ip packet check sum err:", err)
		return nil
	}

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, tcp, gopacket.Payload(payload)); err != nil {
		log.Println("Wrap Tcp to ip packet  err:", err)
		return nil
	}
	return b.Bytes()
}

func ProtectConn(conn syscall.Conn, protect ConnProtect) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		log.Println("Protect Err:", err)
		return err
	}

	if err := rawConn.Control(protect); err != nil {
		log.Println("Protect Err:", err)
		return err
	}

	return nil
}

func ChangePacket(ip4 *layers.IPv4, tcp *layers.TCP) []byte {

	tcp.SetNetworkLayerForChecksum(ip4)

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, tcp); err != nil {
		log.Println("Wrap Tcp to ip packet  err:", err)
		return nil
	}

	return b.Bytes()
}
