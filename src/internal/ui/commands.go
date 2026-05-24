package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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

// Standard ANSI Color Codes
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

// executeCommand returns true if the application should exit
func executeCommand(cmd string, args []string, manager *core.Manager, w *wallet.Wallet, out io.Writer) bool {
	switch cmd {
	// System
	case "help", "?", "h":
		cmdHelp(out)
	case "exit", "quit", "q":
		return true // Signal to the parent wrapper to cleanly shut down

	// Node & Hardware
	case "info", "i", "status":
		cmdInfo(manager, out)
	case "id", "identity", "miner":
		cmdIdentity(manager, out)
	case "cores", "c", "miners", "m":
		cmdCores(args, manager, out)
	case "note":
		cmdNote(args, manager, out)
	case "stats":
		cmdStats(args, manager, w, out)

	// Networking
	case "peers", "p":
		cmdPeers(out)
	case "connect":
		cmdConnect(args, manager, out)
	case "disconnect", "drop":
		cmdDisconnect(args, out)
	case "sync":
		cmdSync(out)

	// Moderation
	case "muted":
		cmdMuted(out)
	case "mute":
		cmdMute(args, out)
	case "unmute":
		cmdUnmute(args, out)
	case "banned":
		cmdBanned(out)
	case "ban":
		cmdBan(args, out)
	case "unban":
		cmdUnban(args, out)

	// Wallet & Transactions
	case "alias":
		cmdAlias(args, w, out)
	case "sweep":
		cmdSweep(w, out)
	case "balance":
		cmdBalance(w, out)
	case "send":
		cmdSend(args, w, out)

	// Database
	case "objects", "o":
		cmdObjects(manager, out)
	case "get":
		cmdGet(args, manager, out)

	default:
		fmt.Fprintf(out, "%sUnknown command: %s%s\n", red, cmd, reset)
	}

	return false
}

// -------------------------------------------------------------------------
// DOMAIN HANDLERS (Using ANSI and io.Writer)
// -------------------------------------------------------------------------

