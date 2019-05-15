package wallet

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc.torrent"

//const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent"

type Config struct {
	Addr       string
	Cipher     string
	License    string
	SettingUrl string
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

	payConn    *service.JsonConn
	aesKey     account.PipeCryptKey
	license    *service.License
	curService *service.ServeNodeId
}

func NewWallet(conf *Config, password, bootNodes string) (*Wallet, error) {

	acc, err := account.AccFromString(conf.Addr, conf.Cipher, password)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\nUnlock client success:%s", conf.Addr)

	l, err := service.ParseLicense(conf.License)
	if err != nil {
		return nil, err
	}
	fmt.Println("\nParse license success")

	if l.UserAddr != acc.Address.ToString() {
		return nil, fmt.Errorf("license and account address are not same")
	}

	mi := FindBestPath(conf, bootNodes)
	if mi == nil {
		return nil, fmt.Errorf("no valid boot strap node")
	}

	fmt.Printf("\nfind server:%s", mi.ToString())

	w := &Wallet{
		Account:    acc,
		fatalErr:   make(chan error, 5),
		license:    l,
		curService: mi,
	}

	if err := w.Key.GenerateAesKey(&w.aesKey, mi.ID.ToPubKey()); err != nil {
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

	fmt.Println("\ncreate payment channel success")

	w.payMonitor()
}

func (w *Wallet) createPayChannel() error {

	addr := w.curService.TONetAddr()
	conn, err := net.Dial("tcp", addr)
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

func FindBestPath(conf *Config, bootNodes string) *service.ServeNodeId {
	var nodes []string
	if len(bootNodes) == 0 {
		nodes = LoadFromServer(conf.SettingUrl)
	} else {
		nodes = strings.Split(bootNodes, "\n")
	}

	return probeAllNodes(nodes)
}

func LoadFromServer(url string) []string {
	if len(url) == 0 {
		url = DefaultSeedSever
	}
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	servers := make([]string, 0)

	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		nodeStr, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println(err)
			if err == io.EOF {
				break
			} else {
				continue
			}
		}

		nodeId := base58.Decode(string(nodeStr))
		servers = append(servers, string(nodeId))
		fmt.Printf("LoadFromServer:\n%s\n", nodeId)
	}
	return servers
}

func probeAllNodes(paths []string) *service.ServeNodeId {

	var locker sync.Mutex
	s := make([]*service.ServeNodeId, 0)

	var waiter sync.WaitGroup
	for _, path := range paths {

		mi := service.ParseService(path)
		waiter.Add(1)

		go func() {
			defer waiter.Done()
			now := time.Now()
			if mi == nil || !mi.IsOK() {
				fmt.Printf("\nserver(%s) is invalid\n", mi.IP)
				return
			}

			mi.Ping = time.Now().Sub(now)
			fmt.Printf("\nserver(%s) is ok (%dms)\n", mi.IP, mi.Ping/time.Millisecond)
			locker.Lock()
			s = append(s, mi)
			locker.Unlock()
		}()
	}

	waiter.Wait()

	if len(s) == 0 {
		return nil
	}

	sort.Slice(s, func(i, j int) bool {
		return s[i].Ping < s[j].Ping
	})
	return s[0]
}
