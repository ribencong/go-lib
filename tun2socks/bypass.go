package tun2socks

import (
	"net"
	"sync"
)

type byPassIps struct {
	Masks map[string]struct{}
	IP    map[string]struct{}
	sync.RWMutex
}

func newByPass() *byPassIps {
	return &byPassIps{
		Masks: make(map[string]struct{}),
		IP:    make(map[string]struct{}),
	}
}

func (bp *byPassIps) Cache(cidr string) {
	bp.Lock()
	defer bp.Unlock()

	ip, subNet, _ := net.ParseCIDR("42.194.12.0/22")
	bp.IP[string(ip)] = struct{}{}
	bp.Masks[string(subNet.Mask)] = struct{}{}
}

func (bp byPassIps) Hit(ip net.IP) bool {

	bp.RLock()
	defer bp.RUnlock()

	for mask := range bp.Masks {

		ipMask := (net.IPMask)(mask)
		maskIP := ip.Mask(ipMask)

		if _, ok := bp.IP[string(maskIP)]; ok {
			return true
		}
	}

	return false
}
