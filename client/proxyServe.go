package client

import (
	"encoding/json"
	"fmt"
	"github.com/ribencong/go-youPipe/network"
	"github.com/ribencong/go-youPipe/service"
	"net"
)

func (c *Client) consume(conn net.Conn) {
	defer conn.Close()

	obj, err := ProxyHandShake(conn)
	if err != nil {
		fmt.Println("\nSock5 handshake err:->", err)
		return
	}
	fmt.Println("\nProxy handshake success:", obj.Target)

	jsonConn, err := c.connectSockServer()
	if err != nil {
		fmt.Printf("\nConnet to socks server err:%v\n", err)
		return
	}

	if err := c.pipeHandshake(jsonConn, obj.Target); err != nil {
		fmt.Printf("\nForward to server err:%v\n", err)
		return
	}

	consumeConn := service.NewConsumerConn(jsonConn.Conn, c.aesKey)
	if consumeConn == nil {
		return
	}

	pipe := NewPipe(conn, consumeConn, c.FlowCounter, obj.Target)

	fmt.Printf("\nNew pipe:%s", pipe.String())

	go pipe.collectRequest()

	pipe.pullDataFromServer()

	fmt.Printf("\n\nPipe for(%s) is closing", pipe.target)
}

func (c *Client) connectSockServer() (*service.JsonConn, error) {

	port := c.curService.ID.ToServerPort()
	addr := network.JoinHostPort(c.curService.IP, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to (%s) access point server (%s):->", addr, err)

	}
	conn.(*net.TCPConn).SetKeepAlive(true)
	return &service.JsonConn{conn}, nil
}

func (c *Client) pipeHandshake(conn *service.JsonConn, target string) error {

	reqData := &service.PipeReqData{
		Addr:   c.Address.ToString(),
		Target: target,
	}

	data, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal hand shake data err:%v", err)
	}

	sig := c.Sign(data)

	hs := &service.YPHandShake{
		CmdType: service.CmdPipe,
		Sig:     sig,
		Pipe:    reqData,
	}

	if err := conn.WriteJsonMsg(hs); err != nil {
		return fmt.Errorf("write hand shake data err:%v", err)

	}
	ack := &service.YouPipeACK{}
	if err := conn.ReadJsonMsg(ack); err != nil {
		return fmt.Errorf("failed to read miner's response :->%v", err)
	}

	if !ack.Success {
		return fmt.Errorf("hand shake to miner err:%s", ack.Message)
	}

	return nil
}
