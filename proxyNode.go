package main

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/youpipe/go-youPipe/account"
	"github.com/youpipe/go-youPipe/pbs"
	"github.com/youpipe/go-youPipe/service"
	"io"
	"net"
	"time"
)

type Node struct {
	ctx         context.Context
	stop        context.CancelFunc
	isOnline    bool
	localServe  net.Listener
	accessPoint string
	account     *account.Account
	proxyID     string
}

func NewNode(lAddr, rAddr string) *Node {
	l, err := net.Listen("tcp", lAddr)
	if err != nil {
		fmt.Printf("failed to listen on:%s : %v", lAddr, err)
		panic(err)
	}

	ctx, s := context.WithCancel(context.Background())

	node := &Node{
		ctx:         ctx,
		stop:        s,
		localServe:  l,
		accessPoint: rAddr,
	}

	fmt.Printf("listening at:%s\n", lAddr)

	return node
}

func (node *Node) Serving() {

	node.isOnline = true

	defer func() {
		node.isOnline = false
		fmt.Println("node service go routine exit")
	}()

	for {
		select {
		case <-node.ctx.Done():
			fmt.Println("node service stop.....")
			return
		default:
			conn, err := node.localServe.Accept()
			if err != nil {
				fmt.Printf("failed to accept :%s", err)
				panic(err)
			}

			go node.handleConn(conn)
		}
	}
}

func (node *Node) handleConn(conn net.Conn) {
	defer conn.Close()

	conn.(*net.TCPConn).SetKeepAlive(true)

	fmt.Println("a new connection :->", conn.RemoteAddr().String())

	obj, err := HandShake(conn)
	if err != nil {
		fmt.Println("sock5 handshake err:->", err)
		return
	}
	fmt.Println("target info:->", obj.target)

	apConn, err := node.findProxy(obj, conn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer apConn.Close()

	_, _, err = relay(conn, apConn)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		fmt.Printf("relay error: %v\n", err)
	}
}

func (node *Node) findProxy(obj *rfcObj, conn net.Conn) (net.Conn, error) {
	apConn, err := net.Dial("tcp", node.accessPoint)
	if err != nil {
		fmt.Printf("failed to connect to (%s) access point server (%s):->", node.accessPoint, err)
		return nil, err
	}
	apConn.(*net.TCPConn).SetKeepAlive(true)

	req := &pbs.Sock5Req{
		Address: string(node.account.Address),
		Target:  obj.target,
		IsRaw:   false,
	}
	data, _ := proto.Marshal(req)
	if _, err := apConn.Write(data); err != nil {
		fmt.Printf("failed to send target(%s) \n", obj.target)
		return nil, err
	}

	buffer := make([]byte, 1024)
	n, err := apConn.Read(buffer)
	if err != nil {
		fmt.Printf("failed to read ap response :->%v", err)
		return nil, err
	}

	ack := &pbs.Sock5Res{}
	if err = proto.Unmarshal(buffer[:n], ack); err != nil {
		fmt.Printf("unmarshal response :->%v", err)
		return nil, err
	}

	if ack.Address != node.proxyID {
		return nil, fmt.Errorf("input address(%s) and proxy id(%s) are not same", ack.Address, node.proxyID)
	}

	if req.IsRaw == false {
		var aesKey [32]byte
		if err := node.account.CreateAesKey(&aesKey, ack.Address); err != nil {
			fmt.Printf("create aes key for address(%s) err(%v)", ack.Address, err)
			return nil, err
		}
		apConn, err = service.Shadow(apConn, aesKey)
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
