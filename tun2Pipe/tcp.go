package tun2Pipe

import (
	"fmt"
	"io"
	"math"
	"net"
	"syscall"
)

func (t2s *Tun2Pipe) Pivoting() {

	defer VpnInstance.Log(fmt.Sprintln("TunTcp Proxy Edn>>>>>>", t2s.innerTcpPivot.Addr()))

	VpnInstance.Log(fmt.Sprintln("TunTcp Proxy start......", t2s.innerTcpPivot.Addr()))

	for {
		conn, err := t2s.innerTcpPivot.Accept()
		if err != nil {
			VpnInstance.Log(fmt.Sprintln("Accept:", err))
			t2s.Done <- err
			return
		}
		//log.Println("Accept:", conn.RemoteAddr(), conn.LocalAddr())
		go t2s.process(conn)

		select {
		case err := <-t2s.Done:
			VpnInstance.Log(fmt.Sprintln("Tun2Proxy pivoting thread exit:", err))
			t2s.releaseResource()
			return
		default:
			continue
		}
	}
}

func (t2s *Tun2Pipe) releaseResource() {
	t2s.Lock()
	defer t2s.Unlock()
	if t2s.innerTcpPivot == nil {
		return
	}

	t2s.innerTcpPivot.Close()
	t2s.innerTcpPivot = nil
	//TODO::check other resources
}

func (t2s *Tun2Pipe) process(conn net.Conn) {
	leftConn := conn.(*net.TCPConn)
	leftConn.SetKeepAlive(true)

	defer leftConn.Close()

	port := leftConn.RemoteAddr().(*net.TCPAddr).Port
	s := t2s.GetSession(port)
	if s == nil {
		VpnInstance.Log(fmt.Sprintln("Can't proxy this one:", leftConn.RemoteAddr()))
		return
	}

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

			s.Pipe = &ProxyPipe{
				Left:  leftConn,
				Right: rightConn,
			}
			go s.Pipe.Right2Left()
		}

		s.Pipe.WriteTunnel(buff[:n])
	}

}
