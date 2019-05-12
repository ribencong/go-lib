package client

import (
	"fmt"
	"github.com/ribencong/go-youPipe/service"
	"net"
	"time"
)

type LeftPipe struct {
	*FlowCounter
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
		FlowCounter: pay,
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
			return
		}
	}
}

func (p *LeftPipe) pullDataFromServer() {
	defer p.expire()
	defer fmt.Println("consume conn failed......")
	for {
		n, err := p.consume.ReadCryptData(p.responseBuf)

		fmt.Printf("\n\n Pull data(no:%d, err:%v) for(%s) from(%s)\n", n, err,
			p.target, p.consume.RemoteAddr().String())
		if n > 0 {
			if nw, errW := p.proxyConn.Write(p.responseBuf[:n]); errW != nil {
				fmt.Printf("\nwrite data to system proxy err:%d, %v\n", nw, errW)
				return
			}
		}

		if err != nil {
			return
		}

		if p.closed {
			return
		}

		p.FlowCounter.Consume(n)
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
