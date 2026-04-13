//go:build no_bootstrap

package bootstrap

import (
	"marabu/internal/peer"
	"marabu/internal/storage"
)

func StartNode(store *storage.Store) {
	go peer.StartServer(18018, store)
}
