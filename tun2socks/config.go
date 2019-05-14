package tun2socks

import (
	"io"
	"log"
	"net"
	"strings"
)

var (
	SysConfig = &PacConfig{
		TunLocalIP: net.ParseIP("10.8.0.2"),
		byPass:     newByPass(),
	}
)

const (
	GFListUrl = "https://raw.githubusercontent.com/youpipe/ypctorrent/master/gfw.torrent"
)

type PacConfig struct {
	Protector    ConnProtect
	TunWriteBack io.WriteCloser
	TunLocalIP   net.IP
	IsGlobal     bool
	byPass       *byPassIps
}

func (c *PacConfig) NeedProxy(ip net.IP) bool {
	if c.IsGlobal {
		return true
	}

	return c.byPass.Hit(ip)
}

func (c *PacConfig) ParseByPassIP(domains string) {
	array := strings.Split(domains, "\n")
	log.Println("domainArr Len :", len(array))

	for _, cidr := range array {
		c.byPass.Cache(cidr)
	}
}
