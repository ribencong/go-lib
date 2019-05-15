package tun2socks

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

const (
	SysDialTimeOut    = time.Second * 2
	UDPSessionTimeOut = time.Second * 80
	MTU               = math.MaxInt16
	LocalProxyPort    = 51414
)

type ConnProtect func(fd uintptr)

type Tun2Socks struct {
	sync.RWMutex
	innerTcpPivot *net.TCPListener
	SessionCache  map[int]*Session
	udpProxy      *UdpProxy
	dataSource    VpnInputStream
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream) (*Tun2Socks, error) {

	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		Port: LocalProxyPort,
	})
	if e != nil {
		return nil, e
	}
	tsc := &Tun2Socks{
		innerTcpPivot: l,
		SessionCache:  make(map[int]*Session),
		udpProxy:      NewUdpProxy(),
		dataSource:    reader,
	}
	go tsc.Transfer()

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

		var tcp *layers.TCP = nil
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp = tcpLayer.(*layers.TCP)
			t2s.ReceivePacket(ip4, tcp)
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

	s := t2s.GetSession(int(tcp.SrcPort))
	if s == nil {
		s = newSession(ip4, tcp)
		t2s.AddSession(int(tcp.SrcPort), s)
	}

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = SysConfig.TunLocalIP
	tcp.DstPort = LocalProxyPort

	data := ChangePacket(ip4, tcp)
	//log.Printf("After:%02x", data)
	//log.Println("session:", len(tcp.Payload), s.ToString())
	//PrintFlow("-=->tun2Proxy", ip4, tcp)

	if _, err := SysConfig.TunWriteBack.Write(data); err != nil {
		log.Println("-=->tun2Proxy write to tun err:", err)
		return
	}
}

func (t2s *Tun2Socks) proxy2Tun(ip4 *layers.IPv4, tcp *layers.TCP) {
	//PrintFlow("<-=-proxy2Tun", ip4, tcp)
	s := t2s.GetSession(int(tcp.DstPort))

	if s == nil {
		log.Printf("No such sess:%s:%d->%s:%d",
			ip4.DstIP, tcp.DstPort, ip4.SrcIP, tcp.SrcPort)
		return
	}

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = SysConfig.TunLocalIP
	tcp.SrcPort = layers.TCPPort(s.RemotePort)
	data := ChangePacket(ip4, tcp)

	//log.Printf("After:%02x", data)
	//log.Println("session:", s.ToString())
	//PrintFlow("<-=-proxy2Tun", ip4, tcp)

	if _, err := SysConfig.TunWriteBack.Write(data); err != nil {
		log.Println("<-=-proxy2Tun write to tun err:", err)
		return
	}
}

func (t2s *Tun2Socks) ReceivePacket(ip4 *layers.IPv4, tcp *layers.TCP) {
	if int(tcp.SrcPort) == LocalProxyPort {
		t2s.proxy2Tun(ip4, tcp)
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

func (t2s *Tun2Socks) RemoveSession(key int) {
	t2s.Lock()
	defer t2s.Unlock()
	delete(t2s.SessionCache, key)
}
