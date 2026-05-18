//go:build cli

package ui

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/peer"
	"marabu/internal/types"
	"marabu/internal/utils"
	"marabu/internal/wallet"
)

// ANSI Color Codes for the CLI
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

func Start(manager *core.Manager, wallet *wallet.Wallet) {
	fmt.Print("\033[2J")
	go startLivePanel(manager)
	fmt.Print("\033[7;1H")

	fmt.Printf("%sMarabu CLI started. Type 'help' for commands.%s\n", bold+cyan, reset)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(bold + green + "> " + reset)

		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		executeCommand(strings.ToLower(args[0]), args, manager, wallet)
	}
}

func executeCommand(cmd string, args []string, manager *core.Manager, wallet *wallet.Wallet) {
	switch cmd {
	case "help", "?", "h":
		printHelp()

	case "info", "i", "status":
		printInfo(manager)

	case "id", "identity", "miner":
		printIdentity(manager)

	case "peers", "p":
		listPeers()

	case "connect":
		if len(args) == 2 {
			connectToPeer(args[1], manager)
		} else {
			fmt.Printf("%sUsage: connect <ip:port>%s\n", red, reset)
		}

	case "disconnect", "drop":
		if len(args) == 2 {
			disconnectPeer(args[1])
		} else {
			fmt.Printf("%sUsage: disconnect <ip:port>%s\n", red, reset)
		}

	case "muted":
		listMuted()

	case "mute":
		if len(args) == 2 && args[1] == "list" {
			listMuted()
		} else if len(args) >= 2 {
			val := strings.Join(args[1:], " ")
			peer.ConnManager.MutePeer(val)
			fmt.Printf("%sMuted logs matching:%s %s\n", yellow, reset, val)
		} else {
			fmt.Printf("%sUsage: mute <ip or agent> OR mute list%s\n", red, reset)
		}

	case "unmute":
		if len(args) >= 2 {
			val := strings.Join(args[1:], " ")
			if val == "*" {
				cleared := peer.ConnManager.UnmuteAll()
				fmt.Printf("%sCleared %d mute rules. All peers unmuted.%s\n", green, cleared, reset)
			} else {
				peer.ConnManager.UnmutePeer(val)
				fmt.Printf("%sUnmuted logs matching:%s %s\n", green, reset, val)
			}
		} else {
			fmt.Printf("%sUsage: unmute <ip or agent> OR unmute *%s\n", red, reset)
		}

	case "banned":
		listBanned()

	case "ban":
		if len(args) == 2 && args[1] == "list" {
			listBanned()
		} else if len(args) >= 2 {
			val := strings.Join(args[1:], " ")
			peer.ConnManager.BanPeer(val)
			fmt.Printf("%sBanned and kicked target matching:%s %s\n", red, reset, val)
		} else {
			fmt.Printf("%sUsage: ban <ip or agent> OR ban list%s\n", red, reset)
		}

	case "unban":
		if len(args) >= 2 {
			val := strings.Join(args[1:], " ")
			if val == "*" {
				cleared := peer.ConnManager.UnbanAll()
				fmt.Printf("%sCleared %d ban rules. All peers unbanned.%s\n", green, cleared, reset)
			} else {
				peer.ConnManager.UnbanPeer(val)
				fmt.Printf("%sUnbanned target matching:%s %s\n", green, reset, val)
			}
		} else {
			fmt.Printf("%sUsage: unban <ip or agent> OR unban *%s\n", red, reset)
		}

	case "cores", "c", "miners", "m":
		if len(args) == 2 {
			cores, err := strconv.Atoi(args[1])
			if err != nil || cores < 0 {
				fmt.Printf("%sError: Cores must be a positive integer.%s\n", red, reset)
				return
			}

			maxCores := runtime.NumCPU()
			if cores > maxCores {
				fmt.Printf("%sWarning: Requested %d cores, but system only has %d. Capping at %d.%s\n",
					yellow, cores, maxCores, maxCores, reset)
				cores = maxCores
			}

			manager.SetMiningCores(cores)

			if cores == 0 {
				fmt.Printf("%sMiner paused. Using 0 cores.%s\n", yellow, reset)
			} else {
				fmt.Printf("%sMiner restarted with %d cores.%s\n", green, cores, reset)
			}
		} else {
			fmt.Printf("%sUsage: cores <number>%s\n", red, reset)
		}

	case "note":
		if len(args) >= 2 {
			if args[1] == "new" || args[1] == "set" {
				fmt.Printf("%sEnter new note (max 256 chars):%s ", yellow, reset)
				noteScanner := bufio.NewScanner(os.Stdin)
				if noteScanner.Scan() {
					newNote := noteScanner.Text()
					if len(newNote) > 256 {
						fmt.Printf("%sError: Note cannot exceed 256 characters.%s\n", red, reset)
						return
					}
					manager.SetNote(types.BuString(newNote))
					fmt.Printf("%sNote updated successfully.%s\n", green, reset)
				} else {
					fmt.Printf("%sError reading note input.%s\n", red, reset)
				}
			}
		} else {
			note := manager.Config().Note
			if note == "" {
				fmt.Printf("%sNo note set.%s\n", yellow, reset)
			} else {
				fmt.Printf("%sCurrent note:%s %s\n", cyan, reset, note)
			}
		}

	case "alias":
		if len(args) < 2 {
			fmt.Println("Usage:")
			fmt.Println("  alias add <name> <pubkey>")
			fmt.Println("  alias list")
			return
		}

		switch args[1] {
		case "add":
			if len(args) != 4 {
				fmt.Println("Usage: alias add <name> <pubkey>")
				return
			}
			name := args[2]
			pubkey := types.HashID(args[3])
			wallet.AddAlias(name, pubkey)
			fmt.Printf("Alias saved: %s -> %s\n", name, pubkey)

		case "list":
			aliases := wallet.GetAliases()
			if len(aliases) == 0 {
				fmt.Println("Address book is empty.")
				return
			}
			fmt.Println("\n=== Address Book ===")
			for name, pubkey := range aliases {
				fmt.Printf("%-10s : %s\n", name, pubkey)
			}
			fmt.Println("====================")
		}

	case "sweep":
		fmt.Printf("Scanning the blockchain for unknown public keys...\n")
		found := wallet.AutoAliasUTXOs()
		fmt.Printf("Sweep complete. Found and aliased %d new unique public keys.\n", found)
		fmt.Printf("Type 'alias list' to see them!\n")

	case "balance":
		balance, utxos := wallet.GetBalance()

		// Convert Picabus to BU for display
		buValue := utils.PicabuToBu(balance)

		fmt.Printf("\n=== Wallet Balance ===\n")
		fmt.Printf("Confirmed UTXOs: %d\n", len(utxos))
		fmt.Printf("Total Balance:   %.6f BU (%s Picabus)\n", buValue, balance.String())
		fmt.Printf("======================\n\n")

	case "send":
		if len(args) != 3 {
			fmt.Printf("Usage: send <target_pubkey> <amount_in_BU>\n")
			return
		}

		targetPubkey := wallet.ResolveAddress(args[1])

		amountBU, err := strconv.ParseFloat(args[2], 64)
		if err != nil || amountBU <= 0 {
			fmt.Printf("Error: Amount must be a positive number.\n")
			return
		}

		// Convert BU to Picabu
		amountPicabu := utils.BuToPicabu(amountBU)

		// Set a flat network fee (e.g., 0.001 BU)
		feePicabu := types.NewPicabu(uint64(0.001 * 1_000_000_000_000))

		fmt.Printf("Building transaction...\n")

		tx, err := wallet.SendPicabus(targetPubkey, amountPicabu, feePicabu)
		if err != nil {
			fmt.Printf("Transaction Failed: %v\n", err)
			return
		}

		// Safely get the ID using our smart wrapper!
		txid, _ := crypto.GetObjectID(tx)
		fmt.Printf("Transaction broadcasted successfully!\n")
		fmt.Printf("TXID: %s\n", txid)

	case "objects", "o":
		listObjects(manager)

	case "get":
		if len(args) == 2 {
			inspectObject(args[1], manager)
		} else {
			fmt.Printf("%sUsage: get <hash>%s\n", red, reset)
		}

	case "sync":
		fmt.Printf("%sForcing network sync...%s\n", yellow, reset)
		peer.BroadcastGetPeers()
		peer.BroadcastGetChainTip()
		fmt.Printf("%sSync requests broadcasted.%s\n", green, reset)

	case "exit", "quit", "q":
		fmt.Printf("%sExiting CLI...%s\n", cyan, reset)
		os.Exit(0)

	default:
		fmt.Printf("%sUnknown command: %s%s\n", red, cmd, reset)
	}
}
