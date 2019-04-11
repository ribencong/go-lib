package main

import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/youpipe/go-lib/pbs"
	"github.com/youpipe/go-youPipe/account"
	"github.com/youpipe/go-youPipe/network"
	"golang.org/x/crypto/ed25519"
)

var currentService *Node = nil
var unlockedAcc *account.Account = nil

const KingFinger = account.ID("YP5rttHPzRsAe2RmF52sLzbBk4jpoPwJLtABaMv6qn7kVm")

//export LibCreateAccount
func LibCreateAccount(password string) (*C.char, *C.char) {

	key, err := account.GenerateKey(password)
	if err != nil {
		return C.CString(""), C.CString("")
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)
	fmt.Println(string(address))
	fmt.Println(cipherTxt)

	return C.CString(address.ToString()), C.CString(cipherTxt)
}

//export LibInitAccount
func LibInitAccount(cipherTxt, address, password string) bool {
	if unlockedAcc != nil && address == unlockedAcc.Address.ToString() {
		return true
	}

	id, err := account.ConvertToID(address)
	if err != nil {
		fmt.Println(err)
		return false
	}
	acc := &account.Account{
		Address: id,
		Key: &account.Key{
			LockedKey: base58.Decode(cipherTxt),
		},
	}

	if ok := acc.UnlockAcc(password); ok {
		unlockedAcc = acc
		fmt.Println("Unlock account success")
		fmt.Println(acc)
		return true
	}
	return false
}

//export LibStartService
func LibStartService(ls, rip, proxyID string) bool {
	if nil == unlockedAcc {
		fmt.Println("please unlock this node first")
		return false
	}

	pid, err := account.ConvertToID(proxyID)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if currentService != nil && currentService.IsRunning() {
		LibStopService()
	}

	port := pid.ToSocketPort()
	rs := network.JoinHostPort(rip, port)
	fmt.Printf("start service:%s<->%s peerId(%s)\n", ls, rs, pid)

	if currentService = NewNode(ls, rs); currentService == nil {
		return false
	}
	currentService.account = unlockedAcc
	currentService.proxyID = proxyID

	go currentService.Serving()
	return true
}

//export LibStopService
func LibStopService() bool {
	if currentService == nil {
		return true
	}

	currentService.Stop()
	currentService = nil
	defer fmt.Print("stop service success.....\n")
	return true
}

//export LibVerifyLicense
func LibVerifyLicense(license string) bool {
	fmt.Println(license)
	l := &pbs.License{}
	if err := json.Unmarshal([]byte(license), l); err != nil {
		fmt.Println(err)
		return false
	}

	msg, err := json.Marshal(l.Data)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return ed25519.Verify(KingFinger.ToPubKey(), msg, l.Sig)
}

//export LibIsAccountInit
func LibIsAccountInit() bool {
	return currentService != nil
}
