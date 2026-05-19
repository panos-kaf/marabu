//go:build cli

package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

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

		executeCommand(strings.ToLower(args[0]), args, manager, wallet, scanner)
	}
}

func executeCommand(cmd string, args []string, manager *core.Manager, w *wallet.Wallet, scanner *bufio.Scanner) {
	switch cmd {
	// System
	case "help", "?", "h":
		cmdHelp()
	case "exit", "quit", "q":
		cmdExit()

	// Node & Hardware
	case "info", "i", "status":
		cmdInfo(manager)
	case "id", "identity", "miner":
		cmdIdentity(manager)
	case "cores", "c", "miners", "m":
		cmdCores(args, manager)
	case "note":
		cmdNote(args, manager, scanner)
	case "stats":
		cmdStats(args, manager, w)

	// Networking
	case "peers", "p":
		cmdPeers()
	case "connect":
		cmdConnect(args, manager)
	case "disconnect", "drop":
		cmdDisconnect(args)
	case "sync":
		cmdSync()

	// Moderation
	case "muted":
		cmdMuted()
	case "mute":
		cmdMute(args)
	case "unmute":
		cmdUnmute(args)
	case "banned":
		cmdBanned()
	case "ban":
		cmdBan(args)
	case "unban":
		cmdUnban(args)

	// Wallet & Transactions
	case "alias":
		cmdAlias(args, w)
	case "sweep":
		cmdSweep(w)
	case "balance":
		cmdBalance(w)
	case "send":
		cmdSend(args, w)

	// Database
	case "objects", "o":
		cmdObjects(manager)
	case "get":
		cmdGet(args, manager)

	default:
		fmt.Printf("%sUnknown command: %s%s\n", red, cmd, reset)
	}
}

func cmdExit() {
	fmt.Printf("%sExiting CLI...%s\n", cyan, reset)
	os.Exit(0)
}

func cmdHelp() {

	fmt.Printf("\n%s=== Marabu CLI Commands ===%s\n", bold+yellow, reset)

	fmt.Printf("\n%s[ Node & Hardware ]%s\n", bold+magenta, reset)
	fmt.Printf("  %sinfo, i, status%s         - Show node diagnostics and chaintip\n", cyan, reset)
	fmt.Printf("  %sid, identity, miner%s     - Show node identity, agent, and student IDs\n", cyan, reset)
	fmt.Printf("  %scores, c <num>%s          - Adjust the number of CPU cores used for mining\n", cyan, reset)
	fmt.Printf("  %snote%s                    - Show current miner note\n", cyan, reset)
	fmt.Printf("  %snote set%s                - Set a custom miner note\n", cyan, reset)
	fmt.Printf("  %sstats [target]%s          - Show total historical blocks mined by a pubkey/alias\n", cyan, reset)

	fmt.Printf("\n%s[ Wallet & Transactions ]%s\n", bold+magenta, reset)
	fmt.Printf("  %sbalance%s                 - Show wallet balance and UTXOs\n", cyan, reset)
	fmt.Printf("  %ssend <target> <BU>%s      - Send funds to a pubkey or alias (fee is auto-calculated)\n", cyan, reset)
	fmt.Printf("  %salias add <name> <key>%s  - Save a public key to your address book\n", cyan, reset)
	fmt.Printf("  %salias list%s              - View your address book\n", cyan, reset)
	fmt.Printf("  %ssweep%s                   - Auto-alias all unknown public keys in the UTXO set\n", cyan, reset)

	fmt.Printf("\n%s[ Networking & Peers ]%s\n", bold+magenta, reset)
	fmt.Printf("  %speers, p%s                - List detailed connected peers\n", cyan, reset)
	fmt.Printf("  %sconnect <ip:port>%s       - Manually connect to a node\n", cyan, reset)
	fmt.Printf("  %sdisconnect <ip:port>%s    - Disconnect from a node (alias: drop)\n", cyan, reset)
	fmt.Printf("  %ssync%s                    - Force broadcast GetPeers and GetChainTip\n", cyan, reset)

	fmt.Printf("\n%s[ Moderation ]%s\n", bold+magenta, reset)
	fmt.Printf("  %smuted%s                   - List all muted targets\n", cyan, reset)
	fmt.Printf("  %smute <target>%s           - Mute logs from a spammy IP or Agent\n", cyan, reset)
	fmt.Printf("  %sunmute <target> | *%s     - Unmute a target (use * for all)\n", cyan, reset)
	fmt.Printf("  %sbanned%s                  - List all currently banned targets\n", cyan, reset)
	fmt.Printf("  %sban <target>%s            - Ban and kick an IP or Agent\n", cyan, reset)
	fmt.Printf("  %sunban <target> | *%s      - Unban a target (use * for all)\n", cyan, reset)

	fmt.Printf("\n%s[ Database ]%s\n", bold+magenta, reset)
	fmt.Printf("  %sobjects, o%s              - List all objects in the database\n", cyan, reset)
	fmt.Printf("  %sget <hash>%s              - Fetch and display a specific object\n", cyan, reset)

	fmt.Printf("\n%s[ System ]%s\n", bold+magenta, reset)
	fmt.Printf("  %shelp, ?, h%s              - Show this menu\n", cyan, reset)
	fmt.Printf("  %sexit, quit, q%s           - Exit the CLI\n\n", cyan, reset)
}

