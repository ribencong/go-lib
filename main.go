package main

import "C"
import (
	"fmt"
	"log"
	"os"
)

func main() {
	startService(os.Args[1], os.Args[2])
	<-make(chan struct{})
}

var currentService *Node = nil
var logger = log.New(os.Stdout, "main", 1)

//export startService
func startService(ls, rs string) bool {
	if currentService != nil && currentService.IsRunning() {
		stopService()
	}
	lServe := fmt.Sprintf("%s", ls)
	rServer := fmt.Sprintf("%s", rs)
	logger.Printf("start service:%s<->%s", lServe, rServer)
	if currentService = NewNode(lServe, rServer); currentService == nil {
		return false
	}

	go currentService.Serving()
	return true
}

//export stopService
func stopService() bool {

	if currentService == nil {
		return true
	}

	defer logger.Print("stop service success.....")
	currentService.Stop()
	currentService = nil
	return true
}
