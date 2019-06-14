package main

import "C"
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ribencong/go-lib/pipeProxy"
	"github.com/ribencong/go-lib/wallet"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
)

var proxyConf = &pipeProxy.ProxyConfig{
	WConfig: &wallet.WConfig{
		BCAddr:     "YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ",
		Cipher:     "GffT4JanGFefAj4isFLYbodKmxzkJt9HYTQTKquueV8mypm3oSicBZ37paYPnDscQ7XoPa4Qgse6q4yv5D2bLPureawFWhicvZC5WqmFp9CGE",
		License:    `{"sig":"diILHCClpQM+oo3c3aDPr0RY1AhmLRqDapT077h9a9toxb1GgbGkpd4uYP2v1HmYPCU+QITR7saydty3P43TBw==","start":"2019-06-14 02:26:22","end":"2019-07-14 02:26:22","user":"YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ"}`,
		SettingUrl: "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent",
		Saver:      nil,
	},
	BootNodes: "YPBzFaBFv8ZjkPQxtozNQe1c9CvrGXYg4tytuWjo9jiaZx@192.168.30.12",
}

func main() {

	mis := proxyConf.FindBootServers(".")
	if len(mis) == 0 {
		fmt.Println("no valid boot strap node")
		panic("no boot node")
	}

	proxyConf.ServerId = mis[0]

	fmt.Println(proxyConf.ToString())

	w, err := wallet.NewWallet(proxyConf.WConfig, "12345678")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	proxy, e := pipeProxy.NewProxy(":51080", w, NewTunReader())
	if e != nil {
		fmt.Println(e)
		panic(e)
	}

	proxy.Proxying()
}

func test11() {
	license := `{"sig":"8BeMM5+orhR4YY+TJbNw1Wg4BhAq8PSMNuFgx8Ne+I4prJPJmViY2O/OQ7i5VelN3aUR3y3ZpoQ7pFd1612GAw==","start":"2019-06-06 06:50:52","end":"2019-06-21 06:50:52","user":"YPAMynSajgK8SVgHAaFocZTT8QF2za1u9xmS5TQpnqoTjX"}`
	l, e := service.ParseLicense(license)
	if e != nil {
		panic(e)
	}

	data, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	acc, err := account.AccFromString("YPAMynSajgK8SVgHAaFocZTT8QF2za1u9xmS5TQpnqoTjX",
		"kBnUbRqGnreZvkDhgVkYSR6L4oc1izBwaKdo99uWnijRwKqyL8zhiZX8cwVGxjerfhde8MjPjnHgRfTjWyQ5zUYGheJrLxrfVTPWgFb4YQ3Nh",
		"12345678")
	if err != nil {
		panic(err)
	}

	sign := ed25519.Sign(acc.Key.PriKey, data)
	str := base64.StdEncoding.EncodeToString(sign)
	println(str)
}
func test10() {
	fmt.Println(publicsuffix.EffectiveTLDPlusOne("1-apple.com.tw"))
}
func test9() {
	tt, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(tt))
}

func test8() {
	ip, subNet, _ := net.ParseCIDR("58.248.0.0/13")
	mask := subNet.Mask

	srcIP := net.ParseIP("58.251.82.180")
	maskIP := srcIP.Mask(mask)

	fmt.Println(ip.String(), maskIP.String())
	fmt.Println(string(maskIP), string(ip), maskIP.String() == ip.String(), subNet.Contains(srcIP))
}

