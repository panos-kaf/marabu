//go:build cli

package ui

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"marabu/internal/core"
	"marabu/internal/peer"
	"marabu/internal/types"
)

func listPeers() {
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

func connectToPeer(addr string, manager *core.Manager) {
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
}

func disconnectPeer(addr string) {
	p, exists := peer.ConnManager.Exists(addr)
	if !exists {
		fmt.Printf("%sError: No active connection found for '%s'%s\n", red, addr, reset)
		return
	}

	p.Disconnect()
	fmt.Printf("%sDropped connection to %s%s\n", yellow, addr, reset)
}

func listMuted() {
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

func listBanned() {
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
