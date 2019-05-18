package pipeProxy

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"log"
	"net"
	"strconv"
)

type PipeProxy struct {
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
	return ap, nil
}

func (pp *PipeProxy) GetServeIP() string {
	return pp.serverIP
}

func (pp *PipeProxy) GetServePort() int {
	return pp.serverPort
}

func (pp *PipeProxy) Proxying() {

	log.Println("Proxy start working at:", pp.Addr().String())
	defer pp.Close()
	defer log.Println("Proxy exit")

	for {
		conn, err := pp.Accept()
		if err != nil {
			fmt.Printf("\nFinish to proxy system request :%s", err)
			return
		}

		fmt.Println("\nNew system proxy request:", conn.RemoteAddr().String())
		conn.(*net.TCPConn).SetKeepAlive(true)
		go pp.consume(conn)
	}
}

func (pp *PipeProxy) consume(conn net.Conn) {
	defer conn.Close()

	tgtAddr := pp.tunSrc.GetTarget(conn)
	if len(tgtAddr) == 0 {
		fmt.Println("\nNo such connection's target address:->", conn.RemoteAddr().String())
		return
	}
	fmt.Println("\n Socks Proxy handshake success:", tgtAddr)

	pipe := pp.wallet.SetupPipe(conn, tgtAddr)
	if nil == pipe {
		fmt.Println("Create pipe failed:", tgtAddr)
		return
	}

	pipe.PullDataFromServer()

	fmt.Printf("\n\nPipe for(%s) is closing", tgtAddr)
}
