package wallet

import (
	"encoding/json"
	"fmt"
	"github.com/ribencong/go-youPipe/network"
	"github.com/ribencong/go-youPipe/service"
	"log"
	"net"
	"time"
)

type LeftPipe struct {
	payCounter  *FlowCounter
	target      string
	requestBuf  []byte
	responseBuf []byte
	proxyConn   net.Conn
	consume     *service.PipeConn
}

func NewPipe(l net.Conn, r *service.PipeConn, pay *FlowCounter, tgt string) *LeftPipe {
	return &LeftPipe{
		target:      tgt,
		requestBuf:  make([]byte, service.BuffSize),
		responseBuf: make([]byte, service.BuffSize),
		proxyConn:   l,
		consume:     r,
		payCounter:  pay,
	}
}

func (p *LeftPipe) collectRequest() {
	defer p.expire()
	defer fmt.Println("collect system proxy conn exit......")
	for {
		nr, err := p.proxyConn.Read(p.requestBuf)
		if nr > 0 {
			if nw, errW := p.consume.WriteCryptData(p.requestBuf[:nr]); errW != nil {
				fmt.Printf("\n forward system proxy err:%d, %v", nw, errW)
				return
			}
		}
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (p *LeftPipe) PullDataFromServer() {
	defer p.expire()
	defer fmt.Println("consume conn failed......")
	for {
		n, err := p.consume.ReadCryptData(p.responseBuf)

		//fmt.Printf("\n\n Wallet Left pipe Pull data(no:%d, err:%v) for(%s) from(%s)\n", n, err,
		//	p.target, p.consume.RemoteAddr().String())
		if n > 0 {
			if nw, errW := p.proxyConn.Write(p.responseBuf[:n]); errW != nil {
				fmt.Printf("\n Wallet Left pipe write data to system proxy err:%d, %v\n", nw, errW)
				return
			}
		}

		if err != nil {
			log.Println(err)
			return
		}

		if p.payCounter.Closed {
			return
		}

		p.payCounter.Consume(n)
	}
}

func (p *LeftPipe) expire() {
	p.consume.SetDeadline(time.Now())
	p.proxyConn.SetDeadline(time.Now())
}

func (p *LeftPipe) String() string {
	return fmt.Sprintf("%s<->%s",
		p.proxyConn.RemoteAddr().String(),
		p.consume.RemoteAddr().String())
}

func (w *Wallet) SetupPipe(lConn net.Conn, tgtAddr string) *LeftPipe {
	jsonConn, err := w.connectSockServer()
	if err != nil {
		fmt.Printf("\nConnet to socks server err:%v\n", err)
		return nil
	}

	if err := w.pipeHandshake(jsonConn, tgtAddr); err != nil {
		fmt.Printf("\nForward to server err:%v\n", err)
		return nil
	}

	consumeConn := service.NewConsumerConn(jsonConn.Conn, w.aesKey)
	if consumeConn == nil {
		return nil
	}

	pipe := NewPipe(lConn, consumeConn, w.FlowCounter, tgtAddr)

	fmt.Printf("\nNew pipe:%s", pipe.String())

	go pipe.collectRequest()

	return pipe
}

func (w *Wallet) connectSockServer() (*service.JsonConn, error) {
	fmt.Printf("\nconnectSockServer Wallet socks ID:%s, IP:%s ", w.curService.ID, w.curService.IP)
	port := w.curService.ID.ToServerPort()
	addr := network.JoinHostPort(w.curService.IP, port)

	conn, err := w.getOuterConn(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to (%s) access point server (%s):->", addr, err)
	}
	conn.(*net.TCPConn).SetKeepAlive(true)
	return &service.JsonConn{conn}, nil
}

func (w *Wallet) pipeHandshake(conn *service.JsonConn, target string) error {

	reqData := &service.PipeReqData{
		Addr:   w.Address.ToString(),
		Target: target,
	}

	data, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal hand shake data err:%v", err)
	}

	sig := w.Sign(data)

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
