package androidLib

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-lib/tun2Pipe"
	"github.com/ribencong/go-lib/wallet"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
)

type VpnDelegate interface {
	tun2Pipe.VpnDelegate
	GetBootPath() string
}

const Separator = "@@@"

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
	tun2Pipe.VpnInstance = d
	tun2Pipe.Protector = pt

	proxyConf.BootNodes = boot
	tun2Pipe.ByPassInst().Load(IPs)

	mis := proxyConf.FindBootServers(d.GetBootPath())
	if len(mis) == 0 {
		return fmt.Errorf("no valid boot strap node")
	}

	proxyConf.ServerId = mis[0]
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
	if _instance == nil {
		return
	}

	_instance.Proxying()
}

func StopVpn() {
	if _instance != nil {
		_instance.Finish()
		_instance = nil
	}
}

func InputPacket(data []byte) error {

	if _instance == nil {
		return fmt.Errorf("tun isn't initilized ")
	}

	//fmt.Printf("[%d][%02x]", len(data), data)
	_instance.TunSrc.InputPacket(data)

	return nil
}

func VerifyAccount(addr, cipher, password string) bool {
	if _, err := account.AccFromString(addr, cipher, password); err != nil {
		fmt.Println("ValidAccount:", err)
		return false
	}
	return true
}

func VerifyLicense(license string) bool {
	if len(license) == 0 {
		return false
	}

	if _, err := service.ParseLicense(license); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func CreateAccount(password string) string {

	key, err := account.GenerateKey(password)
	if err != nil {
		return ""
	}
	address := key.ToNodeId().ToString()
	cipherTxt := base58.Encode(key.LockedKey)

	return address + Separator + cipherTxt
}
