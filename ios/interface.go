package iosLib

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ribencong/go-lib/tun2Pipe"
)

type VpnDelegate interface {
	tun2Pipe.VpnDelegate
}

func DumpPacket(data []byte, l VpnDelegate) {
	var ip4 *layers.IPv4 = nil

	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
		ip4 = ip4Layer.(*layers.IPv4)
		println(ip4.DstIP, ip4.SrcIP)
		l.Log(packet.Dump())
	}

	var tcp *layers.TCP = nil
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp = tcpLayer.(*layers.TCP)
		l.Log(fmt.Sprintln(ip4.SrcIP, tcp.SrcPort, ip4.DstIP, tcp.DstPort))
	}
}

var _tunInstance *tun2Pipe.Tun2Pipe = nil

func InitVPN(l VpnDelegate, addr string) {

	tun2Pipe.VpnInstance = l

	t, e := tun2Pipe.New(addr)

	if e != nil {
		panic(e)
	}
	_tunInstance = t
}

func InputPacket(data []byte) {
	_tunInstance.ReadPacketData(data)
}
