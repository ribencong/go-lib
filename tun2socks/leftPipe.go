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

func (pipe LeftPipe) Left2Right() {
	defer log.Println("Left2Right pipe work exit:", pipe.leftConn.LocalAddr(), pipe.leftConn.RemoteAddr())
	defer pipe.leftConn.Close()

	n, e := io.Copy(pipe.rightConn, pipe.leftConn)
	log.Println("Copy from local to remote finished:", n, e)
}

func (pipe LeftPipe) Right2Left() {

	defer log.Println("Right2Left pipe work exit:", pipe.leftConn.LocalAddr(), pipe.leftConn.RemoteAddr())
	defer pipe.leftConn.Close()

	n, e := io.Copy(pipe.leftConn, pipe.rightConn)
	log.Println("Copy from server to local finished:", n, e)
}
