package tun2Pipe

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"io"
	"net"
	"time"
)

type Session struct {
	byPass     bool
	Pipe       *directPipe
	UPTime     time.Time
	RemoteIP   net.IP
	RemotePort int
	ServerPort int
	byteSent   int
	packetSent int
	HostName   string
}

func (s *Session) ToString() string {
	return fmt.Sprintf("(srvPort=%d, bypass=%t) %s:%d t=%s", s.ServerPort, s.byPass,
		s.RemoteIP, s.RemotePort,
		s.UPTime.Format("2006-01-02 15:04:05"))
}

func newSession(ip4 *layers.IPv4, tcp *layers.TCP, srvPort int, bp bool) *Session {
	s := &Session{
		UPTime:     time.Now(),
		RemoteIP:   ip4.DstIP,
		RemotePort: int(tcp.DstPort),
		ServerPort: srvPort,
		byPass:     bp,
	}

	VpnInstance.Log(fmt.Sprintln("New Tcp Session:", s.ToString()))
	return s
}

type directPipe struct {
	Left  *net.TCPConn
	Right *net.TCPConn
}

func (dp *directPipe) readingIn() {
	defer dp.Right.Close()

	if _, err := io.Copy(dp.Left, dp.Right); err != nil {
		VpnInstance.Log(fmt.Sprintln("Tun Proxy pipe right 2 left finished:", err))
		dp.expire()
	}
}

func (dp *directPipe) writeOut(buf []byte) error {
	if _, e := dp.Right.Write(buf); e != nil {
		VpnInstance.Log(fmt.Sprintln("Tun Proxy pipe left 2 right err:", e))
		dp.expire()
		return e
	}
	return nil
}

func (dp *directPipe) expire() {
	dp.Left.SetDeadline(time.Now())
	dp.Right.SetDeadline(time.Now())
}
