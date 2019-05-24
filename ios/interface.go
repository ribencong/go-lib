package iosDelegate

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func ParseNsData(data []byte) string {
	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	if ip4Layer := packet.Layer(layers.LayerTypeIPv4); ip4Layer != nil {
		ip4 := ip4Layer.(*layers.IPv4)
		println(ip4.DstIP, ip4.SrcIP)
		return packet.Dump()
	}
	return "---=>:unknown"
}

//
//type VpnDelegate interface {
//	tun2Pipe.VpnDelegate
//	GetBootPath() string
//}
//
//var _instance *pipeProxy.PipeProxy = nil
//var proxyConf = &pipeProxy.ProxyConfig{}
//
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
