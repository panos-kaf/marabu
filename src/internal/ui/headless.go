//go:build headless

package ui

import "marabu/internal/core"

func Start(manager *core.Manager) {
	select {}
}
