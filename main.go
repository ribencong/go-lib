package main

import "C"
import (
	"fmt"
	"github.com/youpipe/go-youPipe/service/client"
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
	clientMain()
}
func test1() {
	LibVerifyLicense(`{"sig":"vQlEcc5XKX7B2Qxtwln4B6oijiUEUnI1DlI30hQEhELW1IUpVFvr2kTDunOrD2tWn39WagM3gk4trBx+jq5kAA==","start":"2019-04-25 09:09:54","end":"2019-05-05 09:09:54","user":"YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ"}`)
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
