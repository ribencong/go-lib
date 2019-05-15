package tun2socks

import (
	"log"
	"net"
	"sync"
)

type byPassIps struct {
	Masks map[string]net.IPMask
	IP    map[string]struct{}
	sync.RWMutex
}

func newByPass() *byPassIps {
	return &byPassIps{
		Masks: make(map[string]net.IPMask),
		IP:    make(map[string]struct{}),
	}
}

func (bp *byPassIps) Cache(cidr string) {
	bp.Lock()
	defer bp.Unlock()

	ip, subNet, _ := net.ParseCIDR(cidr)
	bp.IP[ip.String()] = struct{}{}
	bp.Masks[subNet.Mask.String()] = subNet.Mask
	//log.Printf("\nCache:ip:%s->ip mask:%s", string(ip), string(subNet.Mask))
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
