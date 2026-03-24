//go:build no_bootstrap

package bootstrap

import (
	"marabu/internal/objectManager"
	"marabu/internal/peer"
)

func StartNode(objectManager *objectManager.ObjectManager) {
	go peer.StartServer(18018, objectManager)
}
