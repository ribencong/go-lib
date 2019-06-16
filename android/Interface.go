package androidLib

import (
	"fmt"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-lib/tun2Pipe"
	"github.com/ribencong/go-lib/wallet"
)

type VpnDelegate interface {
	tun2Pipe.VpnDelegate
	GetBootPath() string
}

var _instance *pipeProxy.PipeProxy = nil
var proxyConf = &pipeProxy.ProxyConfig{}

func InitVPN(addr, cipher, license, url, boot, IPs string, d VpnDelegate) error {

	pt := func(fd uintptr) {
		d.ByPass(int32(fd))
	}

	proxyConf.WConfig = &wallet.WConfig{
		BCAddr:     addr,
		Cipher:     cipher,
		License:    license,
		SettingUrl: url,
		Saver:      pt,
	}

	proxyConf.BootNodes = boot
	tun2Pipe.ByPassInst().Load(IPs)

	mis := proxyConf.FindBootServers(d.GetBootPath())
	if len(mis) == 0 {
		return fmt.Errorf("no valid boot strap node")
	}

	proxyConf.ServerId = mis[0]

	tun2Pipe.VpnInstance = d
	tun2Pipe.Protector = pt

	return nil
}

func SetupVpn(password, locAddr string) error {

	t2s, err := tun2Pipe.New(locAddr)
	if err != nil {
		return err
	}

	w, err := wallet.NewWallet(proxyConf.WConfig, password)
	if err != nil {
		return err
	}

	proxy, e := pipeProxy.NewProxy(locAddr, w, t2s)
	if e != nil {
		return e
	}
	_instance = proxy
	return nil
}

func Proxying() {
	_instance.Proxying()
}

func StopVpn() {
	_instance.Finish()
}

func InputPacket(data []byte) error {

	if _instance == nil {
		return fmt.Errorf("tun isn't initilized ")
	}

	fmt.Printf("[%d][%02x]", len(data), data)
	_instance.TunSrc.InputPacket(data)

	return nil
}
