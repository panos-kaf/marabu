package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"marabu/internal/bootstrap"
	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/logs"
	"marabu/internal/miner"
	"marabu/internal/peer"
	"marabu/internal/types"
	"marabu/internal/ui"
	"os"
	"path/filepath"
	"runtime"
)

func main() {

	defaultCores := max(runtime.NumCPU()-1, 1)

	var config core.NodeConfig
	var agentStr string

	flag.IntVar(&config.ServerPort, "port", 18018, "The port to listen on")
	flag.IntVar(&config.ServerPort, "p", 18018, "Alias for --port")

	flag.IntVar(&config.MiningCores, "cores", defaultCores, "Number of CPU cores to use for mining (default: all cores minus one)")

	flag.StringVar(&agentStr, "agent", "marabobos", "Agent name")
	flag.StringVar(&agentStr, "a", "marabobos", "Alias for --agent")

	flag.Parse()

	logFile := logs.InitLogs()
	defer logFile.Close()

	err := os.MkdirAll("./db", 0755)
	if err != nil {
		fmt.Printf("Error creating db directory: %v\n", err)
		os.Exit(1)
	}

	PEERS_FILE := filepath.Join(".", "db", "peers.csv")
	peersFile, err := os.OpenFile(PEERS_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error creating peers file: %v\n", err)
		os.Exit(1)
	}
	defer peersFile.Close()

	config.DBPath = filepath.Join(".", "db")

	config.AgentName = types.BuString(agentStr)

	Manager := core.NewManager(config)

	Manager.InitializeMempool()

	go Manager.CleanupPendingBlocks(peer.NotifyPeerUnfindable)

	go Manager.SyncNodeState(peer.BroadcastGetMempool)

	go peer.StartServer(Manager)

	privKey, err := crypto.LoadOrGenerateKey("db/node.priv")
	if err != nil {
		panic("Failed to load node keys!")
	}

	// Extract the public key and cast it to your HashID type
	pubKeyBytes := privKey.Public().(ed25519.PublicKey)
	myPubKey := types.HashID(hex.EncodeToString(pubKeyBytes))

	Miner := miner.NewMiner(Manager, myPubKey)
	go Miner.StartMining()

	bootstrap.StartNode(Manager)
	ui.Start(Manager)

}
