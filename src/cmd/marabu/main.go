package main

import (
	"fmt"
	"marabu/internal/bootstrap"
	"marabu/internal/core"
	"marabu/internal/logs"
	"marabu/internal/peer"
	"marabu/internal/ui"
	"os"
	"path/filepath"
)

func main() {

	logFile := logs.InitLogs()
	defer logFile.Close()

	PEERS_FILE := filepath.Join(".", "db", "peers.csv")
	peersFile, err := os.OpenFile(PEERS_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error creating peers file: %v\n", err)
		os.Exit(1)
	}
	defer peersFile.Close()

	DB_PATH := filepath.Join(".", "db")

	Manager := core.NewManager(DB_PATH)

	go Manager.CleanupPendingBlocks(peer.NotifyPeerUnfindable)

	bootstrap.StartNode(Manager)
	ui.Start(Manager)

}
