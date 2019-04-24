package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/youpipe/go-youPipe/account"
	"github.com/youpipe/go-youPipe/service"
	"github.com/youpipe/go-youPipe/service/client"
)

var proxyClient *client.Client = nil

const KingFinger = account.ID("YP5rttHPzRsAe2RmF52sLzbBk4jpoPwJLtABaMv6qn7kVm")

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
func LibCreateClient(addr, cipher, password, license, locSer, netUrl string) bool {

	if proxyClient != nil {
		return true
	}

	conf := &client.Config{
		Addr:        addr,
		Cipher:      cipher,
		License:     license,
		LocalServer: locSer,
		SettingUrl:  netUrl,
	}

	pc, err := client.NewClient(conf, password)
	if err != nil {
		return false
	}
	proxyClient = pc
	return true
}

//export LibProxyRun
func LibProxyRun() {
	if proxyClient == nil {
		return
	}
	fmt.Print("start proxy success.....\n")

	err := proxyClient.Running()
	fmt.Println(err)
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
