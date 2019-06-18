package wallet

import (
	"encoding/json"
	"fmt"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"golang.org/x/crypto/ed25519"
	"log"
	"net"
	"sync"
	"syscall"
)

type WConfig struct {
	BCAddr     string
	Cipher     string
	License    string
	SettingUrl string
	ServerId   *ServeNodeId
	Saver      func(fd uintptr)
}

func (c *WConfig) ToString() string {
	return fmt.Sprintf("\t BCAddr:%s\n"+
		"\t Ciphere:%s\n"+
		"\tLicense:%s\n"+
		"\tSettingUrl:%s\n"+
		"\tServerId:%s\n",
		c.BCAddr,
		c.Cipher,
		c.License,
		c.SettingUrl,
		c.ServerId.ToString())
}

type FlowCounter struct {
	sync.RWMutex
	Closed    bool
	totalUsed int64
	unSigned  int64
	token     int64
}

func (p *FlowCounter) ToString() string {
	return fmt.Sprintf("close:%t totalUsed:%d unsigned:%d", p.Closed, p.totalUsed, p.unSigned)
}

type Wallet struct {
	acc      *account.Account
	counter  *FlowCounter
	sysSaver func(fd uintptr)

	payConn      *service.JsonConn
	aesKey       account.PipeCryptKey
	license      *service.License
	minerID      ed25519.PublicKey
	minerNetAddr string
}

func NewWallet(conf *WConfig, password string) (*Wallet, error) {

	acc, err := account.AccFromString(conf.BCAddr, conf.Cipher, password)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n Unlock client success:%s", conf.BCAddr)

	l, err := service.ParseLicense(conf.License)
	if err != nil {
		return nil, err
	}
	fmt.Println("\nParse license success")

	if l.UserAddr != acc.Address.ToString() {
		return nil, fmt.Errorf("license and account address are not same")
	}

	fmt.Printf("\n Selected miner id:%s", conf.ServerId.ToString())

	w := &Wallet{
		acc:          acc,
		license:      l,
		minerID:      make(ed25519.PublicKey, ed25519.PublicKeySize),
		sysSaver:     conf.Saver,
		minerNetAddr: conf.ServerId.TONetAddr(),
	}
	copy(w.minerID, conf.ServerId.ID.ToPubKey())

	if err := w.acc.Key.GenerateAesKey(&w.aesKey, conf.ServerId.ID.ToPubKey()); err != nil {
		return nil, err
	}

	if err := w.createPayChannel(); err != nil {
		log.Println("Create payment channel err:", err)
		return nil, err
	}

	fmt.Printf("\nCreate payment channel success:%s", w.ToString())

	go w.Running()

	return w, nil
}

func (w *Wallet) createPayChannel() error {
	fmt.Printf("\ncreatePayChannel Wallet socks ID addr:%s ", w.minerNetAddr)
	conn, err := w.getOuterConn(w.minerNetAddr)
	if err != nil {
		return err
	}

	data, err := json.Marshal(w.license)
	if err != nil {
		return err
	}

	hs := &service.YPHandShake{
		CmdType: service.CmdPayChanel,
		Sig:     w.acc.Sign(data),
		Lic:     w.license,
	}

	jsonConn := &service.JsonConn{Conn: conn}
	if err := jsonConn.Syn(hs); err != nil {
		return err
	}

	w.payConn = jsonConn

	w.counter = &FlowCounter{
		token: service.BandWidthPerToPay,
	}
	return nil
}

func (w *Wallet) Finish() {
	if w.counter.Closed {
		return
	}

	fmt.Println("Wallet is closing")
	w.counter.Closed = true
	w.payConn.Close()
}

func (w *Wallet) getOuterConn(addr string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: PipeDialTimeOut,
		Control: func(network, address string, c syscall.RawConn) error {
			if w.sysSaver != nil {
				return c.Control(w.sysSaver)
			}
			return nil
		},
	}

	return d.Dial("tcp", addr)
}

func (w *Wallet) ToString() string {
	return fmt.Sprintf("\t account:%s\n"+
		"\t counter:%s\n"+
		"\t minerID:%s\n"+
		"\t Address:%s\n",
		w.acc.Address,
		w.counter.ToString(),
		string(w.minerID),
		w.minerNetAddr)
}