func cmdHelp(out io.Writer) {
	fmt.Fprintf(out, "\n%s=== Marabu CLI Commands ===%s\n", bold+yellow, reset)
	fmt.Fprintf(out, "\n%s[ Node & Hardware ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %sinfo, i, status%s         - Show node diagnostics and chaintip\n", cyan, reset)
	fmt.Fprintf(out, "  %sid, identity, miner%s     - Show node identity, agent, and student IDs\n", cyan, reset)
	fmt.Fprintf(out, "  %scores, c <num>%s          - Adjust the number of CPU cores used for mining\n", cyan, reset)
	fmt.Fprintf(out, "  %snote%s                    - Show current miner note\n", cyan, reset)
	fmt.Fprintf(out, "  %snote set <msg>%s          - Set a custom miner note\n", cyan, reset)
	fmt.Fprintf(out, "  %sstats [target]%s          - Show total historical blocks mined by a pubkey/alias\n", cyan, reset)

	fmt.Fprintf(out, "\n%s[ Wallet & Transactions ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %sbalance%s                 - Show wallet balance and UTXOs\n", cyan, reset)
	fmt.Fprintf(out, "  %ssend <target> <BU>%s      - Send funds to a pubkey or alias\n", cyan, reset)
	fmt.Fprintf(out, "  %salias add <name> <key>%s  - Save a public key to your address book\n", cyan, reset)
	fmt.Fprintf(out, "  %salias list%s              - View your address book\n", cyan, reset)
	fmt.Fprintf(out, "  %ssweep%s                   - Auto-alias all unknown public keys in the UTXO set\n", cyan, reset)

	fmt.Fprintf(out, "\n%s[ Networking & Peers ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %speers, p%s                - List detailed connected peers\n", cyan, reset)
	fmt.Fprintf(out, "  %sconnect <ip:port>%s       - Manually connect to a node\n", cyan, reset)
	fmt.Fprintf(out, "  %sdisconnect <ip:port>%s    - Disconnect from a node (alias: drop)\n", cyan, reset)
	fmt.Fprintf(out, "  %ssync%s                    - Force broadcast GetPeers and GetChainTip\n", cyan, reset)

	fmt.Fprintf(out, "\n%s[ Moderation ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %smuted%s                   - List all muted targets\n", cyan, reset)
	fmt.Fprintf(out, "  %smute <target>%s           - Mute logs from a spammy IP or Agent\n", cyan, reset)
	fmt.Fprintf(out, "  %sunmute <target> | *%s     - Unmute a target (use * for all)\n", cyan, reset)
	fmt.Fprintf(out, "  %sbanned%s                  - List all currently banned targets\n", cyan, reset)
	fmt.Fprintf(out, "  %sban <target>%s            - Ban and kick an IP or Agent\n", cyan, reset)
	fmt.Fprintf(out, "  %sunban <target> | *%s      - Unban a target (use * for all)\n", cyan, reset)

	fmt.Fprintf(out, "\n%s[ Database ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %sobjects, o%s              - List all objects in the database\n", cyan, reset)
	fmt.Fprintf(out, "  %sget <hash>%s              - Fetch and display a specific object\n", cyan, reset)

	fmt.Fprintf(out, "\n%s[ System ]%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "  %shelp, ?, h%s              - Show this menu\n", cyan, reset)
	fmt.Fprintf(out, "  %sexit, quit, q%s           - Exit the CLI\n\n", cyan, reset)
}

func cmdInfo(manager *core.Manager, out io.Writer) {
	icnt, ocnt, pcnt, _ := peer.ConnManager.GetCounts()

	fmt.Fprintf(out, "\n%s--- Node Status ---%s\n", bold+cyan, reset)
	fmt.Fprintf(out, "%sPeers:%s     %d Total (%s%d Inbound%s | %s%d Outbound%s | %s%d VIP%s)\n",
		bold, reset, (icnt + ocnt), green, icnt, reset, blue, ocnt, reset, magenta, pcnt, reset)

	tip, height, err := manager.GetChaintip()
	if err != nil {
		fmt.Fprintf(out, "%sChaintip:%s  [None / Genesis]\n", bold, reset)
	} else {
		fmt.Fprintf(out, "%sChaintip:%s  %s%s%s\n", bold, reset, magenta, tip, reset)
		fmt.Fprintf(out, "%sHeight:%s    %s%d%s\n", bold, reset, yellow, height, reset)
	}

	active, elapsed, hashrate := manager.GetMiningStats()
	cores := manager.GetMiningCores()

	if active {
		fmt.Fprintf(out, "%sMiner:%s     %sActive%s (%s over %s using %d cores)\n",
			bold, reset, green, reset, formatHashrate(hashrate), elapsed.Round(time.Second), cores)
	} else {
		state := "Idle"
		if cores == 0 {
			state = "Paused (0 cores)"
		}
		fmt.Fprintf(out, "%sMiner:%s     %s%s%s\n", bold, reset, red, state, reset)
	}
	fmt.Fprintf(out, "%s-------------------%s\n\n", bold+cyan, reset)
}

func cmdIdentity(manager *core.Manager, out io.Writer) {
	agent := manager.Config().AgentName
	studentIDs := manager.Config().StudentIDs
	pubkey := manager.Config().PubKey

	fmt.Fprintf(out, "\n%s=== Node Identity ===%s\n", bold+cyan, reset)
	fmt.Fprintf(out, "%sAgent Name:%s  %s%s%s\n", bold, reset, magenta, agent, reset)
	fmt.Fprintf(out, "%sPublic Key:%s  %s%s%s\n", bold, reset, yellow, pubkey, reset)

	if len(studentIDs) > 0 {
		fmt.Fprintf(out, "%sStudent IDs:%s %s", bold, reset, green)
		for i, id := range studentIDs {
			if i > 0 {
				fmt.Fprintf(out, ", ")
			}
			fmt.Fprintf(out, string(id))
		}
		fmt.Fprintf(out, "%s\n", reset)
	} else {
		fmt.Fprintf(out, "%sStudent IDs:%s %s[None configured]%s\n", bold, reset, yellow, reset)
	}

	active, _, hashrate := manager.GetMiningStats()
	cores := manager.GetMiningCores()

	fmt.Fprintf(out, "\n%s--- Mining Hardware ---%s\n", bold+cyan, reset)
	fmt.Fprintf(out, "%sAllocated Cores:%s %d\n", bold, reset, cores)

	if active {
		fmt.Fprintf(out, "%sCurrent Hashrate:%s %s%s%s\n", bold, reset, green, formatHashrate(hashrate), reset)
	} else {
		state := "Idle"
		if cores == 0 {
			state = "Paused (0 cores)"
		}
		fmt.Fprintf(out, "%sCurrent Hashrate:%s %s%s%s\n", bold, reset, red, state, reset)
	}
	fmt.Fprintf(out, "%s=====================%s\n\n", bold+cyan, reset)
}

func cmdCores(args []string, manager *core.Manager, out io.Writer) {
	if len(args) == 1 {
		cores := manager.GetMiningCores()
		fmt.Fprintf(out, "%sCurrently using %d cores for mining.\nTo change, enter: cores <number>%s\n", yellow, cores, reset)
		return
	}
	if len(args) > 2 {
		fmt.Fprintf(out, "%sUsage: cores <number>%s\n", red, reset)
		return
	}
	cores, err := strconv.Atoi(args[1])
	if err != nil || cores < 0 {
		fmt.Fprintf(out, "%sError: Cores must be a positive integer.%s\n", red, reset)
		return
	}

	maxCores := runtime.NumCPU()
	if cores > maxCores {
		fmt.Fprintf(out, "%sWarning: Requested %d cores, but system only has %d. Capping at %d.%s\n", yellow, cores, maxCores, maxCores, reset)
		cores = maxCores
	}

	manager.SetMiningCores(cores)
	if cores == 0 {
		fmt.Fprintf(out, "%sMiner paused. Using 0 cores.%s\n", yellow, reset)
	} else {
		fmt.Fprintf(out, "%sMiner restarted with %d cores.%s\n", green, cores, reset)
	}
}

func cmdNote(args []string, manager *core.Manager, out io.Writer) {
	if len(args) >= 2 && (args[1] == "new" || args[1] == "set") {
		if len(args) < 3 {
			fmt.Fprintf(out, "%sUsage: note set <your message>%s\n", red, reset)
			return
		}
		newNote := strings.Join(args[2:], " ")
		if len(newNote) > 256 {
			fmt.Fprintf(out, "%sError: Note cannot exceed 256 characters.%s\n", red, reset)
			return
		}
		manager.SetNote(types.BuString(newNote))
		fmt.Fprintf(out, "%sNote updated successfully.%s\n", green, reset)
	} else {
		note := manager.Config().Note
		if note == "" {
			fmt.Fprintf(out, "%sNo note set.%s\n", yellow, reset)
		} else {
			fmt.Fprintf(out, "%sCurrent note:%s %s\n", cyan, reset, note)
		}
	}
}

func cmdStats(args []string, manager *core.Manager, w *wallet.Wallet, out io.Writer) {
	targetPubkey := manager.Config().PubKey
	targetName := "You"

	if len(args) >= 2 {
		targetPubkey = w.ResolveAddress(args[1])
		targetName = args[1]
	}

	fmt.Fprintf(out, "\n%sScanning the blockchain history for '%s'...%s\n", yellow, targetName, reset)
	stats, err := manager.GetMiningHistory(targetPubkey)
	if err != nil {
		fmt.Fprintf(out, "%sError scanning chain: %v%s\n", red, err, reset)
		return
	}

	buValue := utils.PicabuToBu(stats.TotalReward)
	fmt.Fprintf(out, "\n%s=== Total Mining Stats ===%s\n", bold+magenta, reset)
	fmt.Fprintf(out, "Target:       %s\n", targetPubkey)
	fmt.Fprintf(out, "Blocks Mined: %d\n", stats.BlocksMined)
	fmt.Fprintf(out, "Total Earned: %.6f BU\n", buValue)
	fmt.Fprintf(out, "%s=============================%s\n\n", bold+magenta, reset)
}

func cmdPeers(out io.Writer) {
	allPeers := peer.ConnManager.GetAll()
	if len(allPeers) == 0 {
		fmt.Fprintf(out, "\n%sNo connected peers.%s\n\n", yellow, reset)
		return
	}

	fmt.Fprintf(out, "\n%s%-20s | %-22s | %-10s | %-10s%s \n", bold+yellow, "AGENT", "ADDRESS", "ORIGIN", "PERSISTENT", reset)
	fmt.Fprintf(out, "%s%s%s\n", cyan, strings.Repeat("-", 70), reset)
	for _, p := range allPeers {
		vipStatus := ""
		if p.IsPersistent() {
			vipStatus = magenta + "*" + reset
		}
		agent := string(p.Agent())
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
		fmt.Fprintf(out, "%-20s | %s%-22s%s | %s%-10s%s | %-10s \n",
			agent, white, p.Addr(), reset, originColor, p.Origin(), reset, vipStatus)
	}
	fmt.Fprintf(out, "\n")
}

func cmdConnect(args []string, manager *core.Manager, out io.Writer) {
	if len(args) == 2 {
		addr := args[1]
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			fmt.Fprintf(out, "%sError: Invalid address format. Use <ip>:<port>%s\n", red, reset)
			return
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Fprintf(out, "%sError: Port must be a number.%s\n", red, reset)
			return
		}
		fmt.Fprintf(out, "%sDialing %s:%d...%s\n", yellow, host, port, reset)
		err = peer.StartClient(host, port, false, manager)
		if err != nil {
			fmt.Fprintf(out, "%sFailed to connect: %v%s\n", red, err, reset)
		} else {
			fmt.Fprintf(out, "%sConnection initiated successfully.%s\n", green, reset)
		}
	} else {
		fmt.Fprintf(out, "%sUsage: connect <ip:port>%s\n", red, reset)
	}
}

