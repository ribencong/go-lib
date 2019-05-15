package tun2socks

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"log"
	"net"
	"syscall"
)

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

func ProtectConn(conn syscall.Conn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		log.Println("Protect Err:", err)
		return err
	}

	if err := rawConn.Control(SysConfig.Protector); err != nil {
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

	if err := gopacket.SerializeLayers(b, opt, ip4, tcp, gopacket.Payload(tcp.Payload)); err != nil {
		log.Println("Wrap Tcp to ip packet  err:", err)
		return nil
	}

	return b.Bytes()
}

func PrintFlow(pre string, ip4 *layers.IPv4, tcp *layers.TCP) {
	fmt.Printf("%s	TCP <%d:%d> (%s:%d)->(%s:%d)",
		pre, tcp.Seq, tcp.Ack,
		ip4.SrcIP, tcp.SrcPort,
		ip4.DstIP, tcp.DstPort)

	if tcp.SYN {
		fmt.Print(" SYN")
	}
	if tcp.ACK {
		fmt.Print(" ACK")
	}
	if tcp.FIN {
		fmt.Print(" FIN")
	}
	if tcp.PSH {
		fmt.Print(" PSH")
	}
	if tcp.RST {
		fmt.Print(" RST")
	}
	fmt.Println()
}
