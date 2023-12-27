package main

import (
	"flag"
	"log"
	"main/config"
	"main/server/app"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	port := flag.Uint("port", 3000, "TCP port number for Blockchain server")
	flag.Parse()
	app := app.NewBlockchainServer(uint16(*port), conf.Blockchain)
	app.Connect()
}
