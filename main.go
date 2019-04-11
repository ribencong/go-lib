package main

import "C"
import (
	"fmt"
	"os"
)

func main() {
	test5()
}

func test5() {
	println("LibVerifyLicense:->", LibVerifyLicense(`{"sig":"fgmReXNZGEuRlwzvkBHXbjV+pwVdpe75KLCLVvFdkknA5k7FfWLTFk50q1FriX2lbt1pTHFtz7+OmwOukciyCQ==","data":{"StartTime":"2019-04-10 19:26:13","EndTime":"2019-04-14 19:26:13","UserAddr":"YP7Bdx1LixC9yBnnmoJky4E4QsKxjCdhjvKfF64JxjRJfR"}}`))
}

func test4() {
	b := LibInitAccount(os.Args[6], os.Args[5], os.Args[4])
	fmt.Printf("unlock:%t\n", b)
	if !b {
		panic("unlock failed")
	}
	b = LibStartService(os.Args[1], os.Args[2], os.Args[3])
	if !b {
		panic("service failed")
	}
	<-make(chan struct{})
}
func test1() {
	b := LibStartService(os.Args[1], os.Args[2], os.Args[3])
	if !b {
		panic("failed")
	}
	<-make(chan struct{})
}
func test2() {
	a, c := LibCreateAccount(os.Args[1])
	fmt.Println(a)
	fmt.Println(c)
}
func test3() {
	b := LibInitAccount(os.Args[1], os.Args[2], os.Args[3])
	fmt.Printf("unlock:%t\n", b)
}
