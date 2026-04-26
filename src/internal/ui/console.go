//go:build cli

package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"marabu/internal/core"
	"marabu/internal/peer"
	"marabu/internal/types"
)

// Pass the Manager in so the CLI can query the DB and start new clients
func Start(manager *core.Manager) {
	fmt.Println("Marabu CLI started. Type 'help' for commands.")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // Exit on EOF (Ctrl+D)
		}

		input := scanner.Text()
		args := strings.Fields(input) // Splits by spaces
		if len(args) == 0 {
			continue
		}

		cmd := strings.ToLower(args[0])

		switch cmd {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  info, i, ?            - Show node diagnostics and chaintip")
			fmt.Println("  peers [list]          - List detailed connected peers")
			fmt.Println("  peers add <ip:port>   - Manually connect to a node")
			fmt.Println("  peers drop <ip:port>  - Disconnect from a node")
			fmt.Println("  objects get <hash>    - Fetch and display an object from the database")
			fmt.Println("  network sync          - Force broadcast GetPeers and GetChainTip")
			fmt.Println("  exit, quit, q         - Exit the CLI")

		case "info", "?", "i":
			// Fetch networking stats
			icnt, ocnt, bcnt := peer.ConnManager.GetCounts()

			fmt.Println("--- Node Status ---")
			fmt.Printf("Peers:     %d Total (%d Inbound | %d Outbound | %d VIP)\n", (icnt + ocnt), icnt, ocnt, bcnt)

			// Fetch blockchain state from DB
			tip, height, err := manager.GetChaintip()
			if err != nil {
				fmt.Println("Chaintip:  [None / Genesis]")
			} else {
				fmt.Printf("Chaintip:  %s\n", tip)
				fmt.Printf("Height:    %d\n", height)
			}
			fmt.Println("-------------------")

		case "peers", "p":
			if len(args) == 1 || args[1] == "list" {
				listPeers()
			} else if args[1] == "add" && len(args) == 3 {
				connectToPeer(args[2], manager)
			} else if args[1] == "drop" && len(args) == 3 {
				disconnectPeer(args[2])
			} else {
				fmt.Println("Usage: peers [list | add <ip:port> | drop <ip:port>]")
			}

		case "objects", "o":
			if len(args) == 3 && args[1] == "get" {
				inspectObject(args[2], manager)
			} else if len(args) == 2 && args[1] == "list" {
				listObjects(manager) // NEW
			} else {
				fmt.Println("Usage: objects [list | get <hash>]")
			}

		case "network":
			if len(args) == 2 && args[1] == "sync" {
				fmt.Println("Forcing network sync...")
				peer.BroadcastGetPeers()
				peer.BroadcastGetChainTip()
				fmt.Println("Sync requests broadcasted.")
			} else {
				fmt.Println("Usage: network sync")
			}

		case "exit", "quit", "q":
			fmt.Println("Exiting CLI...")
			os.Exit(0)

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}

// -- Helper Functions --

func listPeers() {
	allPeers := peer.ConnManager.GetAll()
	if len(allPeers) == 0 {
		fmt.Println("No connected peers.")
		return
	}

	fmt.Printf("%-20s | %-22s | %-10s | %-10s \n", "AGENT", "ADDRESS", "ORIGIN", "PERSISTENT")
	fmt.Println(strings.Repeat("-", 70))
	for _, p := range allPeers {
		vipStatus := ""
		if p.IsPersistent() {
			vipStatus = "*"
		}

		agent := p.Agent()
		if agent == "" {
			agent = "Unknown"
		}

		// If someone has a ridiculously long agent name cap it so it doesn't break table formatting
		if len(agent) > 20 {
			agent = agent[:17] + "..."
		}

		fmt.Printf("%-20s | %-22s | %-10s | %-10s \n", agent, p.Addr(), p.Origin(), vipStatus)
	}
}

func connectToPeer(addr string, manager *core.Manager) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Println("Error: Invalid address format. Use <ip>:<port>")
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("Error: Port must be a number.")
		return
	}

	fmt.Printf("Dialing %s:%d...\n", host, port)
	err = peer.StartClient(host, port, false, manager)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
	} else {
		fmt.Println("Connection initiated successfully.")
	}
}

func disconnectPeer(addr string) {
	p, exists := peer.ConnManager.Exists(addr)
	if !exists {
		fmt.Printf("Error: No active connection found for '%s'\n", addr)
		return
	}

	p.Disconnect()
	fmt.Printf("Dropped connection to %s\n", addr)
}

func inspectObject(hashStr string, manager *core.Manager) {
	hash := types.HashID(hashStr)

	obj, err := manager.GetObject(hash)
	if err != nil {
		fmt.Printf("Error: Could not find object %s in database.\n", hashStr)
		return
	}

	// Pretty-print the JSON output so it's actually readable in the terminal!
	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting object: %v\n", err)
		return
	}

	fmt.Printf("--- Object %s ---\n", hashStr)
	fmt.Println(string(prettyJSON))
	fmt.Println("----------------------------------------------------------------")
}

func listObjects(manager *core.Manager) {
	ids, err := manager.GetAllObjectIDs()
	if err != nil {
		fmt.Printf("Error scanning database: %v\n", err)
		return
	}

	if len(ids) == 0 {
		fmt.Println("No objects found in the database.")
		return
	}

	var blocks []types.HashID
	var txs []types.HashID
	var unknown []types.HashID

	fmt.Println("Scanning database...")

	// Sort the objects into buckets
	for _, id := range ids {
		obj, err := manager.GetObject(id)
		if err != nil {
			unknown = append(unknown, id)
			continue
		}

		switch obj.ObjectType() {
		case types.OBJ_BLOCK:
			blocks = append(blocks, id)
		case types.OBJ_TRANSACTION:
			txs = append(txs, id)
		default:
			unknown = append(unknown, id)
		}
	}

	// Print the formatted results
	fmt.Println("\n=== Stored Objects ===")
	fmt.Printf("Total: %d\n\n", len(ids))

	fmt.Printf("Blocks (%d):\n", len(blocks))
	for _, b := range blocks {
		fmt.Printf("  | %s\n", b)
	}

	fmt.Printf("\nTransactions (%d):\n", len(txs))
	for _, t := range txs {
		fmt.Printf("  | %s\n", t)
	}

	if len(unknown) > 0 {
		fmt.Printf("\nUnknown/Corrupted (%d):\n", len(unknown))
		for _, u := range unknown {
			fmt.Printf("    %s\n", u)
		}
	}
	fmt.Println("======================")
}
