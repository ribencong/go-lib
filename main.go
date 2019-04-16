package main

import "C"
import "fmt"

func main() {
	test5()
}

func test5() {
	fmt.Println("LibVerifyLicense:->", LibVerifyLicense(`{"sig":"fgmReXNZGEuRlwzvkBHXbjV+pwVdpe75KLCLVvFdkknA5k7FfWLTFk50q1FriX2lbt1pTHFtz7+OmwOukciyCQ==","data":{"StartTime":"2019-04-10 19:26:13","EndTime":"2019-04-14 19:26:13","UserAddr":"YP7Bdx1LixC9yBnnmoJky4E4QsKxjCdhjvKfF64JxjRJfR"}}`))
}
