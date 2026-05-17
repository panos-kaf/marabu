//go:build cli

package ui

import (
	"fmt"
	"time"

	"marabu/internal/core"
	"marabu/internal/peer"
)

func startLivePanel(manager *core.Manager) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Fetch latest stats
		icnt, ocnt, bcnt := peer.ConnManager.GetCounts()
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
				"\033[K %sPeers:%s %d Total (%s%d In%s | %s%d Out%s | %s%d Ban%s)\n"+
				"\033[K %sTip:%s   %s%s%s (Height: %s%d%s)\n"+
				"\033[K %sMiner:%s %s\n"+
				"\033[K %s==========================%s\n"+
				"\033[?25h\033[u",
			bold+cyan, reset,
			bold, reset, (icnt + ocnt), green, icnt, reset, blue, ocnt, reset, red, bcnt, reset,
			bold, reset, magenta, tipStr, reset, yellow, height, reset,
			bold, reset, miningStr,
			bold+cyan, reset,
		)

		fmt.Print(panel)
	}
}

// formatHashrate converts raw H/s into readable metrics (kH/s, MH/s)
func formatHashrate(h float64) string {
	if h >= 1_000_000 {
		return fmt.Sprintf("%.2f MH/s", h/1_000_000)
	} else if h >= 1_000 {
		return fmt.Sprintf("%.2f kH/s", h/1_000)
	}
	return fmt.Sprintf("%.1f H/s", h)
}
