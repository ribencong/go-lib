package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/youpipe/go-youPipe/account"
)

var currentService *Node = nil
var unlockedAcc *account.Account = nil

//export createAccount
func createAccount(password string) (*C.char, string) {

	key, err := account.GenerateKey(password)
	if err != nil {
		return C.CString(""), ""
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)
	fmt.Println(string(address))
	fmt.Println(cipherTxt)

	return C.CString(address.ToString()), cipherTxt
}

//export initAccount
func initAccount(cipherTxt, address, password string) bool {
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
		return true
	}
	return false
}

//export startService
func startService(ls, rs, pid string) bool {
	if nil == unlockedAcc {
		fmt.Println("please unlock this node first")
		return false
	}

	if currentService != nil && currentService.IsRunning() {
		stopService()
	}
	fmt.Printf("start service:%s<->%s\n", ls, rs)
	if currentService = NewNode(ls, rs); currentService == nil {
		return false
	}
	currentService.account = unlockedAcc
	currentService.proxyID = pid

	go currentService.Serving()
	return true
}

//export stopService
func stopService() bool {

	if currentService == nil {
		return true
	}

	defer fmt.Print("stop service success.....\n")
	currentService.Stop()
	currentService = nil
	return true
}
