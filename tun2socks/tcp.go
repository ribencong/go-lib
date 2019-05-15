package tun2socks

import (
	"fmt"
	"log"
	"math"
	"net"
	"syscall"
)

func (t2s *Tun2Socks) Transfer() {

	defer log.Println("Tcp Proxy Edn>>>>>>", t2s.innerTcpPivot.Addr())
	defer t2s.innerTcpPivot.Close()

	log.Println("Tcp Proxy start......", t2s.innerTcpPivot.Addr())

	for {
		conn, err := t2s.innerTcpPivot.Accept()
		if err != nil {
			log.Println("Accept:", err)
			return
		}
		//log.Println("Accept:", conn.RemoteAddr(), conn.LocalAddr())
		go t2s.process(conn)
	}
}

func (t2s *Tun2Socks) process(conn net.Conn) {
	leftConn := conn.(*net.TCPConn)
	leftConn.SetKeepAlive(true)

	defer leftConn.Close()

	port := leftConn.RemoteAddr().(*net.TCPAddr).Port
	s := t2s.GetSession(port)
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

			if s.ByPass {
				print("This one no need proxy:", s.ToString())
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
