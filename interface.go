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
	"net"
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
func LibStartService(ls, rip, proxyID, license string) bool {
	fmt.Println(ls, rip, proxyID, license)
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
	l := &pbs.License{}
	if err := json.Unmarshal([]byte(license), l); err != nil {
		fmt.Println("parse license err:", err)
		return false
	}

	if !verifyLicenseData(l) {
		fmt.Println("license signature failed:")
		return false
	}

	currentService.account = unlockedAcc
	currentService.proxyID = proxyID
	currentService.license = l

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

	return verifyLicenseData(l)
}

func verifyLicenseData(l *pbs.License) bool {
	msg, err := json.Marshal(l.Data)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return ed25519.Verify(KingFinger.ToPubKey(), msg, l.Sig)
}

//export LibVerifyNetwork
func LibVerifyNetwork(ip, id string) int {
	fmt.Println(ip, id)
	trial := net.ParseIP(ip)
	if trial.To4() == nil {
		fmt.Printf("%v is not a valid IPv4 address\n", trial)

		if trial.To16() == nil {
			fmt.Printf("%v is not a valid IP address\n", trial)
			return -1
		}
	}

	if !account.ID(id).IsValid() {
		fmt.Println("not a valid id:->", id)
		return -2
	}

	return 0
}
