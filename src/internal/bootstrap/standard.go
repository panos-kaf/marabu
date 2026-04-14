//go:build standard

package bootstrap

import (
	"marabu/internal/discovery"
	"marabu/internal/peer"
	"marabu/internal/storage"
	"marabu/internal/types"

	"fmt"
	"net"
	"strconv"
)

// start server and initiate client connections to bootstrap peers
func StartNode(Store *storage.Store) {
	go peer.StartServer(18018, Store)

	for _, p := range discovery.BOOTSTRAP_PEERS {
		go func(p types.Peer) {
			host, portStr, _ := net.SplitHostPort(string(p))
			port, _ := strconv.Atoi(portStr)
			err := peer.StartClient(host, port, Store)
			if err != nil {
				fmt.Printf("Error connecting to peer %s: %v\n", p, err)
			}
		}(p)
	}
}