func cmdDisconnect(args []string, out io.Writer) {
	if len(args) == 2 {
		addr := args[1]
		p, exists := peer.ConnManager.Exists(addr)
		if !exists {
			fmt.Fprintf(out, "%sError: No active connection found for '%s'%s\n", red, addr, reset)
			return
		}
		p.Disconnect()
		fmt.Fprintf(out, "%sDropped connection to %s%s\n", yellow, addr, reset)
	} else {
		fmt.Fprintf(out, "%sUsage: disconnect <ip:port>%s\n", red, reset)
	}
}

func cmdSync(out io.Writer) {
	fmt.Fprintf(out, "%sForcing network sync...%s\n", yellow, reset)
	peer.BroadcastGetPeers()
	peer.BroadcastGetChainTip()
	fmt.Fprintf(out, "%sSync requests broadcasted.%s\n", green, reset)
}

func cmdMuted(out io.Writer) {
	records := peer.ConnManager.GetMuted()
	fmt.Fprintf(out, "\n%s=== Muted Targets ===%s\n", bold+cyan, reset)
	if len(records) == 0 {
		fmt.Fprintf(out, "%sNo peers are currently muted.%s\n", yellow, reset)
		fmt.Fprintf(out, "%s=====================%s\n\n", bold+cyan, reset)
		return
	}
	fmt.Fprintf(out, "%s%-20s | %-20s | %-15s%s\n", bold+yellow, "RULE / TARGET", "KNOWN AGENT", "KNOWN IP", reset)
	fmt.Fprintf(out, "%s%s%s\n", cyan, strings.Repeat("-", 61), reset)
	for _, r := range records {
		agent, target := string(r.Agent), string(r.Target)
		if len(agent) > 17 {
			agent = agent[:14] + "..."
		}
		if len(target) > 17 {
			target = target[:14] + "..."
		}
		fmt.Fprintf(out, "%s%-20s%s | %-20s | %-15s\n", red, target, reset, agent, r.IP)
	}
	fmt.Fprintf(out, "%s=====================%s\n\n", bold+cyan, reset)
}

