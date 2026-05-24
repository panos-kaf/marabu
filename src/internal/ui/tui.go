//go:build tui

package ui

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"marabu/internal/core"
	"marabu/internal/peer"
	"marabu/internal/wallet"
)

var uiPaused atomic.Bool

func Start(manager *core.Manager, w *wallet.Wallet) {
	app := tview.NewApplication()

	statsPanel := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	statsPanel.SetBorder(false)

	outputView := tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetWordWrap(true)
	outputView.SetBorder(false)

	inputField := tview.NewInputField().
		SetLabel("[green]>[white] ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.ColorWhite)

	separator := tview.NewTextView().
		SetText(strings.Repeat("─", 500)).
		SetWrap(false)
	separator.SetTextColor(tcell.ColorDarkGray)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statsPanel, 6, 1, false).
		AddItem(separator, 1, 0, false).
		AddItem(outputView, 0, 1, false).
		AddItem(inputField, 1, 1, true)

	go startLivePanel(manager, app, statsPanel)

	fmt.Fprintf(outputView, "[cyan][bold]Marabu TUI started. Type 'help' for commands.[white]\n")

	var cmdHistory []string
	var historyIdx int

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			input := inputField.GetText()
			args := strings.Fields(input)
			if len(args) == 0 {
				return
			}

			if len(cmdHistory) == 0 || cmdHistory[len(cmdHistory)-1] != input {
				cmdHistory = append(cmdHistory, input)
			}
			historyIdx = len(cmdHistory)

			cmd := strings.ToLower(args[0])
			fmt.Fprintf(outputView, "\n[yellow]> %s[white]\n", input)

			// THE MAGIC TRICK: Wrap outputView with an ANSI parser!
			ansiWriter := tview.ANSIWriter(outputView)
			shouldExit := executeCommand(cmd, args, manager, w, ansiWriter)

			if shouldExit {
				app.Stop()
				os.Exit(0)
			}

			inputField.SetText("")
			outputView.ScrollToEnd()
		}
	})

	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if len(cmdHistory) == 0 {
			return event
		}
		switch event.Key() {
		case tcell.KeyUp:
			if historyIdx > 0 {
				historyIdx--
				inputField.SetText(cmdHistory[historyIdx])
			}
			return nil
		case tcell.KeyDown:
			if historyIdx < len(cmdHistory)-1 {
				historyIdx++
				inputField.SetText(cmdHistory[historyIdx])
			} else if historyIdx == len(cmdHistory)-1 {
				historyIdx++
				inputField.SetText("")
			}
			return nil
		case tcell.KeyPgUp, tcell.KeyPgDn:
			// Pass the keyboard event directly to the output view's native handler
			outputView.InputHandler()(event, nil)
			return nil
		}
		return event
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlF {
			newState := !uiPaused.Load()
			uiPaused.Store(newState)
			if newState {
				fmt.Fprintf(outputView, "\n[bg:red][white][bold]--- UI FROZEN FOR COPYING (Press Ctrl+F to unfreeze) ---[-:-:-]\n")
			} else {
				fmt.Fprintf(outputView, "\n[bg:green][white][bold]--- UI UNFROZEN ---[-:-:-]\n")
			}
			return nil
		}
		return event
	})

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		fmt.Printf("Error starting TUI: %v\n", err)
		os.Exit(1)
	}
}

func startLivePanel(manager *core.Manager, app *tview.Application, panel *tview.TextView) {
	renderPanel := func() {
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
		minedCount := len(manager.GetSessionBlocks())
		miningStr := "[yellow]Idle (Waiting for network...)"
		if cores == 0 {
			miningStr = "[red]Paused (0 Cores)"
		} else if active {
			miningStr = fmt.Sprintf("[green]Active [%d Cores] (%s) - %s",
				cores, elapsed.Round(time.Second), formatHashrate(hashrate))
		}

		app.QueueUpdateDraw(func() {
			if uiPaused.Load() {
				return
			}
			panel.SetText(fmt.Sprintf(
				"[cyan][bold]Marabu Node Status[white]\n"+
					"[cyan][bold]Peers:[white] %d Total ([green]%d In[white] | [blue]%d Out[white] | [magenta]%d VIP[white] | [red]%d Ban[white])\n"+"[cyan][bold]Tip:[white]   [magenta]%s[white] (Height: [yellow]%d[white])\n"+
					"[cyan][bold]Miner:[white] %s[white]\n"+
					"[cyan][bold]Mined:[white] [green]%d blocks this session[white]",
				(icnt + ocnt), icnt, ocnt, pcnt, bcnt,
				tipStr, height,
				miningStr,
				minedCount,
			))
		})
	}
	renderPanel()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		renderPanel()
	}
}
