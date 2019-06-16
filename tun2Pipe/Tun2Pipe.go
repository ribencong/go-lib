package tun2Pipe

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	SysDialTimeOut    = time.Second * 2
	UDPSessionTimeOut = time.Second * 80
	MTU               = math.MaxInt16
	InnerPivotPort    = 51414
)

type Tun2Pipe struct {
	sync.RWMutex
	innerTcpPivot *net.TCPListener
	SessionCache  map[int]*Session
	udpProxy      *UdpProxy
	proxyPort     int
	tunIP         net.IP
}

type VpnDelegate interface {
	ByPass(fd int32) bool
	io.Writer
	Log(str string)
}

var VpnInstance VpnDelegate = nil

type ConnProtect func(fd uintptr)

var Protector ConnProtect

func New(proxyAddr string) (*Tun2Pipe, error) {

	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		Port: InnerPivotPort,
	})
	if e != nil {
		return nil, e
	}

	ip, port, _ := net.SplitHostPort(proxyAddr)
	intPort, _ := strconv.Atoi(port)

	tsc := &Tun2Pipe{
		innerTcpPivot: l,
		SessionCache:  make(map[int]*Session),
		udpProxy:      NewUdpProxy(),
		tunIP:         net.ParseIP(ip),
		proxyPort:     intPort,
	}

	go tsc.Pivoting()

	return tsc, nil
}

func (t2s *Tun2Pipe) GetTarget(conn net.Conn) string {
	keyPort := conn.RemoteAddr().(*net.TCPAddr).Port
	s := t2s.GetSession(keyPort)
	if s == nil {
		return ""
	}

	if len(s.HostName) != 0 {
		return fmt.Sprintf("%s:%d", s.HostName, s.RemotePort)
	}

	addr := &net.TCPAddr{
		IP:   s.RemoteIP,
		Port: s.RemotePort,
	}

	return addr.String()
}

func (t2s *Tun2Pipe) InputPacket(buf []byte) {
	//buf := VpnInstance.ReadBuff()
	//if len(buf) == 0 {
	//	time.Sleep(time.Millisecond * 100)
	//	continue
	//}
	var ip4 *layers.IPv4 = nil
	packet := gopacket.NewPacket(buf, layers.LayerTypeIPv4, gopacket.Default)

	VpnInstance.Log(packet.Dump())

	if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
		ip4 = ip4Layer.(*layers.IPv4)
	} else {
		VpnInstance.Log("Unsupported network layer :")
		return
	}

	var tcp *layers.TCP = nil
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp = tcpLayer.(*layers.TCP)
		t2s.ProcessTcpPacket(ip4, tcp)
		return
	}

	var udp *layers.UDP = nil
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp = udpLayer.(*layers.UDP)
		t2s.udpProxy.ReceivePacket(ip4, udp)
		return
	}

	VpnInstance.Log(fmt.Sprintf("Unsupported transport layer :%s", ip4.Protocol.String()))
}

func (t2s *Tun2Pipe) Finish() {
}

func (t2s *Tun2Pipe) tun2Proxy(ip4 *layers.IPv4, tcp *layers.TCP) {

	//PrintFlow("-=->tun2Proxy", ip4, tcp)
	srcPort := int(tcp.SrcPort)
	s := t2s.GetSession(srcPort)
	if s == nil {
		var srvPort = InnerPivotPort
		bypass := ByPassInst().Hit(ip4.DstIP)
		if !bypass {
			srvPort = t2s.proxyPort
			VpnInstance.Log(fmt.Sprintln("This session will be proxy:", ip4.DstIP, tcp.DstPort))
		}

		s = newSession(ip4, tcp, srvPort, bypass)
		t2s.AddSession(srcPort, s)
	}

	tcpLen := len(tcp.Payload)

	s.UPTime = time.Now()
	s.packetSent++
	if s.packetSent == 2 && tcpLen == 0 {
		VpnInstance.Log(fmt.Sprintf("Discard the ack:%t\n syn:%t psh:%t", tcp.ACK, tcp.SYN, tcp.PSH))
		return
	}

	if s.byteSent == 0 && tcpLen > 10 {
		host := ParseHost(tcp.Payload)
		if len(host) > 0 {
			VpnInstance.Log(fmt.Sprintln("Session host success:", host))
			s.HostName = host
		}
	}

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = t2s.tunIP
	tcp.DstPort = layers.TCPPort(s.ServerPort)

	data := ChangePacket(ip4, tcp)
	//PrintFlow("-=->tun2Proxy", ip4, tcp)

	if _, err := VpnInstance.Write(data); err != nil {
		VpnInstance.Log(fmt.Sprintln("-=->tun2Proxy write to tun err:", err))
		return
	}
	s.byteSent += tcpLen
}

func (t2s *Tun2Pipe) proxy2Tun(ip4 *layers.IPv4, tcp *layers.TCP, rPort int) {
	//PrintFlow("<-=-proxy2Tun", ip4, tcp)

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = t2s.tunIP
	tcp.SrcPort = layers.TCPPort(rPort)
	data := ChangePacket(ip4, tcp)

	//PrintFlow("<-=-proxy2Tun", ip4, tcp)
	if _, err := VpnInstance.Write(data); err != nil {
		VpnInstance.Log(fmt.Sprintln("<-=-proxy2Tun write to tun err:", err))
		return
	}
}

func (t2s *Tun2Pipe) ProcessTcpPacket(ip4 *layers.IPv4, tcp *layers.TCP) {
	srcPort := int(tcp.SrcPort)
	if srcPort == InnerPivotPort ||
		srcPort == t2s.proxyPort {
		dstPort := int(tcp.DstPort)
		if s := t2s.GetSession(dstPort); s != nil {
			t2s.proxy2Tun(ip4, tcp, s.RemotePort)
		}
		return
	}

	t2s.tun2Proxy(ip4, tcp)
}

func (t2s *Tun2Pipe) GetSession(key int) *Session {
	t2s.RLock()
	defer t2s.RUnlock()
	return t2s.SessionCache[key]
}
func (t2s *Tun2Pipe) AddSession(portKey int, s *Session) {
	t2s.Lock()
	defer t2s.Unlock()
	t2s.SessionCache[portKey] = s
}

//TODO:: remove the old session
func (t2s *Tun2Pipe) RemoveSession(key int) {
	t2s.Lock()
	defer t2s.Unlock()
	delete(t2s.SessionCache, key)
}