func cmdMute(args []string, out io.Writer) {
	if len(args) == 2 && args[1] == "list" {
		cmdMuted(out)
	} else if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		peer.ConnManager.MutePeer(val)
		fmt.Fprintf(out, "%sMuted logs matching:%s %s\n", yellow, reset, val)
	} else {
		fmt.Fprintf(out, "%sUsage: mute <ip or agent> OR mute list%s\n", red, reset)
	}
}

func cmdUnmute(args []string, out io.Writer) {
	if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		if val == "*" {
			cleared := peer.ConnManager.UnmuteAll()
			fmt.Fprintf(out, "%sCleared %d mute rules. All peers unmuted.%s\n", green, cleared, reset)
		} else {
			peer.ConnManager.UnmutePeer(val)
			fmt.Fprintf(out, "%sUnmuted logs matching:%s %s\n", green, reset, val)
		}
	} else {
		fmt.Fprintf(out, "%sUsage: unmute <ip or agent> OR unmute *%s\n", red, reset)
	}
}

func cmdBanned(out io.Writer) {
	records := peer.ConnManager.GetBanned()
	fmt.Fprintf(out, "\n%s=== Banned Targets ===%s\n", bold+cyan, reset)
	if len(records) == 0 {
		fmt.Fprintf(out, "%sNo peers are currently banned.%s\n", yellow, reset)
		fmt.Fprintf(out, "%s======================%s\n\n", bold+cyan, reset)
		return
	}
	fmt.Fprintf(out, "%s%-20s | %-20s | %-15s%s\n", bold+yellow, "RULE / TARGET", "KNOWN AGENT", "KNOWN IP", reset)
	fmt.Fprintf(out, "%s%s%s\n", cyan, strings.Repeat("-", 61), reset)
	for _, r := range records {
		agent, target := string(r.Agent), string(r.Target)
		if len(agent) > 17 {
			agent = agent[:14] + "..."
		}
		if len(target) > 17 {
			target = target[:14] + "..."
		}
		fmt.Fprintf(out, "%s%-20s%s | %-20s | %-15s\n", red, target, reset, agent, r.IP)
	}
	fmt.Fprintf(out, "%s======================%s\n\n", bold+cyan, reset)
}

