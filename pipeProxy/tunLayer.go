package pipeProxy

import "net"

type Tun2Pipe interface {
	GetTarget(conn net.Conn) string
	Finish()
	InputPacket(data []byte)
}
