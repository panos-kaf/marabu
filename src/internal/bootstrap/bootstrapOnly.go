//go:build bootstrap_only

package bootstrap

import (
	"fmt"
	"marabu/internal/core"
	"marabu/internal/discovery"
	"marabu/internal/logs"
	"marabu/internal/peer"
	"marabu/internal/types"

	"net"
	"strconv"
)

// start server and initiate client connections to bootstrap peers
func StartNode(Manager *core.Manager) {

	logs.GlobalLog("Starting marabu, only connecting to bootstrap nodes")

	for _, p := range discovery.BOOTSTRAP_PEERS {
		go func(p types.Peer) {
			host, portStr, _ := net.SplitHostPort(string(p))
			port, _ := strconv.Atoi(portStr)
			err := peer.StartClient(host, port, true, Manager)
			if err != nil {
				logs.GlobalLog(fmt.Sprintf("Error connecting to peer %s: %v\n", p, err))
			}
		}(p)
	}
}
