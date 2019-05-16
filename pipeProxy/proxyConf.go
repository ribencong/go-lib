package pipeProxy

import (
	"bufio"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ribencong/go-lib/wallet"
	"github.com/ribencong/go-youPipe/service"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc.torrent"

//const DefaultSeedSever = "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent"

type RefreshNodeIDs func(ids string)

type ProxyConfig struct {
	*wallet.WConfig
	BootNodes string
}

func (c *ProxyConfig) ToString() string {
	return fmt.Sprintf(">>>>>>>>>>>>>>>>>>>>>>\n"+
		"wallet:%s\n"+
		"bootnodes:%s\n"+
		">>>>>>>>>>>>>>>>>>>>>>\n",
		c.WConfig.ToString(),
		c.BootNodes,
	)
}

func (c *ProxyConfig) FindBootServers(path string) []*service.ServeNodeId {
	var nodes []string
	if len(c.BootNodes) == 0 {

		nodes = LoadFromServer(c.SettingUrl)
		if e := ioutil.WriteFile(path, []byte(strings.Join(nodes, "\n")), 0644); e != nil {
			println("create boot nodes file failed:", path, e)
		}

	} else {

		nodes = strings.Split(c.BootNodes, "\n")
	}

	IDs := probeAllNodes(nodes)

	if len(IDs) == 0 && len(c.BootNodes) != 0 {

		nodes = LoadFromServer(c.SettingUrl)

		if e := ioutil.WriteFile(path, []byte(strings.Join(nodes, "\n")), 0644); e != nil {
			println("replace boot nodes failed:", path, e)
		}

		return probeAllNodes(nodes)
	}

	return IDs
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

func probeAllNodes(paths []string) []*service.ServeNodeId {

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
		return s
	}

	sort.Slice(s, func(i, j int) bool {
		return s[i].Ping < s[j].Ping
	})
	return s
}
