package iosLib

import "C"
import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	Separator = "@@@"

	DefaultDomainUrl = "https://raw.githubusercontent.com/youpipe/ypctorrent/master/gfw.torrent"
)

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

func GenPriKey(cipherTxt, address, password string) []byte {

	acc, err := account.AccFromString(address, cipherTxt, password)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	//fmt.Println(acc.Key.PriKey)

	return acc.Key.PriKey
}

func GenAesKey(priKey []byte, peerID string) []byte {

	var aesKey account.PipeCryptKey

	if err := account.GenerateAesKey(&aesKey, account.ID(peerID).ToPubKey(), priKey); err != nil {
		fmt.Println(err)
		return nil
	}

	return aesKey[:]
}

func LoadDomain(url string) string {
	if len(url) == 0 {
		url = DefaultDomainUrl
	}
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(body)
}

func TestAes(priKey []byte, iv []byte, data []byte) []byte {
	block, err := aes.NewCipher(priKey)
	if err != nil {
		fmt.Println("create cipher for connection err:", err)
		return nil
	}

	Coder := cipher.NewCFBEncrypter(block, iv)
	Coder.XORKeyStream(data, data)

	fmt.Printf(" encrypter:%02x\n\n", data)

	Decoder := cipher.NewCFBDecrypter(block, iv)
	Decoder.XORKeyStream(data, data)
	fmt.Printf(" decrypter:%02x\n\n", data)

	Coder2 := cipher.NewCFBEncrypter(block, iv)
	Coder2.XORKeyStream(data, data)
	fmt.Printf(" 222--->encrypter:%02x\n\n", data)

	return data
}
