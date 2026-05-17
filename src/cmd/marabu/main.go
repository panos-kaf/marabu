package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"marabu/internal/bootstrap"
	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/logs"
	"marabu/internal/miner"
	"marabu/internal/peer"
	"marabu/internal/types"
	"marabu/internal/ui"

	"github.com/joho/godotenv"
)

// ParseConfig handles .env loading, flag parsing, and strict type conversion
func ParseConfig() core.NodeConfig {
	var config core.NodeConfig

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to system environment variables or defaults.")
	}

	defaultPort := 18018
	portStr := os.Getenv("PORT")

	if portStr != "" {
		if parsedPort, err := strconv.Atoi(portStr); err == nil && parsedPort > 0 && parsedPort <= 65535 {
			defaultPort = parsedPort
		} else {
			log.Printf("Warning: Invalid PORT value '%s' in .env, falling back to %d\n", portStr, defaultPort)
		}
	}

	defaultAgent := os.Getenv("AGENT")
	if defaultAgent == "" {
		defaultAgent = "marabobos"
	}

	defaultStudentIDs := os.Getenv("STUDENT_IDS")

	defaultCores := runtime.NumCPU() - 1
	if defaultCores < 1 {
		defaultCores = 1
	}

	var agentStr string
	var studentIDsStr string

	flag.IntVar(&config.ServerPort, "port", defaultPort, "The port to listen on")
	flag.IntVar(&config.ServerPort, "p", defaultPort, "Alias for --port")

	flag.IntVar(&config.MiningCores, "cores", defaultCores, "Number of CPU cores to use for mining")
	flag.IntVar(&config.MiningCores, "c", defaultCores, "Alias for --cores")

	flag.StringVar(&agentStr, "agent", defaultAgent, "Agent name")
	flag.StringVar(&agentStr, "a", defaultAgent, "Alias for --agent")

	flag.StringVar(&studentIDsStr, "studentids", defaultStudentIDs, "Comma-separated list of student IDs")
	flag.StringVar(&studentIDsStr, "s", defaultStudentIDs, "Alias for --studentids")

	flag.Parse()

	config.AgentName = types.BuString(agentStr)
	config.DBPath = filepath.Join(".", "db")

	if studentIDsStr != "" {
		splitIDs := strings.Split(studentIDsStr, ",")
		for _, id := range splitIDs {
			cleanID := strings.TrimSpace(id)
			if cleanID != "" {
				config.StudentIDs = append(config.StudentIDs, types.BuString(cleanID))
			}
		}
	}

	return config
}

func main() {

	logFile := logs.InitLogs()
	defer logFile.Close()

	config := ParseConfig()

	err := os.MkdirAll(config.DBPath, 0755)
	if err != nil {
		fmt.Printf("Error creating db directory: %v\n", err)
		os.Exit(1)
	}

	peersFilePath := filepath.Join(config.DBPath, "peers.csv")
	peersFile, err := os.OpenFile(peersFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error creating peers file: %v\n", err)
		os.Exit(1)
	}
	peersFile.Close()

	// Boot Sequence
	Manager := core.NewManager(config)
	Manager.InitializeMempool()

	go Manager.CleanupPendingBlocks(peer.NotifyPeerUnfindable)
	go Manager.SyncNodeState(peer.BroadcastGetMempool)
	go peer.StartServer(Manager)

	// Load secret key
	secretKey, err := crypto.LoadOrGenerateKey(filepath.Join(config.DBPath, "node.priv"))
	if err != nil {
		panic(fmt.Sprintf("Failed to load node key: %v", err))
	}

	pubKeyBytes := secretKey.Public().(ed25519.PublicKey)
	myPubKey := types.HashID(hex.EncodeToString(pubKeyBytes))

	Miner := miner.NewMiner(Manager, myPubKey)
	go Miner.StartMining()

	bootstrap.StartNode(Manager)

	ui.Start(Manager)
}
