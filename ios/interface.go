package iosLib

import (
	"github.com/ribencong/go-lib/tun2Pipe"
)

//func ReadPacketData(data []byte) string {
//	var ip4 *layers.IPv4 = nil
//
//	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
//	if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
//		ip4 = ip4Layer.(*layers.IPv4)
//		println(ip4.DstIP, ip4.SrcIP)
//		return packet.Dump()
//	}
//
//	var tcp *layers.TCP = nil
//	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
//		tcp = tcpLayer.(*layers.TCP)
//		println( ip4.SrcIP, tcp.SrcPort, ip4.DstIP, tcp.DstPort)
//	}
//
//	return "---=>:unknown"
//}

type VpnDelegate interface {
	tun2Pipe.VpnDelegate
}

func InitVPN(d VpnDelegate, lAddr string) error {

	t2p, err := tun2Pipe.New(lAddr)
	if err != nil {
		return err
	}

	tun2Pipe.VpnInstance = d
	_tunInstance = t2p
	return nil
}

func InputPacketData(data []byte) {
	if _tunInstance == nil {
		return
	}
	_tunInstance.ReadPacketData(data)
}

var _tunInstance *tun2Pipe.Tun2Socks = nil

//var _instance *pipeProxy.PipeProxy = nil
//var proxyConf = &pipeProxy.ProxyConfig{}
////
//func InitVPN(addr, cipher, license, url, boot, IPs string, d VpnDelegate) error {
//
//	pt := func(fd uintptr) {
//		d.ByPass(int32(fd))
//	}
//
//	proxyConf.WConfig = &wallet.WConfig{
//		BCAddr:     addr,
//		Cipher:     cipher,
//		License:    license,
//		SettingUrl: url,
//		Saver:      pt,
//	}
//
//	proxyConf.BootNodes = boot
//	tun2Pipe.ByPassInst().Load(IPs)
//
//	mis := proxyConf.FindBootServers(d.GetBootPath())
//	if len(mis) == 0 {
//		return fmt.Errorf("no valid boot strap node")
//	}
//
//	proxyConf.ServerId = mis[0]
//
//	tun2Pipe.VpnInstance = d
//	tun2Pipe.Protector = pt
//
//	return nil
//}
//
//func SetupVpn(password, locAddr string) error {
//
//	t2s, err := tun2Pipe.New(locAddr)
//	if err != nil {
//		return err
//	}
//
//	w, err := wallet.NewWallet(proxyConf.WConfig, password)
//	if err != nil {
//		return err
//	}
//
//	proxy, e := pipeProxy.NewProxy(locAddr, w, t2s)
//	if e != nil {
//		return e
//	}
//	_instance = proxy
//	return nil
//}
//
//func Run() {
//	_instance.Proxying()
//}
//
//func StopVpn() {
//	_instance.Close()
//}
