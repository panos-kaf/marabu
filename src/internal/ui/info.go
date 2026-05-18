//go:build cli

package ui

import (
	"fmt"
	"marabu/internal/core"
	"marabu/internal/peer"
	"time"
)

func printHelp() {
	fmt.Printf("\n%sAvailable commands:%s\n", bold+yellow, reset)
	fmt.Printf("  %sinfo, i, status%s         - Show node diagnostics and chaintip\n", cyan, reset)
	fmt.Printf("  %sid, identity, miner%s     - Show node identity, agent, and student IDs\n", cyan, reset)
	fmt.Printf("  %speers, p%s                - List detailed connected peers\n", cyan, reset)
	fmt.Printf("  %sconnect <ip:port>%s       - Manually connect to a node\n", cyan, reset)
	fmt.Printf("  %sbalance%s                 - Show wallet balance and UTXOs\n", cyan, reset)
	fmt.Printf("  %ssend <pubkey> <amount> [fee]%s - Send funds to a public key (amount in Picabus, fee optional)\n", cyan, reset)
	fmt.Printf("  %sdisconnect <ip:port>%s    - Disconnect from a node (alias: drop)\n", cyan, reset)
	fmt.Printf("  %smute <ip|agent> <val>%s   - Mute logs from a spammy IP or Agent\n", cyan, reset)
	fmt.Printf("  %sunmute <ip|agent> <val>%s - Unmute logs\n", cyan, reset)
	fmt.Printf("  %sban <target>%s            - Ban and kick an IP or Agent\n", cyan, reset)
	fmt.Printf("  %sunban <target>%s          - Unban an IP or Agent\n", cyan, reset)
	fmt.Printf("  %sbanned%s                  - List all currently banned targets\n", cyan, reset)
	fmt.Printf("  %sobjects, o%s              - List all objects in the database\n", cyan, reset)
	fmt.Printf("  %sget <hash>%s              - Fetch and display a specific object\n", cyan, reset)
	fmt.Printf("  %snote%s                    - Show current miner note\n", cyan, reset)
	fmt.Printf("  %snote set%s				  - Set a custom miner note\n", cyan, reset)
	fmt.Printf("  %scores, c <num>%s          - Adjust the number of CPU cores used for mining\n", cyan, reset)
	fmt.Printf("  %ssync%s                    - Force broadcast GetPeers and GetChainTip\n", cyan, reset)
	fmt.Printf("  %sexit, quit, q%s           - Exit the CLI\n\n", cyan, reset)
}

func printInfo(manager *core.Manager) {
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

func printIdentity(manager *core.Manager) {
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
