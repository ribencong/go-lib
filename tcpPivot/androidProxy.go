package tcpPivot

import (
	"log"
	"net"
	"strconv"
	"sync"
)

type NatCache struct {
	sync.RWMutex
	cache map[int]*net.TCPAddr
}

func (nc *NatCache) addCache(key int, value *net.TCPAddr) {
	nc.Lock()
	defer nc.Unlock()
	nc.cache[key] = value
}

func (nc *NatCache) getCache(key int) *net.TCPAddr {
	nc.RLock()
	defer nc.RUnlock()
	val, ok := nc.cache[key]
	if !ok {
		return nil
	}

	return val
}

//TODO:: remove the old nat cache.
func (nc *NatCache) removeCache(key int) {
	nc.Lock()
	defer nc.Unlock()
	delete(nc.cache, key)
}

type AndroidProxy struct {
	serverIP   string
	serverPort int
	*NatCache
	*net.TCPListener
}

func NewAndroidProxy(addr string) (*AndroidProxy, error) {
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}
	ip, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	ap := &AndroidProxy{
		serverIP:   ip,
		serverPort: p,
		NatCache: &NatCache{
			cache: make(map[int]*net.TCPAddr),
		},
		TCPListener: l.(*net.TCPListener),
	}

	go ap.Proxying()
	return ap, nil
}

func (ap *AndroidProxy) GetServeIP() string {
	return ap.serverIP
}

func (ap *AndroidProxy) GetServePort() int {
	return ap.serverPort
}

func (ap *AndroidProxy) Proxying() {

	log.Println("Android proxy start working at:", ap.Addr().String())
	defer ap.Close()
	defer log.Println("Android proxy exit")

	for {
		conn, err := ap.Accept()
		if err != nil {
			log.Println("proxy accept err:", err)
			return
		}

		srcPort := conn.RemoteAddr().(*net.TCPAddr).Port
		tgtAddr := ap.getCache(srcPort)
		if tgtAddr == nil {
			conn.Close()
			log.Println("can't find nat session cache for source port:", srcPort)
			continue
		}
		log.Println("New vpn request:", tgtAddr.String())
	}
}

func (ap *AndroidProxy) AddNatItem(keyPort int, tgtAddr *net.TCPAddr) {
	ap.addCache(keyPort, tgtAddr)
}

func (ap *AndroidProxy) GetTargetAddr(keyPort int) *net.TCPAddr {
	return ap.getCache(keyPort)
}