func test7() {
	str := "zoFqdxIrIwcRWPyALfi7yCVvJagI6hE86K3KNc0ioPxsSJWqYa2A5QWTxfO8fUq5GyDJeCfOjnyNxZsFFmav2KE4z5FsoMeUIbNTjwiFMqeqzObr1JKJi+l/wybgKEfZ0ijbMGaynfEIWbFPlIKxYc1YkZdHcKzeG6yWNxXCtXEK1JJ7pbo9DRcaOWuj2xFBD/Dnasizc7fJOPnPy2JROHmlDyajxz/UavGjFNAmBh5iegAisNexrSoGihG/r5GiY9xP1wCP860nC3RWN6Sxzbb7fCZJvqKuXuPCm8d6KjyrXV7v0PPlrhFfekdviE0dg4f2h/ZGN4dZ4rq7N+qxCw=="
	tt, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(err)
	}

	domainArr := strings.Split(string(tt), "\n")
	fmt.Println("len:", len(domainArr), len(str))
	for idx, dom := range domainArr {
		fmt.Println(idx, dom)
	}
}

func test6() {
	ptr, _ := net.LookupAddr("155.138.201.205")
	for _, ptrvalue := range ptr {
		fmt.Println(ptrvalue)
	}
}

func test5() {
	domains := []string{
		"192.168.0.1",
		"amazon.co.uk",
		"books.amazon.co.uk",
		"www.books.amazon.co.uk",
		"amazon.com",
		"",
		"example0.debian.net",
		"example1.debian.org",
		"",
		"golang.dev",
		"golang.net",
		"play.golang.org",
		"gophers.in.space.museum",
		"",
		"0emm.com",
		"a.0emm.com",
		"b.c.d.0emm.com",
		"",
		"there.is.no.such-tld",
		"",
		// Examples from the PublicSuffix function's documentation.
		"foo.org",
		"foo.co.uk",
		"foo.dyndns.org",
		"foo.blogspot.co.uk",
		"cromulent",
	}

	for _, domain := range domains {
		if domain == "" {
			fmt.Println(">")
			continue
		}

		eTLD, _ := publicsuffix.EffectiveTLDPlusOne(domain)
		fmt.Printf("> %24s%16s \n", domain, eTLD)
		//eTLD, icann := publicsuffix.PublicSuffix(domain)

		// Only ICANN managed domains can have a single label. Privately
		// managed domains must have multiple labels.
		//manager := "Unmanaged"
		//if icann {
		//	manager = "ICANN Managed"
		//} else if strings.IndexByte(eTLD, '.') >= 0 {
		//	manager = "Privately Managed"
		//}
		//
		//fmt.Printf("> %24s%16s  is  %s\n", domain, eTLD, manager)
	}
}

func test4() {
	resp, err := http.Get("https://raw.githubusercontent.com/youpipe/ypctorrent/master/gfw.torrent")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	buf, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		fmt.Println("Update GFW list err:", e)
		return
	}

	domains, err := base64.StdEncoding.DecodeString(string(buf))
	if err != nil {
		fmt.Println("Update GFW list err:", e)
		return
	}
	fmt.Println(string(domains))
}

func test3() {
	l, _ := net.ListenTCP("tcp", &net.TCPAddr{
		Port: 51415,
	})
	fmt.Println(l.Addr().String())
}
func test1() {
	decodeBytes, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}

	ip4 := &layers.IPv4{}
	tcp := &layers.TCP{}
	udp := &layers.UDP{}
	dns := &layers.DNS{}
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, ip4, tcp, udp, dns)
	decodedLayers := make([]gopacket.LayerType, 0, 4)
	if err := parser.DecodeLayers(decodeBytes, &decodedLayers); err != nil {
		panic(err)
	}

	for _, typ := range decodedLayers {
		switch typ {
		case layers.LayerTypeDNS:

			for _, ask := range dns.Questions {
				fmt.Printf("	question:%s-%s-%s\n", ask.Name, ask.Class.String(), ask.Type.String())
			}

			for _, as := range dns.Answers {
				fmt.Println("	Answer:", as.String())
			}
			break
		case layers.LayerTypeIPv4:
			fmt.Println("	IPV4", ip4.SrcIP, ip4.DstIP)
			break
		case layers.LayerTypeTCP:
			fmt.Println("	TCP", tcp.SrcPort, tcp.DstPort)
			break
		case layers.LayerTypeUDP:
			fmt.Println("	UDP", udp.SrcPort, udp.DstPort)
			break
		}
	}
}
