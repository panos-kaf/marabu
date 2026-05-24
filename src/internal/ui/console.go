//go:build cli

package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"marabu/internal/core"
	"marabu/internal/wallet"
)

func Start(manager *core.Manager, w *wallet.Wallet) {
	fmt.Print("\033[2J\033[1;1H\033[8;r\033[8;1H")
	go startLivePanel(manager)

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
		
		cmd := strings.ToLower(args[0])

		// os.Stdout satisfies io.Writer perfectly
		shouldExit := executeCommand(cmd, args, manager, w, os.Stdout)
		
		if shouldExit {
			fmt.Printf("%sExiting CLI...%s\n", cyan, reset)
			os.Exit(0)
		}
	}
}