package main

import (
	"encoding/json"
	"fmt"
	"github.com/youpipe/go-youPipe/account"
	"github.com/youpipe/go-youPipe/service"
	"io"
	"net"
	"time"
)

type Node struct {
	isOnline    bool
	localServe  net.Listener
	accessPoint string
	account     *account.Account
	proxyID     string
	license     *service.License
}

func NewNode(lAddr, rAddr string) *Node {
	l, err := net.Listen("tcp", lAddr)
	if err != nil {
		fmt.Printf("failed to listen on:%s : %v", lAddr, err)
		return nil
	}

	node := &Node{
		localServe:  l,
		accessPoint: rAddr,
	}

	fmt.Printf("\n------>>>listening at:%s\n", lAddr)

	return node
}

func (node *Node) Serving() {

	node.isOnline = true

	defer func() {
		node.isOnline = false
		fmt.Println("node service go routine exit")
	}()

	for {
		conn, err := node.localServe.Accept()
		if err != nil {
			fmt.Printf("finish to accept :%s", err)
			return
		}

		go node.handleConn(conn)
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

	apConn, err := node.findProxy(obj)
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

func (node *Node) findProxy(obj *rfcObj) (net.Conn, error) {
	c, err := net.Dial("tcp", node.accessPoint)
	if err != nil {
		fmt.Printf("failed to connect to (%s) access point server (%s):->", node.accessPoint, err)
		return nil, err
	}
	c.(*net.TCPConn).SetKeepAlive(true)
	apConn := &service.CtrlConn{Conn: c}

	req := service.NewHandReq(string(node.account.Address), obj.target, unlockedAcc.Key.PriKey)

	data, _ := json.Marshal(req)
	if _, err := apConn.Write(data); err != nil {
		fmt.Printf("failed to send target(%s) \n", obj.target)
		return nil, err
	}

	ack := &service.ACK{}
	if err := apConn.ReadMsg(ack); err != nil {
		fmt.Printf("failed to read ap response :->%v", err)
		return nil, err
	}

	var aesKey [32]byte
	if err := node.account.CreateAesKey(&aesKey, node.proxyID); err != nil {
		fmt.Printf("create aes key for address(%s) err(%v)", node.proxyID, err)
		return nil, err
	}

	sConn, err := service.Shadow(c, aesKey)
	return sConn, nil
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
}
