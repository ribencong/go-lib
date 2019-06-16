package pipeProxy

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"log"
	"net"
)

type PipeProxy struct {
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
		TCPListener: l.(*net.TCPListener),
		wallet:      w,
		TunSrc:      t,
	}
	return ap, nil
}

func (pp *PipeProxy) Proxying() {

	log.Println("Proxy start working at:", pp.Addr().String())
	defer pp.Close()
	defer log.Println("Proxy exit......")

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

//TODO::how to finish the proxy?
func (pp *PipeProxy) Finish() {
	pp.TCPListener.Close()
	pp.wallet.Finish()
	pp.TunSrc.Finish()
}
