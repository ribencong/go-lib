package tun2socks

import (
	"io"
	"log"
	"net"
)

type LeftPipe struct {
	*TcpProxy
	tgtAddr   string
	leftConn  *net.TCPConn
	rightConn *net.TCPConn
}

func (pipe LeftPipe) Working() {
	buff := make([]byte, 1024)
	defer log.Println("Left pipe work exit:", pipe.leftConn.LocalAddr(), pipe.leftConn.RemoteAddr())
	defer pipe.leftConn.Close()

	for {
		n, e := pipe.leftConn.Read(buff)
		if e != nil {
			log.Println("Inner socks Read err:", e)
			return
		}

		log.Println("Success Step 1:", n, buff[:n])
	}

	//d := &net.Dialer{
	//	Timeout: time.Second * 2,
	//	Control: func(network, address string, c syscall.RawConn) error {
	//		return c.Control(pipe.protect)
	//	},
	//}
	//
	//c, e := d.Dial("tcp", pipe.tgtAddr)
	//if e != nil {
	//	log.Println("Dial remote err:", e)
	//	return
	//}
	//log.Printf("pipe dial success: %s->%s:", c.LocalAddr(), c.RemoteAddr())
	//pipe.rightConn = c.(*net.TCPConn)
	//
	//go pipe.Waiting()
	//
	//n, e := io.Copy(pipe.rightConn, pipe.leftConn)
	//log.Println("Copy from local to remote finished:", n, e)
}

func (pipe LeftPipe) Waiting() {

	n, e := io.Copy(pipe.leftConn, pipe.rightConn)
	log.Println("Copy from server to client finished:", n, e)
}
