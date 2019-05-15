package tcpPivot

import "net"

type TcpProxy interface {
	GetServeIP() string
	GetServePort() int
	Proxying()
	AddNatItem(keyPort int, tgtAddr *net.TCPAddr)
	GetTargetAddr(keyPort int) *net.TCPAddr
}
