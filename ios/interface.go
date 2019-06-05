package iosLib

import "C"
import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"strings"
)

const Separator = "@@@"

func LoadNodes() string {

	nodes := pipeProxy.LoadFromServer("")
	return strings.Join(nodes, "\n")
}

func FindBestNode(nodesStr string) string {
	nodes := strings.Split(nodesStr, "\n")
	validIDs := pipeProxy.ProbeAllNodes(nodes, nil)
	if len(validIDs) == 0 {
		return ""
	}

	minerId := validIDs[0]

	return fmt.Sprintf("%s%s%s", minerId.ID, Separator, minerId.TONetAddr())
}

func CreateAccount(password string) string {
	key, err := account.GenerateKey(password)
	if err != nil {
		return ""
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)

	return address.ToString() + Separator + cipherTxt
}

func VerifyLicense(license string) bool {
	if _, err := service.ParseLicense(license); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func VerifyAccount(cipherTxt, address, password string) bool {
	if _, err := account.AccFromString(address, cipherTxt, password); err != nil {
		return false
	}
	return true
}

func OpenPriKey(cipherTxt, address, password string) []byte {
	acc, err := account.AccFromString(address, cipherTxt, password)
	if err != nil {
		return nil
	}
	return acc.Key.PriKey
}

func GenAesKey(peerAddr string, priKey []byte) []byte {
	var aesKey account.PipeCryptKey

	if err := account.GenerateAesKey(&aesKey, account.ID(peerAddr).ToPubKey(), priKey); err != nil {
		return nil
	}
	return aesKey[:]
}
