package main

import (
	"fmt"
	"log"
	"main/block"
	"main/config"
	"main/wallet"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	// w := wallet.NewWallet()

	// fmt.Println(w.PrivateKeyStr())
	// fmt.Println(w.PublicKeyStr())
	// fmt.Println(w.BlockchainAddress())

	// t := wallet.NewTransaction(w.PrivateKey(), w.PublicKey(), w.BlockchainAddress(), "B", block.Token{})

	// fmt.Printf("signature: %s\n", t.GenerateSignature())

	initialWallet := wallet.NewWallet()
	// walletA := wallet.NewWallet()
	// walletB := wallet.NewWallet()

	// Wallet
	// t := wallet.NewTransaction(walletA.PrivateKey(), walletA.PublicKey(), walletA.BlockchainAddress(), walletB.BlockchainAddress(), block.Token{TokenName: "ETH", TokenValue: utils.FloatToDecimal(0.6)})

	// Blockchain
	blockchain, err := block.CreateBlockchain(initialWallet.BlockchainAddress(), conf.Blockchain)
	if err != nil {
		panic(err)
	}
	// fmt.Println(err)
	// isAdded := blockchain.AddTransaction(walletA.BlockchainAddress(), walletB.BlockchainAddress(), block.Token{TokenName: "ETH", TokenValue: utils.FloatToDecimal(0.6)},
	// 	walletA.PublicKey(), t.GenerateSignature())
	// fmt.Println("Added? ", isAdded)

	blockchain.Mining()
	blockchain.Print()

	fmt.Printf("M %s\n", blockchain.CalculateTotalAmount(initialWallet.BlockchainAddress(), "CRY"))
}