func cmdInfo(manager *core.Manager) {
	icnt, ocnt, bcnt := peer.ConnManager.GetCounts()

	fmt.Printf("\n%s--- Node Status ---%s\n", bold+cyan, reset)
	fmt.Printf("%sPeers:%s     %d Total (%s%d Inbound%s | %s%d Outbound%s | %s%d VIP%s)\n",
		bold, reset, (icnt + ocnt), green, icnt, reset, blue, ocnt, reset, magenta, bcnt, reset)

	tip, height, err := manager.GetChaintip()
	if err != nil {
		fmt.Printf("%sChaintip:%s  [None / Genesis]\n", bold, reset)
	} else {
		fmt.Printf("%sChaintip:%s  %s%s%s\n", bold, reset, magenta, tip, reset)
		fmt.Printf("%sHeight:%s    %s%d%s\n", bold, reset, yellow, height, reset)
	}

	active, elapsed, hashrate := manager.GetMiningStats()
	cores := manager.GetMiningCores()

	if active {
		fmt.Printf("%sMiner:%s     %sActive%s (%s over %s using %d cores)\n",
			bold, reset, green, reset, formatHashrate(hashrate), elapsed.Round(time.Second), cores)
	} else {
		state := "Idle"
		if cores == 0 {
			state = "Paused (0 cores)"
		}
		fmt.Printf("%sMiner:%s     %s%s%s\n", bold, reset, red, state, reset)
	}
	fmt.Printf("%s-------------------%s\n\n", bold+cyan, reset)
}

func cmdIdentity(manager *core.Manager) {
	agent := manager.Config().AgentName
	studentIDs := manager.Config().StudentIDs
	pubkey := manager.Config().PubKey

	fmt.Printf("\n%s=== Node Identity ===%s\n", bold+cyan, reset)

	fmt.Printf("%sAgent Name:%s  %s%s%s\n", bold, reset, magenta, agent, reset)

	fmt.Printf("%sPublic Key:%s  %s%s%s\n", bold, reset, yellow, pubkey, reset)

	if len(studentIDs) > 0 {
		fmt.Printf("%sStudent IDs:%s %s", bold, reset, green)
		for i, id := range studentIDs {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(id)
		}
		fmt.Println(reset)
	} else {
		fmt.Printf("%sStudent IDs:%s %s[None configured]%s\n", bold, reset, yellow, reset)
	}

	active, _, hashrate := manager.GetMiningStats()
	cores := manager.GetMiningCores()

	fmt.Printf("\n%s--- Mining Hardware ---%s\n", bold+cyan, reset)
	fmt.Printf("%sAllocated Cores:%s %d\n", bold, reset, cores)

	if active {
		fmt.Printf("%sCurrent Hashrate:%s %s%s%s\n", bold, reset, green, formatHashrate(hashrate), reset)
	} else {
		state := "Idle"
		if cores == 0 {
			state = "Paused (0 cores)"
		}
		fmt.Printf("%sCurrent Hashrate:%s %s%s%s\n", bold, reset, red, state, reset)
	}

	fmt.Printf("%s=====================%s\n\n", bold+cyan, reset)
}

