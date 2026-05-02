package main

import (
	"encoding/json"
	"fmt"
	"log"

	"marabu/internal/crypto"
	"marabu/internal/types"
)

func main() {

	genesisJSON := `{
		"T": "00000000abc00000000000000000000000000000000000000000000000000000",
		"created": 1771159355,
		"miner": "Marabu",
		"nonce": "00dd82159556175752d9ba7349df67bddd237b59183747383f7b720e85c32347",
		"note": "Financial Times 2026-02-13: Crypto's battle with the banks is splitting Trump's base",
		"previd": null,
		"txids": [],
		"type": "block"
	  }`

	var genesisBlock types.Block
	if err := json.Unmarshal([]byte(genesisJSON), &genesisBlock); err != nil {
		log.Fatalf("Failed to parse genesis JSON: %v", err)
	}

	hash, err := crypto.HashObject(&genesisBlock)
	if err != nil {
		log.Fatalf("Failed to hash genesis block: %v", err)
	}

	fmt.Printf("Genesis Block Hash: %s\n", hash)
	
	isValid, _ := crypto.VerifyPoW(hash)
	fmt.Printf("Is Proof of Work valid? %t\n", isValid)
}