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
	SysDialTimeOut    = time.Second * 2
	UDPSessionTimeOut = time.Second * 80
	MTU               = math.MaxInt16
	LocalProxyPort    = 51414
)

type ConnProtect func(fd uintptr)

var (
	SysConnProtector ConnProtect    = nil
	SysTunWriteBack  io.WriteCloser = nil
	SysTunLocalIP                   = net.ParseIP("10.8.0.2")
)

type Tun2Socks struct {
	udpProxy   *UdpProxy
	tcpProxy   *TcpProxy
	dataSource VpnInputStream
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, localIP string) (*Tun2Socks, error) {

	SysConnProtector = protect
	SysTunWriteBack = writer
	SysTunLocalIP = net.ParseIP(localIP)

	tcp, err := NewTcpProxy()
	if err != nil {
		return nil, err
	}
	tsc := &Tun2Socks{
		udpProxy:   NewUdpProxy(),
		tcpProxy:   tcp,
		dataSource: reader,
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
		var ip4 *layers.IPv4 = nil

		packet := gopacket.NewPacket(buf, layers.LayerTypeIPv4, gopacket.Default)
		if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
			ip4 = ip4Layer.(*layers.IPv4)
		} else {
			log.Println("Unsupported network layer :")
			continue
		}

		//var tcp *layers.TCP = nil
		//if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		//	tcp = tcpLayer.(*layers.TCP)
		//	t2s.tcpProxy.ReceivePacket(ip4, tcp)
		//	continue
		//}

		var udp *layers.UDP = nil
		if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
			udp = udpLayer.(*layers.UDP)
			t2s.udpProxy.ReceivePacket(ip4, udp)
			continue
		}

		//log.Println("Unsupported transport layer :", ip4.Protocol.String())
	}
}

func (t2s *Tun2Socks) Close() {
}

func WrapIPPacketForUdp(srcIp, DstIp net.IP, srcPort, dstPort int, payload []byte) []byte {

	ip4 := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    srcIp,
		DstIP:    DstIp,
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(dstPort),
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

func ProtectConn(conn syscall.Conn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		log.Println("Protect Err:", err)
		return err
	}

	if err := rawConn.Control(SysConnProtector); err != nil {
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