func cmdCores(args []string, manager *core.Manager) {
	if len(args) != 2 {
		fmt.Printf("%sUsage: cores <number>%s\n", red, reset)
		return
	}
	cores, err := strconv.Atoi(args[1])
	if err != nil || cores < 0 {
		fmt.Printf("%sError: Cores must be a positive integer.%s\n", red, reset)
		return
	}

	maxCores := runtime.NumCPU()
	if cores > maxCores {
		fmt.Printf("%sWarning: Requested %d cores, but system only has %d. Capping at %d.%s\n", yellow, cores, maxCores, maxCores, reset)
		cores = maxCores
	}

	manager.SetMiningCores(cores)
	if cores == 0 {
		fmt.Printf("%sMiner paused. Using 0 cores.%s\n", yellow, reset)
	} else {
		fmt.Printf("%sMiner restarted with %d cores.%s\n", green, cores, reset)
	}
}

func cmdNote(args []string, manager *core.Manager, scanner *bufio.Scanner) {
	if len(args) >= 2 && (args[1] == "new" || args[1] == "set") {
		fmt.Printf("%sEnter new note (max 256 chars):%s ", yellow, reset)
		if scanner.Scan() {
			newNote := scanner.Text()
			if len(newNote) > 256 {
				fmt.Printf("%sError: Note cannot exceed 256 characters.%s\n", red, reset)
				return
			}
			manager.SetNote(types.BuString(newNote))
			fmt.Printf("%sNote updated successfully.%s\n", green, reset)
		}
	} else {
		note := manager.Config().Note
		if note == "" {
			fmt.Printf("%sNo note set.%s\n", yellow, reset)
		} else {
			fmt.Printf("%sCurrent note:%s %s\n", cyan, reset, note)
		}
	}
}

func cmdStats(args []string, manager *core.Manager, w *wallet.Wallet) {
	targetPubkey := manager.Config().PubKey
	targetName := "You"

	if len(args) >= 2 {
		targetPubkey = w.ResolveAddress(args[1])
		targetName = args[1]
	}

	fmt.Printf("\n%sScanning the blockchain history for '%s'...%s\n", yellow, targetName, reset)

	stats, err := manager.GetMiningHistory(targetPubkey)
	if err != nil {
		fmt.Printf("%sError scanning chain: %v%s\n", red, err, reset)
		return
	}

	buValue := utils.PicabuToBu(stats.TotalReward)

	fmt.Printf("\n%s=== Total Mining Stats ===%s\n", bold+magenta, reset)
	fmt.Printf("Target:       %s\n", targetPubkey)
	fmt.Printf("Blocks Mined: %d\n", stats.BlocksMined)
	fmt.Printf("Total Earned: %.6f BU\n", buValue)
	fmt.Printf("%s=============================%s\n\n", bold+magenta, reset)
}

func cmdPeers() {
	allPeers := peer.ConnManager.GetAll()
	if len(allPeers) == 0 {
		fmt.Printf("\n%sNo connected peers.%s\n\n", yellow, reset)
		return
	}

	fmt.Printf("\n%s%-20s | %-22s | %-10s | %-10s%s \n", bold+yellow, "AGENT", "ADDRESS", "ORIGIN", "PERSISTENT", reset)
	fmt.Println(cyan + strings.Repeat("-", 70) + reset)
	for _, p := range allPeers {
		vipStatus := ""
		if p.IsPersistent() {
			vipStatus = magenta + "*" + reset
		}

		agent := p.Agent()
		if agent == "" {
			agent = "Unknown"
		}

		if len(agent) > 20 {
			agent = agent[:17] + "..."
		}

		originColor := blue
		if p.Origin() == types.Inbound {
			originColor = magenta
		}

		fmt.Printf("%-20s | %s%-22s%s | %s%-10s%s | %-10s \n",
			agent, white, p.Addr(), reset, originColor, p.Origin(), reset, vipStatus)
	}
	fmt.Println()
}

func cmdConnect(args []string, manager *core.Manager) {
	if len(args) == 2 {
		addr := args[1]
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			fmt.Printf("%sError: Invalid address format. Use <ip>:<port>%s\n", red, reset)
			return
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Printf("%sError: Port must be a number.%s\n", red, reset)
			return
		}

		fmt.Printf("%sDialing %s:%d...%s\n", yellow, host, port, reset)
		err = peer.StartClient(host, port, false, manager)
		if err != nil {
			fmt.Printf("%sFailed to connect: %v%s\n", red, err, reset)
		} else {
			fmt.Printf("%sConnection initiated successfully.%s\n", green, reset)
		}
	} else {
		fmt.Printf("%sUsage: connect <ip:port>%s\n", red, reset)
	}
}

