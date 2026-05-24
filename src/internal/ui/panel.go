//go:build cli

package ui

import (
	"fmt"
	"time"

	"marabu/internal/core"
	"marabu/internal/peer"
)

func startLivePanel(manager *core.Manager) {

	renderPanel := func() {
		// Fetch latest stats
		icnt, ocnt, pcnt, bcnt := peer.ConnManager.GetCounts()
		tip, height, err := manager.GetChaintip()
		tipStr := string(tip)
		if err != nil {
			tipStr = "[None / Genesis]"
			height = 0
		} else if len(tipStr) > 20 {
			tipStr = tipStr[:10] + "..." + tipStr[len(tipStr)-10:]
		}

		active, elapsed, hashrate := manager.GetMiningStats()
		cores := manager.GetMiningCores()

		sessionBlocks := manager.GetSessionBlocks()
		minedCount := len(sessionBlocks)

		miningStr := fmt.Sprintf("%sIdle (Waiting for network...)%s", yellow, reset)

		if cores == 0 {
			miningStr = fmt.Sprintf("%sPaused (0 Cores)%s", red, reset)
		} else if active {
			miningStr = fmt.Sprintf("%sActive [%d Cores] (%s) - %s%s",
				green, cores, elapsed.Round(time.Second), formatHashrate(hashrate), reset)
		}

		panel := fmt.Sprintf(
			"\033[s\033[?25l\033[1;1H"+
				"\033[K %s=== MARABU NODE STATUS ===%s\n"+
				"\033[K %sPeers:%s %d Total (%s%d In%s | %s%d Out%s | %s%d VIP%s | %s%d Ban%s)\n"+
				"\033[K %sTip:%s   %s%s%s (Height: %s%d%s)\n"+
				"\033[K %sMiner:%s %s\n"+
				"\033[K %sMined:%s %s%d blocks this session%s\n"+
				"\033[K %s==========================%s\n"+
				"\033[?25h\033[u",
			bold+cyan, reset,
			bold, reset, (icnt + ocnt), green, icnt, reset, blue, ocnt, reset, magenta, pcnt, reset, red, bcnt, reset,
			bold, reset, magenta, tipStr, reset, yellow, height, reset,
			bold, reset, miningStr,
			bold, reset, green, minedCount, reset,
			bold+cyan, reset,
		)

		fmt.Print(panel)
	}

	renderPanel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		renderPanel()
	}
}