func cmdBan(args []string, out io.Writer) {
	if len(args) == 2 && args[1] == "list" {
		cmdBanned(out)
	} else if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		peer.ConnManager.BanPeer(val)
		fmt.Fprintf(out, "%sBanned and kicked target matching:%s %s\n", red, reset, val)
	} else {
		fmt.Fprintf(out, "%sUsage: ban <ip or agent> OR ban list%s\n", red, reset)
	}
}

func cmdUnban(args []string, out io.Writer) {
	if len(args) >= 2 {
		val := strings.Join(args[1:], " ")
		if val == "*" {
			cleared := peer.ConnManager.UnbanAll()
			fmt.Fprintf(out, "%sCleared %d ban rules. All peers unbanned.%s\n", green, cleared, reset)
		} else {
			peer.ConnManager.UnbanPeer(val)
			fmt.Fprintf(out, "%sUnbanned target matching:%s %s\n", green, reset, val)
		}
	} else {
		fmt.Fprintf(out, "%sUsage: unban <ip or agent> OR unban *%s\n", red, reset)
	}
}

func cmdAlias(args []string, w *wallet.Wallet, out io.Writer) {
	if len(args) < 2 {
		fmt.Fprintf(out, "Usage:\n  alias add <name> <pubkey>\n  alias list\n")
		return
	}
	switch args[1] {
	case "add":
		if len(args) != 4 {
			fmt.Fprintf(out, "Usage: alias add <name> <pubkey>\n")
			return
		}
		name, pubkey := args[2], types.HashID(args[3])
		w.AddAlias(name, pubkey)
		fmt.Fprintf(out, "Alias saved: %s -> %s\n", name, pubkey)
	case "list":
		aliases := w.GetAliases()
		if len(aliases) == 0 {
			fmt.Fprintf(out, "Address book is empty.\n")
			return
		}
		fmt.Fprintf(out, "\n=== Address Book ===\n")
		for name, pubkey := range aliases {
			fmt.Fprintf(out, "%-10s : %s\n", name, pubkey)
		}
		fmt.Fprintf(out, "====================\n")
	}
}

func cmdSweep(w *wallet.Wallet, out io.Writer) {
	fmt.Fprintf(out, "Scanning the blockchain for unknown public keys...\n")
	found := w.AutoAliasUTXOs()
	fmt.Fprintf(out, "Sweep complete. Found and aliased %d new unique public keys.\n", found)
	fmt.Fprintf(out, "Type 'alias list' to see them!\n")
}

func cmdBalance(w *wallet.Wallet, out io.Writer) {
	balance, utxos := w.GetBalance()
	buValue := utils.PicabuToBu(balance)
	fmt.Fprintf(out, "\n=== Wallet Balance ===\n")
	fmt.Fprintf(out, "Confirmed UTXOs: %d\n", len(utxos))
	fmt.Fprintf(out, "Total Balance:   %.6f BU (%s Picabus)\n", buValue, balance.String())
	fmt.Fprintf(out, "======================\n\n")
}

