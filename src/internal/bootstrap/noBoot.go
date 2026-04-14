//go:build no_bootstrap

package bootstrap

import (
	"marabu/internal/core"
	"marabu/internal/peer"
)

func StartNode(Manager *core.Manager) {
	go peer.StartServer(18018, Manager)
}
