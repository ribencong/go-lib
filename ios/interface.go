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

func LoadNodes() string{

	nodes := pipeProxy.LoadFromServer("")
	return strings.Join(nodes, "\n")
 }

func FindBestNode(nodesStr string) string{
	nodes := strings.Split(nodesStr, "\n")
	validIDs := pipeProxy.ProbeAllNodes(nodes, nil)
	if len(validIDs) == 0{
		return ""
	}

	minerId := validIDs[0]

	return fmt.Sprintf("%s@%s",minerId.ID, minerId.TONetAddr())
}

func CreateAccount(password string) string{
	key, err := account.GenerateKey(password)
	if err != nil {
		return ""
	}
	address := key.ToNodeId()
	cipherTxt := base58.Encode(key.LockedKey)

	return address.ToString() + "@@@"+cipherTxt
}

func VerifyLicense(license string) bool {
	if _, err := service.ParseLicense(license); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}