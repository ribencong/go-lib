package tun2socksA

import (
	"fmt"
	"github.com/ribencong/go-lib/tun2socks"
	"io"
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

func SetupVpn(reader VpnInputStream, writer VpnOutputStream, service VpnService, locSocks string) error {

	if reader == nil || writer == nil || service == nil {
		return fmt.Errorf("parameter invalid")
	}

	control := func(fd uintptr) {
		service.ByPass(int32(fd))
	}

	t2s, err := tun2socks.New(reader, writer, control, locSocks)
	_instance = t2s
	return err
}

func Run() {
	go _instance.Writing()
	go _instance.DnsWaitResponse()
	_instance.Reading()
}

func StopVpn() {
	_instance.Close()
}
