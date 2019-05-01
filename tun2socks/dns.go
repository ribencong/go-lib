package tun2socks

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"sync"
)

type QueryState struct {
	ClientQueryID uint16
	ClientIP      net.IP
	ClientPort    layers.UDPPort
	RemoteIp      net.IP
	RemotePort    layers.UDPPort
}

type DnsProxy struct {
	sync.RWMutex
	poxyConn     *net.UDPConn
	VpnWriteBack io.WriteCloser
	cache        map[uint16]*QueryState
}

func NewDnsCache(protect ConnProtect) (*DnsProxy, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 51080,
	})
	if err != nil {
		return nil, err
	}
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}
	if err := rawConn.Control(protect); err != nil {
		return nil, err
	}

	proxy := &DnsProxy{
		poxyConn: conn,
		cache:    make(map[uint16]*QueryState),
	}

	go proxy.DnsWaitResponse()

	return proxy, nil
}

func (c *DnsProxy) Put(qs *QueryState) {
	c.Lock()
	defer c.Unlock()
	c.cache[qs.ClientQueryID] = qs
}

func (c *DnsProxy) Get(id uint16) *QueryState {
	c.RLock()
	defer c.RUnlock()
	return c.cache[id]
}

func (c *DnsProxy) Pop(id uint16) *QueryState {
	c.Lock()
	defer c.Unlock()
	qs := c.cache[id]
	delete(c.cache, id)
	return qs
}

func (c *DnsProxy) sendOut(dns *layers.DNS, ip4 *layers.IPv4, udp *layers.UDP) {
	qs := &QueryState{
		ClientQueryID: dns.ID,
		ClientIP:      ip4.SrcIP,
		ClientPort:    udp.SrcPort,
		RemotePort:    udp.DstPort,
		RemoteIp:      ip4.DstIP,
	}
	c.Put(qs)

	if _, e := c.poxyConn.WriteTo(dns.LayerContents(), &net.UDPAddr{
		IP:   ip4.DstIP,
		Port: int(udp.DstPort),
	}); e != nil {
		log.Println("DNS Query err:", e)
		return
	}
	log.Println("DNS Query Send Success:")
}

func (c *DnsProxy) DnsWaitResponse() {
	buff := make([]byte, math.MaxInt16)
	defer c.poxyConn.Close()

	for {
		n, rAddr, e := c.poxyConn.ReadFromUDP(buff)
		if e != nil {
			log.Println("DNS exit:", e)
			return
		}
		var decoded []gopacket.LayerType
		dns := &layers.DNS{}
		p := gopacket.NewDecodingLayerParser(layers.LayerTypeDNS, dns)
		if err := p.DecodeLayers(buff[:n], &decoded); err != nil {
			log.Println("DNS-parser:", err)
			continue
		}

		log.Println("DNS-Response-Success:", dns.Answers)

		data := c.makeIpPack(rAddr, dns)
		if data == nil {
			log.Println("make dns ip packet failed:", dns.ID)
			continue
		}

		if _, err := c.VpnWriteBack.Write(data); err != nil {
			log.Println("DNS data write to Tun err:", err)
			continue
		}
	}
}

func (c *DnsProxy) makeIpPack(addr *net.UDPAddr, dns *layers.DNS) []byte {

	qs := c.Get(dns.ID)
	if qs == nil {
		log.Println("no such request:", dns.ID)
		return nil
	}

	ipLayer := &layers.IPv4{
		SrcIP:    qs.RemoteIp,
		DstIP:    qs.ClientIP,
		Protocol: layers.IPProtocolUDP,
	}
	udpLayer := &layers.UDP{
		SrcPort: qs.RemotePort,
		DstPort: qs.ClientPort,
	}

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	if err := gopacket.SerializeLayers(b, opt, &layers.Ethernet{}, ipLayer, udpLayer, dns); err != nil {
		log.Println("Wrap dns to ip packet  err:", err)
		return nil
	}

	return b.Bytes()
}
