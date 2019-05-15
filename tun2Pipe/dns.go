package tun2Pipe

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

type QueryState struct {
	QueryTime     time.Time
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

func NewDnsCache() (*DnsProxy, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: InnerPivotPort,
	})
	if err != nil {
		return nil, err
	}

	if err := ProtectConn(conn); err != nil {
		log.Println("DNS Cache Protect Err:", err)
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
	log.Println("	DNS :", dns.ID, string(dns.Questions[0].Name))

	qs := &QueryState{
		QueryTime:     time.Now(),
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

	log.Printf("DNS Query:(%d) %s-%s-%s", dns.QDCount, dns.Questions[0].Name,
		dns.Questions[0].Type.String(), dns.Questions[0].Class.String())
}

func (c *DnsProxy) DnsWaitResponse() {
	log.Println("DNS Waiting Response......", c.poxyConn.LocalAddr())
	buff := make([]byte, math.MaxInt16)
	defer c.poxyConn.Close()

	for {
		n, _, e := c.poxyConn.ReadFromUDP(buff)
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

		log.Printf("Get dns Response:ID=%d as=%d", dns.ID, dns.ANCount)

		for _, as := range dns.Answers {
			log.Printf("(%d)dns answer:%s\n", dns.ID, as.String())
		}

		qs := c.Get(dns.ID)
		if qs == nil {
			log.Println("no such request:", dns.ID)
			continue
		}

		data := WrapIPPacketForUdp(qs.ClientIP, qs.RemoteIp, int(qs.ClientPort), int(qs.RemotePort), buff[:n])
		if data == nil {
			log.Println("make dns ip packet failed:", dns.ID)
			continue
		}

		n2, err := c.VpnWriteBack.Write(data)
		if err != nil {
			log.Println("DNS data write to Tun err:", err)
			continue
		}
		log.Printf("(%d)DNS (%d)writer(%d) success:", n, len(data), n2)

	}
}
