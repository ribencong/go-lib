package pipeProxy

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"log"
	"net"
	"strconv"
)

type PipeProxy struct {
	runFlag    bool
	serverIP   string
	serverPort int
	*net.TCPListener
	wallet *wallet.Wallet
	tunSrc Tun2Pipe
}

func NewProxy(addr string, w *wallet.Wallet, t Tun2Pipe) (*PipeProxy, error) {
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	ip, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	ap := &PipeProxy{
		serverIP:    ip,
		serverPort:  p,
		TCPListener: l.(*net.TCPListener),
		wallet:      w,
		tunSrc:      t,
	}

	go ap.Proxying()
	return ap, nil
}

func (mp *PipeProxy) GetServeIP() string {
	return mp.serverIP
}

func (mp *PipeProxy) GetServePort() int {
	return mp.serverPort
}

func (mp *PipeProxy) Proxying() {

	log.Println("Mac proxy start working at:", mp.Addr().String())
	defer mp.Close()
	defer log.Println("Mac proxy exit")

	for {
		conn, err := mp.Accept()
		if err != nil {
			fmt.Printf("\nFinish to proxy system request :%s", err)
			return
		}

		fmt.Println("\nNew system proxy request:", conn.RemoteAddr().String())
		conn.(*net.TCPConn).SetKeepAlive(true)
		go mp.consume(conn)
	}
}

func (mp *PipeProxy) consume(conn net.Conn) {
	defer conn.Close()

	tgtAddr := mp.tunSrc.GetTarget(conn)
	if len(tgtAddr) == 0 {
		fmt.Println("\nNo such connection's target address:->", conn.RemoteAddr().String())
		return
	}
	fmt.Println("\nProxy handshake success:", tgtAddr)

	pipe := mp.wallet.SetupPipe(conn, tgtAddr)

	pipe.PullDataFromServer()

	fmt.Printf("\n\nPipe for(%s) is closing", tgtAddr)
}

func (mp *PipeProxy) IsRunning() bool {
	return mp.runFlag
}