func cmdDisconnect(args []string) {
	if len(args) == 2 {
		addr := args[1]
		p, exists := peer.ConnManager.Exists(addr)
		if !exists {
			fmt.Printf("%sError: No active connection found for '%s'%s\n", red, addr, reset)
			return
		}

		p.Disconnect()
		fmt.Printf("%sDropped connection to %s%s\n", yellow, addr, reset)
	} else {
		fmt.Printf("%sUsage: disconnect <ip:port>%s\n", red, reset)
	}
}

func cmdSync() {
	fmt.Printf("%sForcing network sync...%s\n", yellow, reset)
	peer.BroadcastGetPeers()
	peer.BroadcastGetChainTip()
	fmt.Printf("%sSync requests broadcasted.%s\n", green, reset)
}

func cmdMuted() {
	records := peer.ConnManager.GetMuted()

	fmt.Printf("\n%s=== Muted Targets ===%s\n", bold+cyan, reset)

	if len(records) == 0 {
		fmt.Printf("%sNo peers are currently muted.%s\n", yellow, reset)
		fmt.Printf("%s=====================%s\n\n", bold+cyan, reset)
		return
	}

	fmt.Printf("%s%-20s | %-20s | %-15s%s\n", bold+yellow, "RULE / TARGET", "KNOWN AGENT", "KNOWN IP", reset)
	fmt.Println(cyan + strings.Repeat("-", 61) + reset)

	for _, r := range records {
		agent := r.Agent
		if len(agent) > 17 {
			agent = agent[:14] + "..."
		}

		target := r.Target
		if len(target) > 17 {
			target = target[:14] + "..."
		}

		fmt.Printf("%s%-20s%s | %-20s | %-15s\n", red, target, reset, agent, r.IP)
	}
	fmt.Printf("%s=====================%s\n\n", bold+cyan, reset)
}

func cmdMute(args []string) {
	if len(args) == 2 && args[1] == "list" {
		cmdMuted()
	} else if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		peer.ConnManager.MutePeer(val)
		fmt.Printf("%sMuted logs matching:%s %s\n", yellow, reset, val)
	} else {
		fmt.Printf("%sUsage: mute <ip or agent> OR mute list%s\n", red, reset)
	}
}

func cmdUnmute(args []string) {
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
}

func cmdBanned() {
	records := peer.ConnManager.GetBanned()

	fmt.Printf("\n%s=== Banned Targets ===%s\n", bold+cyan, reset)

	if len(records) == 0 {
		fmt.Printf("%sNo peers are currently banned.%s\n", yellow, reset)
		fmt.Printf("%s======================%s\n\n", bold+cyan, reset)
		return
	}

	fmt.Printf("%s%-20s | %-20s | %-15s%s\n", bold+yellow, "RULE / TARGET", "KNOWN AGENT", "KNOWN IP", reset)
	fmt.Println(cyan + strings.Repeat("-", 61) + reset)

	for _, r := range records {
		agent := r.Agent
		if len(agent) > 17 {
			agent = agent[:14] + "..."
		}

		target := r.Target
		if len(target) > 17 {
			target = target[:14] + "..."
		}

		fmt.Printf("%s%-20s%s | %-20s | %-15s\n", red, target, reset, agent, r.IP)
	}
	fmt.Printf("%s======================%s\n\n", bold+cyan, reset)
}

func cmdBan(args []string) {
	if len(args) == 2 && args[1] == "list" {
		cmdBanned()
	} else if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		peer.ConnManager.BanPeer(val)
		fmt.Printf("%sBanned and kicked target matching:%s %s\n", red, reset, val)
	} else {
		fmt.Printf("%sUsage: ban <ip or agent> OR ban list%s\n", red, reset)
	}
}

func cmdUnban(args []string) {
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
}

