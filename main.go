package main

import "C"
import (
	"encoding/base64"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ribencong/go-lib/client"
	"golang.org/x/net/proxy"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var conf = &client.Config{
	Addr:        "YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ",
	Cipher:      "GffT4JanGFefAj4isFLYbodKmxzkJt9HYTQTKquueV8mypm3oSicBZ37paYPnDscQ7XoPa4Qgse6q4yv5D2bLPureawFWhicvZC5WqmFp9CGE",
	LocalServer: ":51080",
	SettingUrl:  "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent",
	License:     `{"sig":"vQlEcc5XKX7B2Qxtwln4B6oijiUEUnI1DlI30hQEhELW1IUpVFvr2kTDunOrD2tWn39WagM3gk4trBx+jq5kAA==","start":"2019-04-25 09:09:54","end":"2019-05-05 09:09:54","user":"YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ"}`,
}

func main() {
	test8()
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

func clientMain() {
	cli, err := client.NewClient(conf, "12345678")
	if err != nil {
		panic(err)
	}

	if err := cli.Running(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func test2() {
	conn, err := net.Listen("tcp", ":1080")
	if err != nil {
		panic(err)
	}

	go func() {

		for {
			c, e := conn.Accept()
			if e != nil {
				panic(e)
			}

			go func() {
				obj, err := client.ProxyHandShake(c)
				if err != nil {
					fmt.Println("\nSock5 handshake err:->", err)
					return
				}
				fmt.Println("\nProxy handshake success:", obj.Target)
				c.Write([]byte("Response"))
				buf := make([]byte, 1024)
				n, e := c.Read(buf)
				if e != nil {
					panic(e)
				}
				fmt.Printf("\n\nServer:%s\n\n", buf[:n])
				time.Sleep(time.Second)
				c.Close()
			}()
		}
	}()

	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, nil)
	if err != nil {
		panic(err)
	}

	c, err := dialer.Dial("tcp", "nbsio.net:80")
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 1024)
	n, e := c.Read(buf)
	if e != nil {
		panic(e)
	}
	fmt.Printf("\n\nClient:%s\n\n", buf[:n])

	if _, e := c.Write([]byte("Get")); e != nil {
		print(e)
	}

	time.Sleep(25 * time.Second)
}
