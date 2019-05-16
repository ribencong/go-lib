package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-lib/wallet"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"strings"
)

var proxyConf *pipeProxy.ProxyConfig = nil
var curProxy *pipeProxy.PipeProxy = nil

//export LibCreateAccount
func LibCreateAccount(password string) (*C.char, *C.char) {

	key, err := account.GenerateKey(password)
	if err != nil {
		return C.CString(""), C.CString("")
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)

	return C.CString(address.ToString()), C.CString(cipherTxt)
}

//export LibIsInit
func LibIsInit() bool {
	return curProxy != nil
}

//export LibVerifyAccount
func LibVerifyAccount(cipherTxt, address, password string) bool {
	if _, err := account.AccFromString(address, cipherTxt, password); err != nil {
		return false
	}
	return true
}

//export LibVerifyLicense
func LibVerifyLicense(license string) bool {
	if _, err := service.ParseLicense(license); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

//export LoadBootNodesFromServer
func LoadBootNodesFromServer() string {
	nodes := pipeProxy.LoadFromServer("")
	return strings.Join(nodes, "\n")
}

//export LibInitProxy
func LibInitProxy(addr, cipher, license, url, boot string) {
	proxyConf = &pipeProxy.ProxyConfig{
		WConfig: &wallet.WConfig{
			BCAddr:     addr,
			Cipher:     cipher,
			License:    license,
			SettingUrl: url,
		},
		BootNodes: boot,
	}
}

//export LibCreateProxy
func LibCreateProxy(password, locSer string) error {

	if proxyConf == nil {
		return fmt.Errorf("init the proxy configuration first please")
	}

	if curProxy.IsRunning() {
		return nil
	}

	mi := proxyConf.FindBestNodeServer()
	if mi == nil {
		return fmt.Errorf("no valid boot strap node")
	}
	proxyConf.ServerId = mi
	fmt.Println(proxyConf.ToString())

	w, err := wallet.NewWallet(proxyConf.WConfig, password)
	if err != nil {
		fmt.Println(err)
		return err
	}

	proxy, e := pipeProxy.NewProxy(locSer, w, NewTunReader())
	if e != nil {
		fmt.Println(e)
		return e
	}

	curProxy = proxy
	return nil
}

//export LibProxyRun
func LibProxyRun() {
	if curProxy == nil {
		return
	}
	fmt.Print("start proxy success.....\n")

	curProxy.Proxying()
	curProxy = nil
}

//export LibStopClient
func LibStopClient() {
	if curProxy == nil {
		return
	}

	curProxy.Close()
	curProxy = nil

	return
}

func main() {
}
