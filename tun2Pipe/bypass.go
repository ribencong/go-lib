package tun2Pipe

import (
	"log"
	"net"
	"strings"
	"sync"
)

type byPassIps struct {
	Masks map[string]net.IPMask
	IP    map[string]struct{}
	sync.RWMutex
}

func newByPass(ips string) *byPassIps {

	bp := &byPassIps{
		Masks: make(map[string]net.IPMask),
		IP:    make(map[string]struct{}),
	}

	array := strings.Split(ips, "\n")
	for _, cidr := range array {
		ip, subNet, _ := net.ParseCIDR(cidr)
		bp.IP[ip.String()] = struct{}{}
		bp.Masks[subNet.Mask.String()] = subNet.Mask
	}
	return bp
}

func (bp byPassIps) Hit(ip net.IP) bool {

	bp.RLock()
	defer bp.RUnlock()

	for _, mask := range bp.Masks {
		maskIP := ip.Mask(mask)
		if _, ok := bp.IP[maskIP.String()]; ok {
			log.Printf("\nHit success ip:%s->ip mask:%s", ip, maskIP)
			return true
		}
	}

	return false
}
