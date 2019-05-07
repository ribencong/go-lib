package tun2socks

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Session struct {
	NeedProxy  bool
	hostName   string
	PacketSent int
	BytesSent  int
	UPTime     time.Time
	RemoteIP   net.IP
	RemotePort int
}

func (s *Session) ToString() string {
	return fmt.Sprintf("(%s:%t)%s:%d p(%d) b(%d) t=%s",
		s.hostName, s.NeedProxy, s.RemoteIP, s.RemotePort,
		s.PacketSent, s.BytesSent, s.UPTime.Format("2006-01-02 15:04:05"))
}

func newSession(ip4 *layers.IPv4, tcp *layers.TCP) *Session {
	s := &Session{
		UPTime:     time.Now(),
		RemoteIP:   ip4.DstIP,
		RemotePort: int(tcp.DstPort),
		NeedProxy:  SysConfig.NeedProxy(ip4.DstIP.String()),
	}
	return s
}

func (s *Session) SetHostName(host string) {
	s.hostName = host
	if !s.NeedProxy {
		s.NeedProxy = SysConfig.NeedProxy(host)
	}
}

type TcpProxy struct {
	sync.RWMutex
	tcpTransfer  *net.TCPListener
	socks5Addr   string
	protect      ConnProtect
	VpnWriteBack io.WriteCloser
	SessionCache map[int]*Session
}

func NewTcpProxy(socksAddr string, protect ConnProtect, writer io.WriteCloser) (*TcpProxy, error) {
	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		Port: 51414,
	})
	if e != nil {
		return nil, e
	}

	p := &TcpProxy{
		tcpTransfer:  l,
		socks5Addr:   socksAddr,
		protect:      protect,
		VpnWriteBack: writer,
		SessionCache: make(map[int]*Session),
	}

	go p.Transfer()

	return p, nil
}
func (p *TcpProxy) Transfer() {

	defer p.tcpTransfer.Close()

	log.Println("Tcp Transfer start......", p.tcpTransfer.Addr())

	buf := make([]byte, 1024)

	for {
		conn, err := p.tcpTransfer.Accept()
		if err != nil {
			log.Println("Accept:", err)
			return
		}
		log.Println("Accept:", conn.RemoteAddr(), conn.LocalAddr())

		n, e := conn.Read(buf)
		if e != nil {
			log.Println("Read:", n, buf[:n])
			continue
		}

		if _, e := conn.Write(buf); e != nil {
			log.Println("Write:", e)
		}
	}
}

func (p *TcpProxy) GetSession(key int) *Session {
	p.RLock()
	defer p.RUnlock()
	return p.SessionCache[key]
}
func (p *TcpProxy) AddSession(portKey int, s *Session) {
	p.Lock()
	defer p.Unlock()
	p.SessionCache[portKey] = s
}

func (p *TcpProxy) RemoveSession(key int) {
	p.Lock()
	defer p.Unlock()
	delete(p.SessionCache, key)
}

var debug = true

func (p *TcpProxy) ReceivePacket(ip4 *layers.IPv4, tcp *layers.TCP) {

	if int(tcp.SrcPort) == 51414 {
		log.Println("Step 1 Success")
		return
	}

	log.Printf("	TCP (%s:%d)->(%s:%d) (SYN(%t)FIN(%t)"+
		", RST(%t), PSH(%t))\n",
		ip4.SrcIP, tcp.SrcPort,
		ip4.DstIP, tcp.DstPort, tcp.SYN, tcp.FIN, tcp.RST, tcp.PSH)
	s := p.GetSession(int(tcp.SrcPort))
	if s == nil {
		s = newSession(ip4, tcp)
		p.AddSession(int(tcp.SrcPort), s)
	}
	s.PacketSent++

	//if len(sess.hostName) == 0 && len(tcp.Payload) > 10 {
	//	buf := bufio.NewReader(bytes.NewReader(tcp.Payload))
	//	if req, err := http.ReadRequest(buf); err == nil {
	//		log.Println("Host:->", req.Host)
	//		sess.SetHostName(req.Host)
	//	}
	//}

	ip4.SrcIP = ip4.DstIP
	ip4.DstIP = net.ParseIP("10.8.0.2")
	tcp.DstPort = 51414
	tcp.SetNetworkLayerForChecksum(ip4)

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, tcp); err != nil {
		log.Println("Wrap Tcp to ip packet  err:", err)
		return
	}

	data := b.Bytes()
	log.Printf("After:%02x", data)

	n, err := p.VpnWriteBack.Write(data)
	if err != nil {
		log.Println("tcp write to proxy err:", err)
		return
	}

	if debug {
		p := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.NoCopy)

		if ipv4Layer := p.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
			ipv4, _ := ipv4Layer.(*layers.IPv4)
			log.Printf("IP %s->%s\n", ipv4.SrcIP, ipv4.DstIP)
		}

		if tcpLayer := p.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			log.Printf("Port: %d->%d\n", tcp.SrcPort, tcp.DstPort)
		}
	}

	log.Println(" write back to vpn success:", n, len(data))
	log.Println("\n\n=====================================================================================================================")
}
