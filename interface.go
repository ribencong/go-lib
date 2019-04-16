package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/youpipe/go-youPipe/account"
	"github.com/youpipe/go-youPipe/service"
	"github.com/youpipe/go-youPipe/service/client"
)

var clientConf = &client.Config{}
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

//export LibCreateClient
func LibCreateClient(password string) bool {
	pc, err := client.NewClient(clientConf, password)
	if err != nil {
		return false
	}
	proxyClient = pc
	return true
}

//export LibProxyRun
func LibProxyRun() bool {
	if proxyClient == nil {
		return false
	}
	fmt.Print("start proxy success.....\n")

	go func() {
		err := proxyClient.Running()
		fmt.Println(err)
		LibStopClient()
	}()

	return true
}

//export LibStopClient
func LibStopClient() bool {
	proxyClient.Close()
	return true
}

//export LibSetAccInfo
func LibSetAccInfo(cipherTxt, address string) {
	clientConf.Addr = address
	clientConf.Cipher = cipherTxt
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
		return false
	}
	return true
}

//export LibSetLicense
func LibSetLicense(license string) {
	clientConf.License = license
}

//export LibVSetNetworks
func LibVSetNetworks(ipIds []string) {
	clientConf.Services = ipIds
}

//export LibVerifyNodeId
func LibVerifyNodeId(ipId string) bool {
	node := service.ParseService(ipId)
	if node == nil {
		return false
	}

	return node.IsOK()
}
