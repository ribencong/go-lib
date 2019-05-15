package tcpPivot

import (
	"fmt"
	"github.com/ribencong/go-lib/wallet"
	"log"
	"net"
	"strconv"
)

type MacProxy struct {
	serverIP   string
	serverPort int
	*net.TCPListener
	wallet *wallet.Wallet
}

func NewMacProxy(addr string, w *wallet.Wallet) (*MacProxy, error) {
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	ip, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	ap := &MacProxy{
		serverIP:    ip,
		serverPort:  p,
		TCPListener: l.(*net.TCPListener),
		wallet:      w,
	}

	go ap.Proxying()
	return ap, nil
}

func (mp *MacProxy) GetServeIP() string {
	return mp.serverIP
}

func (mp *MacProxy) GetServePort() int {
	return mp.serverPort
}

func (mp *MacProxy) Proxying() {

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

func (mp *MacProxy) consume(conn net.Conn) {
	defer conn.Close()

	obj, err := ProxyHandShake(conn)
	if err != nil {
		fmt.Println("\nSock5 handshake err:->", err)
		return
	}
	fmt.Println("\nProxy handshake success:", obj.Target)

	pipe := mp.wallet.SetupPipe(conn, obj.Target)

	pipe.PullDataFromServer()

	fmt.Printf("\n\nPipe for(%s) is closing", obj.Target)
}
