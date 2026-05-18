//go:build headless

package ui

import (
	"marabu/internal/core"
	"marabu/internal/wallet"
)

func Start(manager *core.Manager, wallet *wallet.Wallet) {
	select {}
}
