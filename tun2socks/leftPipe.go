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
	//buf := make([]byte, math.MaxInt16)
	//for {
	//	n, e := pipe.leftConn.Read(buf)
	//	if e != nil{
	//		log.Println("left read err:", e)
	//		return
	//	}
	//	n2, e := pipe.rightConn.Write(buf[:n])
	//	if e != nil{
	//		log.Println("right write err:", e)
	//		return
	//	}
	//
	//	log.Printf("left read:%d, right write:%d", n, n2)
	//}

	n, e := io.Copy(pipe.rightConn, pipe.leftConn)

	log.Println("Copy from local to remote finished:", n, e)
}

func (pipe LeftPipe) Right2Left() {

	defer log.Println("Right2Left pipe work exit:", pipe.leftConn.LocalAddr(), pipe.leftConn.RemoteAddr())
	defer pipe.leftConn.Close()

	//buf := make([]byte, math.MaxInt16)
	//
	//for{
	//	n, e := pipe.rightConn.Read(buf)
	//	if e != nil{
	//		log.Println("right read err:", e)
	//		return
	//	}
	//
	//	n2, e := pipe.leftConn.Write(buf[:n])
	//	if e != nil{
	//		log.Println("right write err:", e)
	//		return
	//	}
	//	log.Printf("left read:%d, right write:%d", n, n2)
	//}

	n, e := io.Copy(pipe.leftConn, pipe.rightConn)

	log.Println("Copy from server to local finished:", n, e)
}
