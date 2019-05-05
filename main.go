package main

import "C"
import (
	"encoding/base64"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/ribencong/go-youPipe/service/client"
	"os"
)

var conf = &client.Config{
	Addr:        "YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ",
	Cipher:      "GffT4JanGFefAj4isFLYbodKmxzkJt9HYTQTKquueV8mypm3oSicBZ37paYPnDscQ7XoPa4Qgse6q4yv5D2bLPureawFWhicvZC5WqmFp9CGE",
	LocalServer: ":51080",
	SettingUrl:  "https://raw.githubusercontent.com/ribencong/ypctorrent/master/ypc_debug.torrent",
	License:     `{"sig":"vQlEcc5XKX7B2Qxtwln4B6oijiUEUnI1DlI30hQEhELW1IUpVFvr2kTDunOrD2tWn39WagM3gk4trBx+jq5kAA==","start":"2019-04-25 09:09:54","end":"2019-05-05 09:09:54","user":"YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ"}`,
}

func main() {
	test1()
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
