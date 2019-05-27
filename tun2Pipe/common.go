package tun2Pipe

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"net/http"
	"syscall"
)

const TLSHeaderLength = 5

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
		VpnInstance.Log(fmt.Sprintln("Udp Wrap ip packet check sum err:", err))
		return nil
	}

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, udp, gopacket.Payload(payload)); err != nil {
		VpnInstance.Log(fmt.Sprintln("Wrap dns to ip packet  err:", err))
		return nil
	}

	return b.Bytes()
}

func ProtectConn(conn syscall.Conn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		VpnInstance.Log(fmt.Sprintln("Protect Err:", err))
		return err
	}

	if err := rawConn.Control(Protector); err != nil {
		VpnInstance.Log(fmt.Sprintln("Protect Err:", err))
		return err
	}

	return nil
}

func ChangePacket(ip4 *layers.IPv4, tcp *layers.TCP) []byte {

	if err := tcp.SetNetworkLayerForChecksum(ip4); err != nil {
		VpnInstance.Log(fmt.Sprintln("set tcp layer check sum err:", err))
		return nil
	}

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, tcp, gopacket.Payload(tcp.Payload)); err != nil {
		VpnInstance.Log(fmt.Sprintln("Wrap Tcp to ip packet  err:", err))
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
		VpnInstance.Log(fmt.Sprintln(" SYN"))
	}
	if tcp.ACK {
		VpnInstance.Log(fmt.Sprintln(" ACK"))
	}
	if tcp.FIN {
		VpnInstance.Log(fmt.Sprintln(" FIN"))
	}
	if tcp.PSH {
		VpnInstance.Log(fmt.Sprintln(" PSH"))
	}
	if tcp.RST {
		VpnInstance.Log(fmt.Sprintln(" RST"))
	}
	fmt.Println()
}

func ParseHost(data []byte) string {
	switch data[0] {
	//GET,HEAD,POST,PUT,DELETE,OPTIONS,TRACE,CONNECT
	case 'G', 'H', 'P', 'D', 'O', 'T', 'C':
		{
			reader := bufio.NewReader(bytes.NewReader(data))
			if r, _ := http.ReadRequest(reader); r != nil {
				VpnInstance.Log(fmt.Sprintln("---===>>>Success host:", r.Host))
				return r.Host
			}
		}
	case 0x16:
		return getSNI(data)
	}
	return ""
}

func getSNI(data []byte) string {
	if len(data) == 0 || data[0] != 0x16 {
		return ""
	}

	extensions, err := GetExtensionBlock(data)
	if err != nil {
		return ""
	}
	sn, err := GetSNBlock(extensions)
	if err != nil {
		return ""
	}
	sni, err := GetSNIBlock(sn)
	if err != nil {
		return ""
	}

	VpnInstance.Log(fmt.Sprintln("---===>>>Success SNI:", string(sni)))
	return string(sni)
}

func GetSNIBlock(data []byte) ([]byte, error) {
	index := 0

	for {
		if index >= len(data) {
			break
		}
		length := int((data[index] << 8) + data[index+1])
		endIndex := index + 2 + length
		if data[index+2] == 0x00 { /* SNI */
			sni := data[index+3:]
			sniLength := int((sni[0] << 8) + sni[1])
			return sni[2 : sniLength+2], nil
		}
		index = endIndex
	}
	return []byte{}, fmt.Errorf(
		"finished parsing the SN block without finding an SNI",
	)
}
func GetExtensionBlock(data []byte) ([]byte, error) {
	/*   data[0]           - content type
	 *   data[1], data[2]  - major/minor version
	 *   data[3], data[4]  - total length
	 *   data[...38+5]     - start of SessionID (length bit)
	 *   data[38+5]        - length of SessionID
	 */
	var index = TLSHeaderLength + 38

	if len(data) <= index+1 {
		return []byte{}, fmt.Errorf("not enough bits to be a Client Hello")
	}

	/* Index is at SessionID Length bit */
	if newIndex := index + 1 + int(data[index]); (newIndex + 2) < len(data) {
		index = newIndex
	} else {
		return []byte{}, fmt.Errorf("not enough bytes for the SessionID")
	}

	/* Index is at Cipher List Length bits */
	if newIndex := index + 2 + int((data[index]<<8)+data[index+1]); (newIndex + 1) < len(data) {
		index = newIndex
	} else {
		return []byte{}, fmt.Errorf("not enough bytes for the Cipher List")
	}

	/* Index is now at the compression length bit */
	if newIndex := index + 1 + int(data[index]); newIndex < len(data) {
		index = newIndex
	} else {
		return []byte{}, fmt.Errorf("not enough bytes for the compression length")
	}

	/* Now we're at the Extension start */
	if len(data[index:]) == 0 {
		return nil, fmt.Errorf("no extensions")
	}
	return data[index:], nil
}

func GetSNBlock(data []byte) ([]byte, error) {
	index := 0

	if len(data) < 2 {
		return []byte{}, fmt.Errorf("not enough bytes to be an SN block")
	}

	extensionLength := int((data[index] << 8) + data[index+1])
	if extensionLength+2 > len(data) {
		return []byte{}, fmt.Errorf("extension looks bonkers")
	}
	data = data[2 : extensionLength+2]

	for {
		if index+3 >= len(data) {
			break
		}
		length := int((data[index+2] << 8) + data[index+3])
		endIndex := index + 4 + length
		if data[index] == 0x00 && data[index+1] == 0x00 {
			return data[index+4 : endIndex], nil
		}

		index = endIndex
	}

	return []byte{}, fmt.Errorf(
		"finished parsing the Extension block without finding an SN block",
	)
}
