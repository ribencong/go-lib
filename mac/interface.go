package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/tcpPivot"
	"github.com/ribencong/go-lib/wallet"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"strings"
)

var proxyClient *tcpPivot.MacProxy = nil
var SysCB SystemCallBack = nil

type SystemCallBack interface {
	RefreshNodeIDs(ids string)
}

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
	return proxyClient != nil
}

//export LibCreateClient
func LibCreateClient(addr, cipher, password, license, locSer, netUrl, bootNodes string, cb SystemCallBack) bool {

	SysCB = cb
	if proxyClient != nil {
		return true
	}

	fmt.Println(addr, cipher, license, locSer, netUrl)

	conf := &wallet.Config{
		Addr:       addr,
		Cipher:     cipher,
		License:    license,
		SettingUrl: netUrl,
	}

	w, err := wallet.NewWallet(conf, password, bootNodes)
	if err != nil {
		fmt.Println(err)
		return false
	}
	proxy, e := tcpPivot.NewMacProxy(locSer, w)
	if e != nil {
		return false
	}
	proxyClient = proxy
	return true
}

//export LibProxyRun
func LibProxyRun() {
	if proxyClient == nil {
		return
	}
	fmt.Print("start proxy success.....\n")

	proxyClient.Proxying()
	proxyClient = nil
}

//export LibStopClient
func LibStopClient() {
	if proxyClient == nil {
		return
	}

	proxyClient.Close()
	proxyClient = nil
	return
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
	nodes := wallet.LoadFromServer("")
	return strings.Join(nodes, "\n")
}

func main() {
}
