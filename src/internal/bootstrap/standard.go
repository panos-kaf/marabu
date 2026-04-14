//go:build standard

package bootstrap

import (
	"marabu/internal/core"
	"marabu/internal/discovery"
	"marabu/internal/peer"
	"marabu/internal/types"

	"fmt"
	"net"
	"strconv"
)

// start server and initiate client connections to bootstrap peers
func StartNode(Manager *core.Manager) {
	go peer.StartServer(18018, Manager)

	for _, p := range discovery.BOOTSTRAP_PEERS {
		go func(p types.Peer) {
			host, portStr, _ := net.SplitHostPort(string(p))
			port, _ := strconv.Atoi(portStr)
			err := peer.StartClient(host, port, Manager)
			if err != nil {
				fmt.Printf("Error connecting to peer %s: %v\n", p, err)
			}
		}(p)
	}
}
