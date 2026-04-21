//go:build cli

package ui

import (
	"fmt"
)

func Start() {
	fmt.Println("Marabu CLI started. Type 'help' for commands.")
	for {
		var cmd string
		fmt.Print("> ")
		fmt.Scanln(&cmd)

		switch cmd {
		case "":
			// Ignore empty input
			continue
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  peers - List connected peers")
			fmt.Println("  objects - List stored objects")
			fmt.Println("  exit - Exit the CLI")
		case "peers":
			fmt.Println("dummy command")
			// logs.ClientLog("", fmt.Sprintf("Connected peers: %d", len(ConnManager.GetCounts())), -1)
		case "objects":
			fmt.Println("dummy command")
			// logs.ClientLog("", fmt.Sprintf("Stored objects: %d", object.GetObjectCount()), -1)
		case "exit":
			fmt.Println("Exiting CLI...")
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}
