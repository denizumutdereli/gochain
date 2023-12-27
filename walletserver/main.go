package main

import (
	"flag"
	"log"
	"main/walletserver/server"
)

func init() {
	log.SetPrefix("Wallet Server: ")
}

func main() {
	port := flag.Uint("port", 8080, "TCP Port for Wallet Server")
	gateway := flag.String("gateway", "http://localhost:3000", "Blockchain Gateway")
	flag.Parse()

	app := server.NewWalletServer(uint16(*port), *gateway)

	app.Run()
}
