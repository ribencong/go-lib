package tun2socksA

import (
	"fmt"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-lib/tun2Pipe"
	"github.com/ribencong/go-lib/wallet"
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

var _instance *pipeProxy.PipeProxy = nil
var proxyConf *pipeProxy.ProxyConfig = &pipeProxy.ProxyConfig{}

func InitWallet(addr, cipher, license, url, boot string) {
	proxyConf.WConfig = &wallet.WConfig{
		BCAddr:     addr,
		Cipher:     cipher,
		License:    license,
		SettingUrl: url,
	}
	proxyConf.BootNodes = boot
}

func InitTun(reader VpnInputStream, writer VpnOutputStream, service VpnService, localAddr string, byPassIPs string) error {

	if reader == nil || writer == nil || service == nil {
		return fmt.Errorf("parameter invalid")
	}

	localIP, _, _ := net.SplitHostPort(localAddr)

	proxyConf.TunConfig = &tun2Pipe.TunConfig{
		Protector: func(fd uintptr) {
			service.ByPass(int32(fd))
		},
		TunWriter:  writer,
		TunReader:  reader,
		TunLocalIP: net.ParseIP(localIP),
		ByPassIPs:  byPassIPs,
	}
	return nil
}

func SetupVpn(password, locSer string) error {

	t2s, err := tun2Pipe.New(proxyConf.TunReader, proxyConf.ByPassIPs, locSer)
	if err != nil {
		return err
	}

	w, err := wallet.NewWallet(proxyConf.WConfig, password)
	if err != nil {
		return err
	}

	proxy, e := pipeProxy.NewProxy(locSer, w, t2s)
	if e != nil {
		return e
	}
	_instance = proxy
	return nil
}

func Run() {
	_instance.Proxying()
}

func StopVpn() {
	_instance.Close()
}