func cmdAlias(args []string, w *wallet.Wallet) {
	if len(args) < 2 {
		fmt.Println("Usage:\n  alias add <name> <pubkey>\n  alias list")
		return
	}
	switch args[1] {
	case "add":
		if len(args) != 4 {
			fmt.Println("Usage: alias add <name> <pubkey>")
			return
		}
		name, pubkey := args[2], types.HashID(args[3])
		w.AddAlias(name, pubkey)
		fmt.Printf("Alias saved: %s -> %s\n", name, pubkey)
	case "list":
		aliases := w.GetAliases()
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
}

func cmdSweep(w *wallet.Wallet) {
	fmt.Printf("Scanning the blockchain for unknown public keys...\n")
	found := w.AutoAliasUTXOs()
	fmt.Printf("Sweep complete. Found and aliased %d new unique public keys.\n", found)
	fmt.Printf("Type 'alias list' to see them!\n")
}

func cmdBalance(w *wallet.Wallet) {
	balance, utxos := w.GetBalance()
	buValue := utils.PicabuToBu(balance)
	fmt.Printf("\n=== Wallet Balance ===\n")
	fmt.Printf("Confirmed UTXOs: %d\n", len(utxos))
	fmt.Printf("Total Balance:   %.6f BU (%s Picabus)\n", buValue, balance.String())
	fmt.Printf("======================\n\n")
}

func cmdSend(args []string, w *wallet.Wallet) {
	if len(args) != 3 {
		fmt.Printf("Usage: send <target_pubkey> <amount_in_BU>\n")
		return
	}
	targetPubkey := w.ResolveAddress(args[1])
	amountBU, err := strconv.ParseFloat(args[2], 64)
	if err != nil || amountBU <= 0 {
		fmt.Printf("Error: Amount must be a positive number.\n")
		return
	}

	amountPicabu := utils.BuToPicabu(amountBU)
	feePicabu := types.NewPicabu(uint64(0.001 * 1_000_000_000_000))

	fmt.Printf("Building transaction...\n")
	tx, err := w.SendPicabus(targetPubkey, amountPicabu, feePicabu)
	if err != nil {
		fmt.Printf("Transaction Failed: %v\n", err)
		return
	}

	txid, _ := crypto.GetObjectID(tx)
	fmt.Printf("Transaction broadcasted successfully!\n")
	fmt.Printf("TXID: %s\n", txid)
}

func cmdObjects(manager *core.Manager) {
	ids, err := manager.GetAllObjectIDs()
	if err != nil {
		fmt.Printf("%sError scanning database: %v%s\n", red, err, reset)
		return
	}

	if len(ids) == 0 {
		fmt.Printf("\n%sNo objects found in the database.%s\n\n", yellow, reset)
		return
	}

	var blocks []types.HashID
	var txs []types.HashID
	var unknown []types.HashID

	fmt.Printf("%sScanning database...%s\n", yellow, reset)

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

	fmt.Printf("\n%s=== Stored Objects ===%s\n", bold+cyan, reset)
	fmt.Printf("%sTotal: %d%s\n\n", bold, len(ids), reset)

	fmt.Printf("%sBlocks (%d):%s\n", bold+magenta, len(blocks), reset)
	for _, b := range blocks {
		fmt.Printf("  | %s%s%s\n", magenta, b, reset)
	}

	fmt.Printf("\n%sTransactions (%d):%s\n", bold+green, len(txs), reset)
	for _, t := range txs {
		fmt.Printf("  | %s%s%s\n", green, t, reset)
	}

	if len(unknown) > 0 {
		fmt.Printf("\n%sUnknown/Corrupted (%d):%s\n", bold+red, len(unknown), reset)
		for _, u := range unknown {
			fmt.Printf("    %s%s%s\n", red, u, reset)
		}
	}
	fmt.Printf("%s======================%s\n\n", bold+cyan, reset)
}

func cmdGet(args []string, manager *core.Manager) {
	if len(args) == 2 {
		hashStr := args[1]
		hash := types.HashID(hashStr)

		obj, err := manager.GetObject(hash)
		if err != nil {
			fmt.Printf("%sError: Could not find object %s in database.%s\n", red, hashStr, reset)
			return
		}

		prettyJSON, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			fmt.Printf("%sError formatting object: %v%s\n", red, err, reset)
			return
		}

		fmt.Printf("\n%s--- Object %s ---%s\n", bold+cyan, hashStr, reset)
		fmt.Println(green + string(prettyJSON) + reset)
		fmt.Printf("%s----------------------------------------------------------------%s\n\n", bold+cyan, reset)

	} else {
		fmt.Printf("%sUsage: get <hash>%s\n", red, reset)
	}
}
