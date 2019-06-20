package tun2Pipe

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"math"
	"net"
	"sync"
	"syscall"
	"time"
)

//TODO::to make sure this is usable
type UdpSession struct {
	sync.RWMutex
	*net.UDPConn
	UTime   time.Time
	SrcIP   net.IP
	SrcPort int
	ID      string
}

func (s *UdpSession) ProxyOut(data []byte) (int, error) {
	s.UpdateTime()
	return s.Write(data)
}

func (s *UdpSession) WaitingIn() {
	defer VpnInstance.Log(fmt.Sprintf("Udp session(%s) reading end:", s.ID))
	defer s.Close()

	buf := make([]byte, math.MaxInt16)
	for {
		n, rAddr, e := s.ReadFromUDP(buf)
		if e != nil {
			VpnInstance.Log(fmt.Sprintf("Udp session(%s) read err:%s", s.ID, e))
			return
		}

		//VpnInstance.Log(fmt.Sprintf("\nFrom(%s) UDP Received:%02x", rAddr.String(), buf[:n]))
		packet := gopacket.NewPacket(buf[:n], layers.LayerTypeDNS, gopacket.Default)
		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			VpnInstance.Log(fmt.Sprintln("---------- DNS answer!-------"))
			dns, _ := dnsLayer.(*layers.DNS)
			for _, a := range dns.Answers {
				VpnInstance.Log(fmt.Sprintf("name:%s -> ip:%s", a.Name, a.IP))
			}
			VpnInstance.Log(fmt.Sprintln("-----------------------------------"))
		}

		data := WrapIPPacketForUdp(rAddr.IP, s.SrcIP, rAddr.Port, s.SrcPort, buf[:n])

		if _, e := VpnInstance.Write(data); e != nil {
			VpnInstance.Log(fmt.Sprintf("Udp session(%s) write to tun err:%s", s.ID, e))
			continue
		}
		s.UpdateTime()
	}
}

func (s *UdpSession) UpdateTime() {
	s.Lock()
	defer s.Unlock()
	s.UTime = time.Now()
}

func (s *UdpSession) IsExpire() bool {
	s.RLock()
	defer s.RUnlock()

	return time.Now().After(s.UTime.Add(UDPSessionTimeOut))
}

type UdpProxy struct {
	sync.RWMutex
	Done       chan error
	NatSession map[int]*UdpSession
}

func NewUdpProxy() *UdpProxy {
	up := &UdpProxy{
		NatSession: make(map[int]*UdpSession),
		Done:       make(chan error),
	}

	go up.ExpireOldSession()
	return up
}

func (up *UdpProxy) ReceivePacket(ip4 *layers.IPv4, udp *layers.UDP) {

	srcPort := int(udp.SrcPort)
	s := up.getSession(srcPort)
	if s == nil {
		if s = up.newSession(ip4, udp); s == nil {
			return
		}
		up.addSession(s)
	}

	_, e := s.ProxyOut(udp.Payload)
	if e != nil {
		VpnInstance.Log(fmt.Sprintln("Udp Session proxy out err:", e))
		up.removeSession(s)
	}

	packet := gopacket.NewPacket(udp.Payload, layers.LayerTypeDNS, gopacket.Default)
	//log.Println(packet.Dump())
	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns, _ := dnsLayer.(*layers.DNS)
		if len(dns.Questions) == 0 {
			return
		}

		VpnInstance.Log(fmt.Sprintln("This is a DNS question!========>"))
		for _, q := range dns.Questions {
			VpnInstance.Log(fmt.Sprintf("%s-%s", q.Name, q.Class.String()))
		}
		VpnInstance.Log(fmt.Sprintln("================================>"))
	}
}

func (up *UdpProxy) getSession(port int) *UdpSession {
	up.RLock()
	defer up.RUnlock()
	return up.NatSession[port]
}

func (up *UdpProxy) newSession(ip4 *layers.IPv4, udp *layers.UDP) *UdpSession {

	d := &net.Dialer{
		Timeout: SysDialTimeOut,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(Protector)
		},
	}

	tarAddr := fmt.Sprintf("%s:%d", ip4.DstIP, udp.DstPort)
	c, e := d.Dial("udp", tarAddr)
	if e != nil {
		VpnInstance.Log(fmt.Sprintln("Udp session create err:", e, tarAddr))
		return nil
	}

	id := fmt.Sprintf("(%s:%d)->(%s)->(%s)", ip4.SrcIP, udp.SrcPort,
		c.LocalAddr().String(), c.RemoteAddr().String())

	s := &UdpSession{
		ID:      id,
		UDPConn: c.(*net.UDPConn),
		UTime:   time.Now(),
		SrcPort: int(udp.SrcPort),
		SrcIP:   ip4.SrcIP,
	}

	//log.Printf("New udp session(%s):", s.ID)

	go s.WaitingIn()
	return s
}

func (up *UdpProxy) addSession(s *UdpSession) {
	up.Lock()
	defer up.Unlock()
	up.NatSession[s.SrcPort] = s
}

func (up *UdpProxy) removeSession(s *UdpSession) {
	up.Lock()
	defer up.Unlock()

	delete(up.NatSession, s.SrcPort)
	s.Close()
}

func (up *UdpProxy) ExpireOldSession() {
	VpnInstance.Log(fmt.Sprintln("Udp proxy session aging start >>>>>>"))

	for {
		select {
		case <-time.After(UDPSessionTimeOut):
			for _, s := range up.NatSession {
				if s.IsExpire() {
					VpnInstance.Log(fmt.Sprintf("session(%s) expired", s.ID))
					up.removeSession(s)
				}
			}

		case err := <-up.Done:
			VpnInstance.Log(fmt.Sprintf("Udp serrsion expire thread exit for %s", err.Error()))
			return
		}
	}
}
