package tun2Pipe

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"log"
	"math"
	"net"
	"sync"
	"syscall"
	"time"
)

type UdpSession struct {
	sync.RWMutex
	net.Conn
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
	//log.Printf("Udp session(%s) start:", s.ID)
	defer log.Printf("Udp session(%s) end:", s.ID)
	defer s.Close()

	buf := make([]byte, math.MaxInt16)
	for {
		n, e := s.Read(buf)
		if e != nil {
			log.Printf("Udp session(%s) read err:%s", s.ID, e)
			return
		}

		rAddr := s.RemoteAddr().(*net.UDPAddr)
		data := WrapIPPacketForUdp(rAddr.IP, s.SrcIP, rAddr.Port, s.SrcPort, buf[:n])

		if _, e := SysConfig.TunWriteBack.Write(data); e != nil {
			log.Printf("Udp session(%s) write to tun err:%s", s.ID, e)
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
	NatSession map[int]*UdpSession
}

func NewUdpProxy() *UdpProxy {
	up := &UdpProxy{
		NatSession: make(map[int]*UdpSession),
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

	if _, e := s.ProxyOut(udp.Payload); e != nil {
		log.Println("Udp Session proxy out err:", e)
		up.removeSession(s)
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
			return c.Control(SysConfig.Protector)
		},
	}

	tarAddr := fmt.Sprintf("%s:%d", ip4.DstIP, udp.DstPort)
	c, e := d.Dial("udp", tarAddr)
	if e != nil {
		log.Println("Udp session create err:", e, tarAddr)
		return nil
	}

	id := fmt.Sprintf("(%s:%d)->(%s)->(%s)", ip4.SrcIP, udp.SrcPort,
		c.LocalAddr().String(), c.RemoteAddr().String())
	s := &UdpSession{
		ID:      id,
		Conn:    c,
		UTime:   time.Now(),
		SrcPort: int(udp.SrcPort),
		SrcIP:   ip4.SrcIP,
	}

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
	log.Println("Udp proxy session aging start >>>>>>")

	for {
		select {
		case <-time.After(UDPSessionTimeOut):
			for _, s := range up.NatSession {
				if s.IsExpire() {
					log.Printf("session(%s) expired", s.ID)
					up.removeSession(s)
				}
			}
		}
	}
}
