package tun2socks

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"
)

type ProxyPipe struct {
	Left  *net.TCPConn
	Right *net.TCPConn
}

func (pp *ProxyPipe) Right2Left() {
	no, err := io.Copy(pp.Left, pp.Right)
	log.Println("Proxy pipe right 2 left finished:", no, err)
}

func (pp *ProxyPipe) WriteTunnel(buf []byte) {
	if _, e := pp.Right.Write(buf); e != nil {
		log.Println("Proxy pipe left 2 right err:", e)
		pp.Close()
	}
}

func (pp *ProxyPipe) Close() {

}

type Session struct {
	NeedProxy  bool
	Pipe       *ProxyPipe
	hostName   string
	UPTime     time.Time
	RemoteIP   net.IP
	RemotePort int
}

func (s *Session) ToString() string {
	return fmt.Sprintf("(%s:%t)%s:%d t=%s",
		s.hostName, s.NeedProxy, s.RemoteIP, s.RemotePort,
		s.UPTime.Format("2006-01-02 15:04:05"))
}

func newSession(ip4 *layers.IPv4, tcp *layers.TCP) *Session {
	s := &Session{
		UPTime:     time.Now(),
		RemoteIP:   ip4.DstIP,
		RemotePort: int(tcp.DstPort),
	}
	return s
}

type TcpProxy struct {
	sync.RWMutex
	tcpTransfer  *net.TCPListener
	SessionCache map[int]*Session
	RightConn    map[string]*net.TCPConn
}

func NewTcpProxy() (*TcpProxy, error) {
	l, e := net.ListenTCP("tcp", &net.TCPAddr{
		Port: LocalProxyPort,
	})
	if e != nil {
		return nil, e
	}

	p := &TcpProxy{
		tcpTransfer:  l,
		SessionCache: make(map[int]*Session),
		RightConn:    make(map[string]*net.TCPConn),
	}

	go p.Transfer()

	return p, nil
}

func (p *TcpProxy) Transfer() {

	defer log.Println("Tcp Proxy Edn>>>>>>", p.tcpTransfer.Addr())
	defer p.tcpTransfer.Close()

	log.Println("Tcp Proxy start......", p.tcpTransfer.Addr())

	for {
		conn, err := p.tcpTransfer.Accept()
		if err != nil {
			log.Println("Accept:", err)
			return
		}
		log.Println("Accept:", conn.RemoteAddr(), conn.LocalAddr())
		go p.process(conn)
	}
}

func (p *TcpProxy) process(conn net.Conn) {
	leftConn := conn.(*net.TCPConn)
	leftConn.SetKeepAlive(true)

	defer leftConn.Close()

	port := leftConn.RemoteAddr().(*net.TCPAddr).Port
	s := p.GetSession(port)
	if s == nil {
		log.Println("Can't proxy this one:", leftConn.RemoteAddr())
		return
	}

	log.Println("New conn for session:", s.ToString())

	tgtAddr := fmt.Sprintf("%s:%d", s.RemoteIP, s.RemotePort)
	buff := make([]byte, math.MaxInt16)

	for {
		n, e := leftConn.Read(buff)
		if e != nil {
			log.Println("read from left conn err:", e)
			return
		}

		if s.Pipe == nil {

			if s.NeedProxy {
				print("This one need proxy:", s.ToString())
			}

			d := &net.Dialer{
				Timeout: SysDialTimeOut,
				Control: func(network, address string, c syscall.RawConn) error {
					return c.Control(SysConfig.Protector)
				},
			}
			c, e := d.Dial("tcp", tgtAddr)
			if e != nil {
				log.Println("Dial remote err:", e)
				return
			}
			rightConn := c.(*net.TCPConn)
			rightConn.SetKeepAlive(true)
			log.Printf("Pipe dial success: %s->%s:", rightConn.LocalAddr(), s.ToString())

			s.Pipe = &ProxyPipe{
				Left:  leftConn,
				Right: rightConn,
			}
			go s.Pipe.Right2Left()
		}

		s.Pipe.WriteTunnel(buff[:n])
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

func (p *TcpProxy) tun2Proxy(ip4 *layers.IPv4, tcp *layers.TCP) {

	//PrintFlow("-=->tun2Proxy", ip4, tcp)

	s := p.GetSession(int(tcp.SrcPort))
	if s == nil {
		s = newSession(ip4, tcp)
		p.AddSession(int(tcp.SrcPort), s)
	}

	if len(s.hostName) == 0 && len(tcp.Payload) > 10 {
		buf := bufio.NewReader(bytes.NewReader(tcp.Payload))
		if req, err := http.ReadRequest(buf); err == nil {
			s.hostName = req.Host
			s.NeedProxy = SysConfig.NeedProxy(s.hostName)
			log.Printf("Host:[%s]->%t", req.Host, s.NeedProxy)
		}
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

func (p *TcpProxy) proxy2Tun(ip4 *layers.IPv4, tcp *layers.TCP) {
	//PrintFlow("<-=-proxy2Tun", ip4, tcp)
	s := p.GetSession(int(tcp.DstPort))

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

func (p *TcpProxy) ReceivePacket(ip4 *layers.IPv4, tcp *layers.TCP) {
	if int(tcp.SrcPort) == LocalProxyPort {
		p.proxy2Tun(ip4, tcp)
		return
	}
	p.tun2Proxy(ip4, tcp)
}
