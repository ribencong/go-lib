package tun2socks

import (
	"encoding/base64"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

var (
	SysConfig = &PacConfig{
		TunLocalIP: net.ParseIP("10.8.0.2"),
		PacCache:   make(map[string]struct{}),
	}
)

const (
	GFListUrl = "https://raw.githubusercontent.com/youpipe/ypctorrent/master/gfw.torrent"
)

type PacConfig struct {
	sync.RWMutex
	Protector    ConnProtect
	TunWriteBack io.WriteCloser
	TunLocalIP   net.IP
	IsGlobal     bool
	PacCache     map[string]struct{}
}

func (c *PacConfig) NeedProxy(target string) bool {
	if c.IsGlobal {
		return true
	}

	c.RLock()
	defer c.RUnlock()

	domain, err := publicsuffix.EffectiveTLDPlusOne(target)
	if err != nil {
		return false
	}

	if _, ok := c.PacCache[domain]; ok {
		return true
	}

	return false
}

func (c *PacConfig) RefreshRemoteGFWList() (string, error) {
	resp, err := http.Get(GFListUrl)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	buf, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		log.Println("Update GFW list err:", e)
		return "", err
	}

	domains, err := base64.StdEncoding.DecodeString(string(buf))
	if err != nil {
		log.Println("Update GFW list err:", e)
		return "", err
	}

	return string(domains), nil
}

func (c *PacConfig) LoadGFW(domains string) {
	domainArr := strings.Split(domains, "\n")
	c.Lock()
	defer c.Unlock()
	for _, dom := range domainArr {
		c.PacCache[dom] = struct{}{}
		log.Println("LoadGFW :", dom)
	}
}