func cmdSend(args []string, w *wallet.Wallet, out io.Writer) {
	if len(args) != 3 {
		fmt.Fprintf(out, "Usage: send <target_pubkey> <amount_in_BU>\n")
		return
	}
	targetPubkey := w.ResolveAddress(args[1])
	amountBU, err := strconv.ParseFloat(args[2], 64)
	if err != nil || amountBU <= 0 {
		fmt.Fprintf(out, "Error: Amount must be a positive number.\n")
		return
	}
	amountPicabu := utils.BuToPicabu(amountBU)
	feePicabu := types.NewPicabu(uint64(0.001 * 1_000_000_000_000))

	fmt.Fprintf(out, "Building transaction...\n")
	tx, err := w.SendPicabus(targetPubkey, amountPicabu, feePicabu)
	if err != nil {
		fmt.Fprintf(out, "Transaction Failed: %v\n", err)
		return
	}
	txid, _ := crypto.GetObjectID(tx)
	fmt.Fprintf(out, "Transaction broadcasted successfully!\n")
	fmt.Fprintf(out, "TXID: %s\n", txid)
}

func cmdObjects(manager *core.Manager, out io.Writer) {
	ids, err := manager.GetAllObjectIDs()
	if err != nil {
		fmt.Fprintf(out, "%sError scanning database: %v%s\n", red, err, reset)
		return
	}
	if len(ids) == 0 {
		fmt.Fprintf(out, "\n%sNo objects found in the database.%s\n\n", yellow, reset)
		return
	}
	var blocks, txs, unknown []types.HashID

	fmt.Fprintf(out, "%sScanning database...%s\n", yellow, reset)
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

	fmt.Fprintf(out, "\n%s=== Stored Objects ===%s\n", bold+cyan, reset)
	fmt.Fprintf(out, "%sTotal: %d%s\n\n", bold, len(ids), reset)

	fmt.Fprintf(out, "%sBlocks (%d):%s\n", bold+magenta, len(blocks), reset)
	for _, b := range blocks {
		fmt.Fprintf(out, "  | %s%s%s\n", magenta, b, reset)
	}

	fmt.Fprintf(out, "\n%sTransactions (%d):%s\n", bold+green, len(txs), reset)
	for _, t := range txs {
		fmt.Fprintf(out, "  | %s%s%s\n", green, t, reset)
	}

	if len(unknown) > 0 {
		fmt.Fprintf(out, "\n%sUnknown/Corrupted (%d):%s\n", bold+red, len(unknown), reset)
		for _, u := range unknown {
			fmt.Fprintf(out, "    %s%s%s\n", red, u, reset)
		}
	}
	fmt.Fprintf(out, "%s======================%s\n\n", bold+cyan, reset)
}

func cmdGet(args []string, manager *core.Manager, out io.Writer) {
	if len(args) == 2 {
		hashStr := args[1]
		hash := types.HashID(hashStr)

		obj, err := manager.GetObject(hash)
		if err != nil {
			fmt.Fprintf(out, "%sError: Could not find object %s in database.%s\n", red, hashStr, reset)
			return
		}
		prettyJSON, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			fmt.Fprintf(out, "%sError formatting object: %v%s\n", red, err, reset)
			return
		}

		fmt.Fprintf(out, "\n%s--- Object %s ---%s\n", bold+cyan, hashStr, reset)
		fmt.Fprintf(out, "%s%s%s\n", green, string(prettyJSON), reset)
		fmt.Fprintf(out, "%s----------------------------------------------------------------%s\n\n", bold+cyan, reset)

	} else {
		fmt.Fprintf(out, "%sUsage: get <hash>%s\n", red, reset)
	}
}

// formatHashrate converts raw H/s into readable metrics
func formatHashrate(h float64) string {
	if h >= 1_000_000 {
		return fmt.Sprintf("%.2f MH/s", h/1_000_000)
	} else if h >= 1_000 {
		return fmt.Sprintf("%.2f kH/s", h/1_000)
	}
	return fmt.Sprintf("%.1f H/s", h)
}
