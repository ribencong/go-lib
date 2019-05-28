package iosLib

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ribencong/go-youPipe/service"
	"golang.org/x/net/publicsuffix"
	"net"
)

func GetDomain(host string) string {

	doamin, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return ""
	}
	return doamin
}

type VpnDelegate interface {
	Log(str string)
}

func DumpPacket(data []byte) {
	var ip4 *layers.IPv4 = nil

	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
		ip4 = ip4Layer.(*layers.IPv4)
		println(ip4.DstIP, ip4.SrcIP)
		_tunInterface.Log(packet.Dump())
	}

	var tcp *layers.TCP = nil
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp = tcpLayer.(*layers.TCP)
		_tunInterface.Log(fmt.Sprintln(ip4.SrcIP, tcp.SrcPort, ip4.DstIP, tcp.DstPort))
		return
	}

	_tunInterface.Log("Unknown protocol")
}

var _tunInterface VpnDelegate = nil

func InitVPN(l VpnDelegate, addr string) {
	_tunInterface = l
	newHttpServer(addr)
}

func newHttpServer(addr string) {
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			c, e := conn.Accept()
			if e != nil {
				_tunInterface.Log(fmt.Sprintf("Http server exit:%s", e))
				return
			}

			go consume(c)
		}
	}()
}

type targetAddr struct {
	host string
	port int
}

func consume(c net.Conn) {
	defer c.Close()
	jsonConn := &service.JsonConn{Conn: c}
	tgt := &targetAddr{}
	if err := jsonConn.ReadJsonMsg(tgt); err != nil {
		_tunInterface.Log(fmt.Sprintf("Http consume read err:%s", err))
		return
	}

	if err := jsonConn.WriteJsonMsg(tgt); err != nil {
		_tunInterface.Log(fmt.Sprintf("Http consume write err:%s", err))
	}
}
