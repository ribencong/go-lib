package pipeProxy

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"net"
)

type PipeProxy struct {
	done chan error
	*net.TCPListener
	wallet *wallet.Wallet
	TunSrc Tun2Pipe
}

func NewProxy(addr string, w *wallet.Wallet, t Tun2Pipe) (*PipeProxy, error) {
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	ap := &PipeProxy{
		done:        make(chan error),
		TCPListener: l.(*net.TCPListener),
		wallet:      w,
		TunSrc:      t,
	}
	return ap, nil
}

func (pp *PipeProxy) Proxying() {

	go pp.Accepting()

	defer pp.releaseResource()

	select {
	case err := <-pp.done:
		fmt.Printf("Proxy finished for pipe proxy reason:%v", err)
	case err := <-pp.wallet.Done:
		fmt.Printf("Proxy finished for wallet err:%v", err)
	}
}

func (pp *PipeProxy) Accepting() {
	fmt.Println("Proxy start working at:", pp.Addr().String())
	defer fmt.Println("Proxy exit......")

	for {
		conn, err := pp.Accept()
		if err != nil {
			fmt.Printf("\nFinish to proxy system request :%s", err)
			return
		}

		fmt.Println("\nNew proxy request:", conn.RemoteAddr().String())
		conn.(*net.TCPConn).SetKeepAlive(true)
		go pp.consume(conn)
	}
}

func (pp *PipeProxy) consume(conn net.Conn) {
	defer conn.Close()

	tgtAddr := pp.TunSrc.GetTarget(conn)

	if len(tgtAddr) < 10 {
		fmt.Println("\nNo such connection's target address:->", conn.RemoteAddr().String())
		return
	}
	fmt.Println("\n Proxying target address:", tgtAddr)

	//TODO::match PAC file in ios or android logic
	pipe := pp.wallet.SetupPipe(conn, tgtAddr)
	if nil == pipe {
		fmt.Println("Create pipe failed:", tgtAddr)
		return
	}

	pipe.PullDataFromServer()

	fmt.Printf("\n\nPipe for(%s) is closing", tgtAddr)
}

func (pp *PipeProxy) Finish() {
	pp.done <- fmt.Errorf("closed by outer controller")
}

func (pp *PipeProxy) releaseResource() {

	if pp.TCPListener == nil {
		return
	}

	pp.TCPListener.Close()
	pp.TCPListener = nil
	pp.wallet.Finish()
	pp.TunSrc.Finish() //TODO:: how to inter action with tun2proxy
}
