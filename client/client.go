package client

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
	"sync"
	"time"
)

const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc.torrent"

//const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent"

type Config struct {
	Addr        string
	Cipher      string
	LocalServer string
	License     string
	SettingUrl  string
}
type FlowCounter struct {
	sync.RWMutex
	closed    bool
	totalUsed int64
	unSigned  int64
}

type Client struct {
	*account.Account
	*FlowCounter
	fatalErr chan error

	proxyServer net.Listener
	payConn     *service.JsonConn

	aesKey     account.PipeCryptKey
	license    *service.License
	curService *service.ServeNodeId
}

func NewClient(conf *Config, password string) (*Client, error) {

	ls, err := net.Listen("tcp", conf.LocalServer)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\nSocks5 proxy server at:%s", conf.LocalServer)

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
	var url = conf.SettingUrl
	if len(url) == 0 {
		url = DefaultSeedSever
	}
	services := LoadBootStrap(url)
	mi := FindBestPath(services)
	if mi == nil {
		return nil, fmt.Errorf("no valid service")
	}

	fmt.Printf("\nfind server:%s", mi.ToString())

	c := &Client{
		Account:     acc,
		fatalErr:    make(chan error, 5),
		proxyServer: ls,
		license:     l,
		curService:  mi,
	}

	if err := c.Key.GenerateAesKey(&c.aesKey, mi.ID.ToPubKey()); err != nil {
		return nil, err
	}

	fmt.Println("\ncreate aes key success")

	return c, nil
}

func (c *Client) Running() error {

	defer c.proxyServer.Close()

	if err := c.createPayChannel(); err != nil {
		return err
	}

	defer c.payConn.Close()
	defer func() {
		c.FlowCounter.closed = true
	}()

	fmt.Println("\ncreate payment channel success")

	go c.payMonitor()

	for {
		conn, err := c.proxyServer.Accept()
		if err != nil {
			return fmt.Errorf("\nFinish to proxy system request :%s", err)
		}

		fmt.Println("\nNew system proxy request:", conn.RemoteAddr().String())
		conn.(*net.TCPConn).SetKeepAlive(true)
		go c.consume(conn)
	}
}

func (c *Client) createPayChannel() error {

	addr := c.curService.TONetAddr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	data, err := json.Marshal(c.license)
	if err != nil {
		return nil
	}

	hs := &service.YPHandShake{
		CmdType: service.CmdPayChanel,
		Sig:     c.Sign(data),
		Lic:     c.license,
	}

	jsonConn := &service.JsonConn{Conn: conn}
	if err := jsonConn.Syn(hs); err != nil {
		return err
	}

	c.payConn = jsonConn

	c.FlowCounter = &FlowCounter{}
	return nil
}

func (c *Client) Close() {
	c.fatalErr <- nil
	c.FlowCounter.closed = true
	c.payConn.Close()
	c.proxyServer.Close()
}

func LoadBootStrap(url string) []string {
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
		fmt.Printf("\n%s\n", nodeId)
	}
	return servers
}

func FindBestPath(paths []string) *service.ServeNodeId {

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
