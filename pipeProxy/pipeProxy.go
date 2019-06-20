package pipeProxy

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"net"
	"strconv"
)

type PipeProxy struct {
	*net.TCPListener
	Wallet *wallet.Wallet
	TunSrc Tun2Pipe
}

func NewProxy(addr string, w *wallet.Wallet, t Tun2Pipe) (*PipeProxy, error) {
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	ap := &PipeProxy{
		TCPListener: l.(*net.TCPListener),
		Wallet:      w,
		TunSrc:      t,
	}
	return ap, nil
}

func (pp *PipeProxy) Proxying() {

	done := make(chan error)

	go pp.Accepting(done)
	go pp.Wallet.Running(done)
	go pp.TunSrc.Proxying(done)

	select {
	case err := <-done:
		fmt.Printf("PipeProxy exit for:%s", err.Error())
	}

	pp.Finish()
}

func (pp *PipeProxy) Accepting(done chan error) {

	fmt.Println("Proxy start working at:", pp.Addr().String())
	defer fmt.Println("Proxy exit......")

	for {
		conn, err := pp.Accept()
		if err != nil {
			fmt.Printf("\nFinish to proxy system request :%s", err)
			done <- err
			return
		}

		conn.(*net.TCPConn).SetKeepAlive(true)

		go pp.consume(conn)

		select {
		case err := <-done:
			fmt.Printf("\nProxy closed by out controller:%s", err.Error())
		default:
		}
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
	pipe := pp.Wallet.SetupPipe(conn, tgtAddr)
	if nil == pipe {
		fmt.Println("Create pipe failed:", tgtAddr)
		return
	}

	pipe.PullDataFromServer()

	rAddr := conn.RemoteAddr().String()
	_, port, _ := net.SplitHostPort(rAddr)
	keyPort, _ := strconv.Atoi(port)
	pp.TunSrc.RemoveFromSession(keyPort)

	//TODO::need to make sure this is ok
	fmt.Printf("\n\nPipe(%s) for(%s) is closing", rAddr, tgtAddr)
}

func (pp *PipeProxy) Finish() {

	if pp.TCPListener != nil {
		pp.TCPListener.Close()
		pp.TCPListener = nil
	}

	if pp.Wallet != nil {
		pp.Wallet.Finish()
		pp.Wallet = nil
	}

	if pp.TunSrc != nil {
		pp.TunSrc.Finish()
		pp.TunSrc = nil
	}
}
