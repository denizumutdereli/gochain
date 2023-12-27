package main

import (
	"fmt"
	"main/utils"
)

//func utils.FindP2P(host string, port uint16, startIp uint8, endIp uint8, startPort uint16, endPort uint16)

func main() {
	//fmt.Println(utils.FindP2P("127.0.0.1", 5000, 0, 3, 5000, 5003))
	fmt.Println(utils.GetHost())
}
