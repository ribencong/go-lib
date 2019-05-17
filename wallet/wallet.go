package wallet

import (
	"encoding/json"
	"fmt"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"net"
	"sync"
	"syscall"
)

type WConfig struct {
	BCAddr     string
	Cipher     string
	License    string
	SettingUrl string
	ServerId   *service.ServeNodeId
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
}

type Wallet struct {
	*account.Account
	*FlowCounter
	fatalErr chan error
	sysSaver func(fd uintptr)

	payConn    *service.JsonConn
	aesKey     account.PipeCryptKey
	license    *service.License
	curService *service.ServeNodeId
}

func NewWallet(conf *WConfig, password string) (*Wallet, error) {

	acc, err := account.AccFromString(conf.BCAddr, conf.Cipher, password)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\nUnlock client success:%s", conf.BCAddr)

	l, err := service.ParseLicense(conf.License)
	if err != nil {
		return nil, err
	}
	fmt.Println("\nParse license success")

	if l.UserAddr != acc.Address.ToString() {
		return nil, fmt.Errorf("license and account address are not same")
	}
	w := &Wallet{
		Account:    acc,
		fatalErr:   make(chan error, 5),
		license:    l,
		curService: conf.ServerId,
		sysSaver:   conf.Saver,
	}

	if err := w.Key.GenerateAesKey(&w.aesKey, conf.ServerId.ID.ToPubKey()); err != nil {
		return nil, err
	}

	fmt.Println("\ncreate aes key success")
	go w.Running()

	return w, nil
}

func (w *Wallet) Running() {

	if err := w.createPayChannel(); err != nil {
		return
	}

	defer w.payConn.Close()
	defer func() {
		w.FlowCounter.Closed = true
	}()

	println("\ncreate payment channel success")

	w.payMonitor()
}

func (w *Wallet) createPayChannel() error {

	addr := w.curService.TONetAddr()
	conn, err := w.getOuterConn(addr)
	//conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	data, err := json.Marshal(w.license)
	if err != nil {
		return nil
	}

	hs := &service.YPHandShake{
		CmdType: service.CmdPayChanel,
		Sig:     w.Sign(data),
		Lic:     w.license,
	}

	jsonConn := &service.JsonConn{Conn: conn}
	if err := jsonConn.Syn(hs); err != nil {
		return err
	}

	w.payConn = jsonConn

	w.FlowCounter = &FlowCounter{}
	return nil
}

func (w *Wallet) Close() {
	w.fatalErr <- nil
	w.FlowCounter.Closed = true
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
