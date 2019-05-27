package tun2Pipe

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type ByPassIPs struct {
	Masks map[string]net.IPMask
	IP    map[string]struct{}
	sync.RWMutex
}

var _instance *ByPassIPs
var once sync.Once

func ByPassInst() *ByPassIPs {
	once.Do(func() {
		_instance = &ByPassIPs{
			Masks: make(map[string]net.IPMask),
			IP:    make(map[string]struct{}),
		}
	})
	return _instance
}

func (bp *ByPassIPs) Load(IPS string) {

	array := strings.Split(IPS, "\n")
	for _, cidr := range array {
		ip, subNet, _ := net.ParseCIDR(cidr)
		bp.IP[ip.String()] = struct{}{}
		bp.Masks[subNet.Mask.String()] = subNet.Mask
	}

	VpnInstance.Log(fmt.Sprintf("Total bypass ips:%d groups:%d \n", len(bp.IP), len(bp.Masks)))
}

func (bp *ByPassIPs) Hit(ip net.IP) bool {

	bp.RLock()
	defer bp.RUnlock()

	for _, mask := range bp.Masks {
		maskIP := ip.Mask(mask)
		if _, ok := bp.IP[maskIP.String()]; ok {
			VpnInstance.Log(fmt.Sprintf("\nHit success ip:%s->ip mask:%s", ip, maskIP))
			return true
		}
	}

	//TODO:: pac domain list

	return false
}
