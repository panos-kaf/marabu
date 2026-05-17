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
	"marabu/internal/peer"
)

func Start(manager *core.Manager) {
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

		executeCommand(strings.ToLower(args[0]), args, manager)
	}
}

func executeCommand(cmd string, args []string, manager *core.Manager) {
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
