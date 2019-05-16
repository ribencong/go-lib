package tun2Pipe

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
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

type Tun2Socks struct {
	sync.RWMutex
	innerTcpPivot *net.TCPListener
	SessionCache  map[int]*Session
	udpProxy      *UdpProxy
	proxyPort     int
	tunIP         net.IP
}

type VpnDelegate interface {
	ReadBuff() []byte
	ByPass(fd int32) bool
	io.Writer
}

var VpnInstance VpnDelegate = nil

type ConnProtect func(fd uintptr)

var Protector ConnProtect

func New(proxyAddr string) (*Tun2Socks, error) {

	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		Port: InnerPivotPort,
	})
	if e != nil {
		return nil, e
	}

	ip, port, _ := net.SplitHostPort(proxyAddr)
	intPort, _ := strconv.Atoi(port)

	tsc := &Tun2Socks{
		innerTcpPivot: l,
		SessionCache:  make(map[int]*Session),
		udpProxy:      NewUdpProxy(),
		tunIP:         net.ParseIP(ip),
		proxyPort:     intPort,
	}

	go tsc.Pivoting()
	go tsc.ReadTunData()

	return tsc, nil
}

func (t2s *Tun2Socks) GetTarget(conn net.Conn) string {
	keyPort := conn.RemoteAddr().(*net.TCPAddr).Port
	s := t2s.GetSession(keyPort)
	if s == nil {
		return ""
	}

	addr := &net.TCPAddr{
		IP:   s.RemoteIP,
		Port: s.RemotePort,
	}

	return addr.String()
}

func (t2s *Tun2Socks) ReadTunData() {
	for {
		buf := VpnInstance.ReadBuff()
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

		var tcp *layers.TCP = nil
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp = tcpLayer.(*layers.TCP)
			t2s.ProcessTcpPacket(ip4, tcp)
			continue
		}

		var udp *layers.UDP = nil
		if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
			udp = udpLayer.(*layers.UDP)
			t2s.udpProxy.ReceivePacket(ip4, udp)
			continue
		}

		log.Println("Unsupported transport layer :", ip4.Protocol.String())
	}
}

func (t2s *Tun2Socks) Close() {
}

func (t2s *Tun2Socks) tun2Proxy(ip4 *layers.IPv4, tcp *layers.TCP) {

	//PrintFlow("-=->tun2Proxy", ip4, tcp)
	srcPort := int(tcp.SrcPort)
	s := t2s.GetSession(srcPort)
	if s == nil {
		var srvPort = InnerPivotPort
		bypass := ByPassInst().Hit(ip4.DstIP)
		if !bypass {
			srvPort = t2s.proxyPort
		}

		s = newSession(ip4, tcp, srvPort, bypass)
		t2s.AddSession(srcPort, s)
	}

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = t2s.tunIP
	tcp.DstPort = layers.TCPPort(s.ServerPort)

	data := ChangePacket(ip4, tcp)
	//PrintFlow("-=->tun2Proxy", ip4, tcp)

	if _, err := VpnInstance.Write(data); err != nil {
		log.Println("-=->tun2Proxy write to tun err:", err)
		return
	}
}

func (t2s *Tun2Socks) proxy2Tun(ip4 *layers.IPv4, tcp *layers.TCP, rPort int) {
	//PrintFlow("<-=-proxy2Tun", ip4, tcp)

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = t2s.tunIP
	tcp.SrcPort = layers.TCPPort(rPort)
	data := ChangePacket(ip4, tcp)

	//PrintFlow("<-=-proxy2Tun", ip4, tcp)
	if _, err := VpnInstance.Write(data); err != nil {
		log.Println("<-=-proxy2Tun write to tun err:", err)
		return
	}
}

func (t2s *Tun2Socks) ProcessTcpPacket(ip4 *layers.IPv4, tcp *layers.TCP) {
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

func (t2s *Tun2Socks) GetSession(key int) *Session {
	t2s.RLock()
	defer t2s.RUnlock()
	return t2s.SessionCache[key]
}
func (t2s *Tun2Socks) AddSession(portKey int, s *Session) {
	t2s.Lock()
	defer t2s.Unlock()
	t2s.SessionCache[portKey] = s
}

//TODO:: remove the old session
func (t2s *Tun2Socks) RemoveSession(key int) {
	t2s.Lock()
	defer t2s.Unlock()
	delete(t2s.SessionCache, key)
}
