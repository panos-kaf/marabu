//go:build standard

package bootstrap

import (
	"fmt"
	"marabu/internal/messages"
	"marabu/internal/storage"
	"marabu/internal/peer"
	"net"
	"strconv"
)

// start server and initiate client connections to bootstrap peers
func StartNode(objectManager *storage.ObjectManager) {
	go peer.StartServer(18018, objectManager)

	for _, p := range peer.BOOTSTRAP_PEERS {
		go func(p messages.T_Peer) {
			host, portStr, _ := net.SplitHostPort(string(p))
			port, _ := strconv.Atoi(portStr)
			err := peer.StartClient(host, port, objectManager)
			if err != nil {
				fmt.Printf("Error connecting to peer %s: %v\n", p, err)
			}
		}(p)
	}
}
