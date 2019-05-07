package tun2socks

import (
	"io"
	"log"
	"net"
	"syscall"
	"time"
)

type LeftPipe struct {
	*TcpProxy
	tgtAddr   string
	leftConn  *net.TCPConn
	rightConn *net.TCPConn
}

func (pipe LeftPipe) Working() {

	d := &net.Dialer{
		Timeout: time.Second * 2,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(pipe.protect)
		},
	}

	c, e := d.Dial("tcp", pipe.tgtAddr)
	if e != nil {
		log.Println("Dial remote err:", e)
		return
	}

	pipe.rightConn = c.(*net.TCPConn)

	go pipe.Waiting()

	n, e := io.Copy(pipe.rightConn, pipe.leftConn)
	log.Println("Copy from local to remote finished:", n, e)
}

func (pipe LeftPipe) Waiting() {

	n, e := io.Copy(pipe.leftConn, pipe.rightConn)
	log.Println("Copy from server to client finished:", n, e)
}
