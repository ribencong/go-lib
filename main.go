package main

import "C"
import (
	"fmt"
	"github.com/youpipe/go-youPipe/service/client"
	"os"
)

var conf = &client.Config{
	Addr:        "YP7Bdx1LixC9yBnnmoJky4E4QsKxjCdhjvKfF64JxjRJfR",
	Cipher:      "2C9fRZqk3SE73w7WEsBnodCCg4yNipv9pJEgyWEjG9ZKWnXUDwD1XfSLtmo9sLSy1sdHMpvvQ2KFgqQjYWKJSUJRM75hoWvaYr34suqdMP1kzP",
	LocalServer: ":51080",
	License:     `{"sig":"VF2XU6t/gXSp1rSnMUMlYrvRJ3KMAmDHKmzP9ZkaM8rOb97CAdksr49bSe7uQ40mS7yxpyM+RxzmoWWlbhLhCQ==","start":"2019-04-24 09:04:00","end":"2019-05-04 09:04:00","user":"YP7Bdx1LixC9yBnnmoJky4E4QsKxjCdhjvKfF64JxjRJfR"}`,
}

func main() {
	clientMain()
}
func test1() {
	LibVerifyLicense(`{"sig":"dzs4In4HGYcgZbAp0cskbh8gvCjZDNcqOTBOHN6+3DSiqZiUYk4Mb4g2CoIBvBJojSTh7JdUNPpp8fPbMwRtAQ==","start":"2019-04-23 08:34:21","end":"2019-05-03 08:34:21","user":"YPDsDm5RBqhA14dgRUGMjE4SVq7A3AzZ4MqEFFL3eZkhjZ"}
`)
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
