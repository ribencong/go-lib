package tun2socks

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
)

type Session struct {
	ID        string
	NeedProxy bool
	conn      *net.TCPConn
	hostName  string
	Src       *net.TCPAddr
	Dst       *net.TCPAddr
}

func genSSID(srcIP, dstIP net.IP, srcPort, dstPort layers.TCPPort) string {
	return fmt.Sprintf("%s:%d->%s:%d", srcIP, srcPort, dstIP, dstPort)
}

func newSession(ip4 *layers.IPv4, tcp *layers.TCP, socks5Addr string, protect ConnProtect) (*Session, error) {
	s := &Session{
		Src: &net.TCPAddr{
			IP:   ip4.SrcIP,
			Port: int(tcp.SrcPort),
		},
		Dst: &net.TCPAddr{
			IP:   ip4.DstIP,
			Port: int(tcp.DstPort),
		},
	}
	s.NeedProxy = SysConfig.NeedProxy(ip4.DstIP.String())

	if s.NeedProxy {
		dialer, err := proxy.SOCKS5("tcp", socks5Addr, nil, nil)
		if err != nil {
			log.Println("connect to local socks server err:", err)
			return nil, err
		}

		c, err := dialer.Dial("tcp", s.Dst.String())
		if err != nil {
			log.Println("Dial to local socks server err:", err)
			return nil, err
		}
		s.conn = c.(*net.TCPConn)
	} else {
		c, err := net.DialTCP("tcp", nil, s.Dst)
		if err != nil {
			log.Println("Create direct conn for tcp err:", err)
			return nil, err
		}
		s.conn = c
	}

	rawConn, err := s.conn.SyscallConn()
	if err != nil {
		log.Println("tcp session SyscallConn err:", s.ID, err)
		return nil, err
	}

	if err := rawConn.Control(protect); err != nil {
		log.Println("tcp session Protect err:", s.ID, err)
		return nil, err
	}

	return s, nil
}

func (s *Session) SetHostName(host string) {
	s.hostName = host
	if !s.NeedProxy {
		s.NeedProxy = SysConfig.NeedProxy(host)
	}
}

type TcpProxy struct {
	sync.RWMutex
	tcpTrans     *net.TCPListener
	socks5Addr   string
	protect      ConnProtect
	VpnWriteBack io.WriteCloser
	SessionCache map[string]*Session
}

func NewTcpProxy(socksAddr string, protect ConnProtect, writer io.WriteCloser) (*TcpProxy, error) {
	conn, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP: net.ParseIP("127.0.0.1"),
	})

	if err != nil {
		return nil, err
	}
	p := &TcpProxy{
		tcpTrans:     conn,
		socks5Addr:   socksAddr,
		protect:      protect,
		VpnWriteBack: writer,
		SessionCache: make(map[string]*Session),
	}

	return p, nil
}

func (p *TcpProxy) ReceivePacket(ip4 *layers.IPv4, tcp *layers.TCP) {
	id := genSSID(ip4.SrcIP, ip4.DstIP, tcp.SrcPort, tcp.DstPort)
	p.Lock()
	defer p.Unlock()

	sess := p.SessionCache[id]
	if sess == nil {

		s, e := newSession(ip4, tcp, p.socks5Addr, p.protect)
		if e != nil {
			return
		}
		p.SessionCache[id] = s
		go p.ReadTcpResponse(s)
		sess = s
	}

	if len(sess.hostName) == 0 && len(tcp.Payload) > 10 {

		buf := bufio.NewReader(bytes.NewReader(tcp.Payload))
		if req, err := http.ReadRequest(buf); err == nil {
			log.Println("Host:->", req.Host)
			sess.SetHostName(req.Host)
		}
	}

	if _, e := sess.conn.Write(tcp.Payload); e != nil {
		log.Println("Forward data to local socks server err:", e)
	}
}

func (p *TcpProxy) ReadTcpResponse(s *Session) {
	buff := make([]byte, math.MaxInt16)

	for {
		n, e := s.conn.Read(buff)
		if e != nil {
			log.Println("session read error", s.ID, e)
			return
		}

		data := WrapIPPacketForTcp(s.Dst.IP, s.Src.IP, layers.TCPPort(s.Dst.Port),
			layers.TCPPort(s.Src.Port), buff[:n])
		if data == nil {
			continue
		}

		n2, err := p.VpnWriteBack.Write(data)
		if err != nil {
			log.Println("Tcp data write to Tun err:", err)
			continue
		}

		log.Printf("(%d)Tcp Vpn (%d)writer(%d) success:", n, len(data), n2)
	}
}
