package tun2Pipe

import (
	"fmt"
	"io"
	"math"
	"net"
	"syscall"
)

func (t2s *Tun2Pipe) simpleForward(conn net.Conn) {

	leftConn := conn.(*net.TCPConn)
	leftConn.SetKeepAlive(true)

	port := leftConn.RemoteAddr().(*net.TCPAddr).Port
	s := t2s.GetSession(port)
	if s == nil {
		VpnInstance.Log(fmt.Sprintln("Can't proxy this one:", leftConn.RemoteAddr()))
		return
	}

	defer t2s.RemoveSession(port)
	defer leftConn.Close()

	VpnInstance.Log(fmt.Sprintln("Tun New conn for tcp session:", s.ToString()))

	tgtAddr := fmt.Sprintf("%s:%d", s.RemoteIP, s.RemotePort)
	buff := make([]byte, math.MaxInt16)

	for {
		n, e := leftConn.Read(buff)
		if e != nil {
			if e != io.EOF {
				VpnInstance.Log(fmt.Sprintln("Tun Read from left conn err:", e))
			}
			return
		}

		if s.Pipe == nil {

			d := &net.Dialer{
				Timeout: SysDialTimeOut,
				Control: func(network, address string, c syscall.RawConn) error {
					return c.Control(Protector)
				},
			}
			c, e := d.Dial("tcp", tgtAddr)
			if e != nil {
				VpnInstance.Log(fmt.Sprintln("Dial remote err:", e))
				return
			}
			rightConn := c.(*net.TCPConn)
			rightConn.SetKeepAlive(true)
			VpnInstance.Log(fmt.Sprintf("TCP:Tun Pipe dial success: %s->%s:", rightConn.LocalAddr(), s.ToString()))

			s.Pipe = &directPipe{
				Left:  leftConn,
				Right: rightConn,
			}
			go s.Pipe.readingIn()
		}

		if err := s.Pipe.writeOut(buff[:n]); err != nil {
			return
		}
	}

}
