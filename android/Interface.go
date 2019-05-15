package tun2socksA

import (
	"fmt"
	"github.com/ribencong/go-lib/tcpPivot"
	"github.com/ribencong/go-lib/tun2Pipe"
	"io"
	"net"
)

type VpnService interface {
	ByPass(fd int32) bool
}

type VpnInputStream interface {
	tun2Pipe.VpnInputStream
}

type VpnOutputStream interface {
	io.WriteCloser
}

var _instance *tun2Pipe.Tun2Socks = nil

func SetupVpn(reader VpnInputStream, writer VpnOutputStream, service VpnService, localIP string, byPassIPs string) error {

	if reader == nil || writer == nil || service == nil {
		return fmt.Errorf("parameter invalid")
	}

	control := func(fd uintptr) {
		service.ByPass(int32(fd))
	}

	tun2Pipe.SysConfig.Protector = control
	tun2Pipe.SysConfig.TunLocalIP = net.ParseIP(localIP)
	tun2Pipe.SysConfig.TunWriteBack = writer

	proxy, e := tcpPivot.NewAndroidProxy(localIP + ":0")
	if e != nil {
		return e
	}

	t2s, err := tun2Pipe.New(reader, proxy, byPassIPs)
	_instance = t2s
	return err
}

func Run() {
	_instance.ReadTunData()
}

func StopVpn() {
	_instance.Close()
}
