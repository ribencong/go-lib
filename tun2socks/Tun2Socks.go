package tun2socks

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

const (
	MTU = math.MaxInt16
)

var (
	_, ip1, _ = net.ParseCIDR("10.0.0.0/8")
	_, ip2, _ = net.ParseCIDR("172.16.0.0/12")
	_, ip3, _ = net.ParseCIDR("192.168.0.0/24")
)

func isPrivate(ip net.IP) bool {
	return ip1.Contains(ip) || ip2.Contains(ip) || ip3.Contains(ip)
}

type ConnProtect func(fd uintptr)
type UDPTuple struct {
	IsDns bool
	Src   *net.UDPAddr
	Dst   *net.UDPAddr
}

func (t *UDPTuple) String() string {
	return fmt.Sprintf("(DNS:%t)%s:%d->%s:%d", t.IsDns, t.Src.IP, t.Src.Port, t.Dst.IP, t.Dst.Port)
}

type Tun2Socks struct {
	udpTrans       *net.UDPConn
	dnsProxy       *DnsProxy
	natLock        sync.RWMutex
	natSessionMap  map[string]*UDPTuple
	dataSource     VpnInputStream
	vpnWriteBack   io.WriteCloser
	localSocksAddr string
	protect        ConnProtect
}

type VpnInputStream interface {
	ReadBuff() []byte
}

func New(reader VpnInputStream, writer io.WriteCloser, protect ConnProtect, locSocks string) (*Tun2Socks, error) {

	//proxy, err := NewDnsCache(protect)
	//if err != nil {
	//	return nil, err
	//}

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

	//proxy.VpnWriteBack = writer
	tsc := &Tun2Socks{
		//dnsProxy:       proxy,
		udpTrans:       conn,
		natSessionMap:  make(map[string]*UDPTuple),
		dataSource:     reader,
		vpnWriteBack:   writer,
		localSocksAddr: locSocks,
		protect:        protect,
	}
	go tsc.UdpRead()

	return tsc, nil
}

func (t2s *Tun2Socks) UdpRead() {
	buff := make([]byte, math.MaxInt16)
	defer t2s.udpTrans.Close()
	log.Println("UDP Server Start......")
	for {
		n, rAddr, e := t2s.udpTrans.ReadFromUDP(buff)
		if e != nil {
			log.Panicln("udp reading thread exit:", e)
			return
		}
		t2s.natLock.RLock()
		sess := t2s.natSessionMap[rAddr.String()]
		if sess == nil {
			t2s.natLock.RUnlock()
			log.Println("No such session :", rAddr.String())
			continue
		}
		t2s.natLock.RUnlock()

		log.Println("Found session :", sess.String())

		data := WrapIPPacket(sess.Src.IP, sess.Dst.IP, layers.UDPPort(sess.Src.Port),
			layers.UDPPort(sess.Dst.Port), buff[:n])

		n2, e := t2s.vpnWriteBack.Write(data)
		if e != nil {
			log.Println("VPN writer err:", e)
		} else {
			log.Printf("(%d)VPN (%d)writer(%d) success:", n, len(data), n2)
		}
	}
}

func (t2s *Tun2Socks) Reading() {
	for {
		buf := t2s.dataSource.ReadBuff()

		if len(buf) == 0 {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		ip4 := &layers.IPv4{}
		tcp := &layers.TCP{}
		udp := &layers.UDP{}
		dns := &layers.DNS{}
		parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, ip4, tcp, udp, dns)
		decodedLayers := make([]gopacket.LayerType, 0, 4)
		if err := parser.DecodeLayers(buf, &decodedLayers); err != nil {
			log.Println("  Decoded Err:", err)
			continue
		}

		var udpSession *UDPTuple = nil
		for _, typ := range decodedLayers {
			//log.Println("  Successfully decoded layer type", typ)
			switch typ {
			case layers.LayerTypeDNS:
				if udpSession != nil {
					udpSession.IsDns = true
				}
				break
			case layers.LayerTypeTCP:
				break
			case layers.LayerTypeUDP:
				udpSession = &UDPTuple{
					Src: &net.UDPAddr{
						IP:   ip4.SrcIP,
						Port: int(udp.SrcPort),
					},
					Dst: &net.UDPAddr{
						IP:   ip4.DstIP,
						Port: int(udp.DstPort),
					},
				}
				break
			}
		}

		if udpSession != nil {
			t2s.CacheNatSession(udpSession, ip4, udp)
		}
	}
}

func (t2s *Tun2Socks) CacheNatSession(udpSession *UDPTuple, ip4 *layers.IPv4, udp *layers.UDP) {
	t2s.natLock.Lock()
	t2s.natSessionMap[udpSession.Dst.String()] = udpSession
	t2s.natLock.Unlock()

	if _, e := t2s.udpTrans.WriteTo(udp.Payload, &net.UDPAddr{
		IP:   ip4.DstIP,
		Port: int(udp.DstPort),
	}); e != nil {
		log.Println("Udp Trans Err:", e)
		return
	}
	log.Println("new udp session:", udpSession.String())
}

func (t2s *Tun2Socks) Close() {
}

func WrapIPPacket(srcIp, DstIp net.IP, srcPort, dstPort layers.UDPPort, payload []byte) []byte {

	ip4 := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    srcIp,
		DstIP:    DstIp,
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: srcPort,
		DstPort: dstPort,
	}

	udp.SetNetworkLayerForChecksum(ip4)

	b := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(b, opt, ip4, udp, gopacket.Payload(payload)); err != nil {
		log.Println("Wrap dns to ip packet  err:", err)
		return nil
	}

	return b.Bytes()
}
