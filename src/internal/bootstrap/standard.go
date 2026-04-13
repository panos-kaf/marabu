//go:build standard

package bootstrap

import (
	"fmt"
	"marabu/internal/messages"
	"marabu/internal/peer"
	"marabu/internal/storage"
	"net"
	"strconv"
)

// start server and initiate client connections to bootstrap peers
func StartNode(Store *storage.Store) {
	go peer.StartServer(18018, Store)

	for _, p := range peer.BOOTSTRAP_PEERS {
		go func(p messages.T_Peer) {
			host, portStr, _ := net.SplitHostPort(string(p))
			port, _ := strconv.Atoi(portStr)
			err := peer.StartClient(host, port, Store)
			if err != nil {
				fmt.Printf("Error connecting to peer %s: %v\n", p, err)
			}
		}(p)
	}
}
