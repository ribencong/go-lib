package main

import "C"
import (
	"fmt"
	"github.com/youpipe/go-youPipe/service/client"
	"os"
)

var conf = &client.Config{
	Addr:        "YPAPwe9SGxqozFTMX42B6isP6zFqYJufYb1hWzuT2bbjJZ",
	Cipher:      "F5cxaiyXXyrGavFB2nU8tcKSp1h2kieaKceMrzAr7ffg5rcHZJta2BgohbYR1NExpRu95YCLVoBH1YL7C8iwoHjTwKmRid4SBj6vmqUVuDbED",
	LocalServer: ":51080",
	License:     `{"Signature":"fzfCN5AOCB0BdFQjGWipq/nC2buv6yF+qr41sc6pjKMsqm7zA1qQ0SjJvGDrXbNmupkV1gR1Odfe2npUxej6Dw==","StartDate":"2019-04-18T11:03:36.886863+08:00","EndDate":"2019-04-25T11:03:36.886863+08:00","UserAddr":"YPAPwe9SGxqozFTMX42B6isP6zFqYJufYb1hWzuT2bbjJZ"}`,
	Services: []string{"YPBzFaBFv8ZjkPQxtozNQe1c9CvrGXYg4tytuWjo9jiaZx@192.168.1.108",
		"YPdtVMDDdgHNTQKbJy447puv68zLjiuFdzfDEwXVtS11H@192.168.103.101",
		"YPBzFaBFv8ZjkPQxtozNQe1c9CvrGXYg4tytuWjo9jiaZx@10.130.147.145"},
}

func main() {
	cli, err := client.NewClient(conf, "12345678")
	if err != nil {
		panic(err)
	}

	if err := cli.Running(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
