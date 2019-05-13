package tun2socksA

import (
	"fmt"
	"github.com/ribencong/go-lib/tun2socks"
	"io"
	"net"
)

type VpnService interface {
	ByPass(fd int32) bool
}

type VpnInputStream interface {
	tun2socks.VpnInputStream
}

type VpnOutputStream interface {
	io.WriteCloser
}

var _instance *tun2socks.Tun2Socks = nil

func SetupVpn(reader VpnInputStream, writer VpnOutputStream, service VpnService, localIP string, GFWList string) error {

	if reader == nil || writer == nil || service == nil {
		return fmt.Errorf("parameter invalid")
	}

	control := func(fd uintptr) {
		service.ByPass(int32(fd))
	}

	tun2socks.SysConfig.Protector = control
	tun2socks.SysConfig.TunLocalIP = net.ParseIP(localIP)
	tun2socks.SysConfig.TunWriteBack = writer
	if len(GFWList) == 0 {
		GFWList, _ = tun2socks.SysConfig.RefreshRemoteGFWList()
	}
	tun2socks.SysConfig.LoadGFW(GFWList)

	t2s, err := tun2socks.New(reader)
	_instance = t2s
	return err
}

func Run() {
	_instance.Reading()
}

func StopVpn() {
	_instance.Close()
}

func LoadGFWList() string {
	str, e := tun2socks.SysConfig.RefreshRemoteGFWList()
	if e != nil {
		return ""
	}
	return str
}
