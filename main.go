package main

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/youpipe/go-youPipe/account"
	"os"
)

func main() {
	test4()
}
func test4() {
	b := initAccount(os.Args[6], os.Args[5], os.Args[4])
	fmt.Printf("unlock:%t\n", b)
	if !b {
		panic("unlock failed")
	}
	b = startService(os.Args[1], os.Args[2], os.Args[3])
	if !b {
		panic("service failed")
	}
	<-make(chan struct{})
}
func test1() {
	b := startService(os.Args[1], os.Args[2], os.Args[3])
	if !b {
		panic("failed")
	}
	<-make(chan struct{})
}
func test2() {
	a, c := createAccount(os.Args[1])
	fmt.Println(a)
	fmt.Println(c)
}
func test3() {
	b := initAccount(os.Args[1], os.Args[2], os.Args[3])
	fmt.Printf("unlock:%t\n", b)
}

var currentService *Node = nil
var unlockedAcc *account.Account = nil

//export createAccount
func createAccount(password string) (string, string) {

	key, err := account.GenerateKey(password)
	if err != nil {
		return "", ""
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)
	return string(address), cipherTxt
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
