package main

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/youpipe/go-youPipe/pbs"
	"io"
	"log"
	"net"
	"time"
)

type Node struct {
	ctx         context.Context
	stop        context.CancelFunc
	isOnline    bool
	localServe  net.Listener
	accessPoint string
}

func NewNode(lAddr, rAddr string) *Node {
	l, err := net.Listen("tcp", lAddr)
	if err != nil {
		logger.Printf("failed to listen on:%s : %v", lAddr, err)
		return nil
	}

	ctx, s := context.WithCancel(context.Background())

	node := &Node{
		ctx:         ctx,
		stop:        s,
		localServe:  l,
		accessPoint: rAddr,
	}

	logger.Printf("new node setup and listen success at:%s", lAddr)

	return node
}

func (node *Node) Serving() {

	node.isOnline = true

	defer func() {
		node.isOnline = false
		logger.Print("node service go routine exit")
	}()

	for {
		select {
		case <-node.ctx.Done():
			logger.Print("node service stop.....")
			return
		default:
			conn, err := node.localServe.Accept()
			if err != nil {
				logger.Printf("failed to accept :%s", err)
				continue
			}

			go node.handleConn(conn)
		}
	}
}

func (node *Node) handleConn(conn net.Conn) {
	defer conn.Close()

	conn.(*net.TCPConn).SetKeepAlive(true)

	logger.Print("a new connection :->", conn.RemoteAddr().String())

	obj, err := HandShake(conn)
	if err != nil {
		logger.Print("sock5 handshake err:->", err)
		return
	}
	logger.Print("target info:->", obj.address)

	apConn, err := node.handShake(obj, conn)
	if err != nil {
		return
	}
	defer apConn.Close()

	_, _, err = relay(conn, apConn)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		logger.Printf("relay error: %v", err)
	}
}

func (node *Node) handShake(obj *rfcObj, conn net.Conn) (net.Conn, error) {
	apConn, err := net.Dial("tcp", node.accessPoint)
	if err != nil {
		log.Print("failed to connect to remote access point server :->", node.accessPoint, err)
		return nil, err
	}
	apConn.(*net.TCPConn).SetKeepAlive(true)

	tarAddr, _ := proto.Marshal(obj.address)
	if _, err := apConn.Write(tarAddr); err != nil {
		logger.Printf("failed to send target TargetAddr:->%v, %v", tarAddr, err)
		return nil, err
	}

	log.Printf("proxy %s <-> %s <-> %s", conn.RemoteAddr(), node.accessPoint, obj.address)

	buffer := make([]byte, 1024)
	n, err := apConn.Read(buffer)
	if err != nil {
		log.Printf("failed to read ap response :->%v", err)
		return nil, err
	}
	ack := &pbs.CommAck{}
	if err = proto.Unmarshal(buffer[:n], ack); err != nil {
		log.Printf("unmarshal response :->%v", err)
		return nil, err
	}
	if ack.ErrNo != pbs.ErrorNo_Success {
		log.Printf("handshake with remote error:->%v", ack.ErrMsg)
		return nil, err
	}

	return apConn, nil
}

func relay(left, right net.Conn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
		ch <- res{n, err}
	}()

	n, err := io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}

func (node *Node) IsRunning() bool {
	return node.isOnline
}

func (node *Node) Stop() {
	node.localServe.Close()
	node.stop()
}
