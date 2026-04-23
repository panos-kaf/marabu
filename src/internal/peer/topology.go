package peer

import (
	"errors"
	"fmt"
	"marabu/internal/core"
	"marabu/internal/discovery"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"math/rand"
	"time"
)

const MaxOutbound = 10

var pendingDials atomic.Int32

func StartTopologyManager(Manager *core.Manager) {

	replenishOutbound(Manager)

	// Drop one peer every 5 mins
	churnTicker := time.NewTicker(5 * time.Minute)

	// Look for missing peers every 10 seconds
	replenishTicker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-churnTicker.C:
			churnRandomPeer()
		case <-replenishTicker.C:
			replenishOutbound(Manager)
		}
	}
}
func churnRandomPeer() {

	disposablePeers := ConnManager.GetDisposable()

	if len(disposablePeers) == 0 {
		return
	}

	victim := disposablePeers[rand.Intn(len(disposablePeers))]

	globalLog("Topology Manager: Churning peer " + victim.addr)
	victim.conn.Close()

}

func replenishOutbound(Manager *core.Manager) {
	_, outbound, _ := ConnManager.GetCounts()

	if outbound >= MaxOutbound {
		return
	}

	candidates := discovery.SelectRecentPeers(50, ConnManager.GetActiveIPs())

	// globalLog("Topology Manager: Replenishing outbound connections...")
	globalLog(fmt.Sprintf("Topology Manager: Attempting to dial %d peers...", len(candidates)))

	// globalLog(fmt.Sprintf("Topology Manager: Known Peers: %d", discovery.GetKnownPeersCount()))

	var wg sync.WaitGroup
	var found atomic.Int32

	for _, addrStr := range candidates {

		wg.Add(1)

		go func(addr string) {

			defer wg.Done()

			host, portStr, _ := net.SplitHostPort(addr)
			port, _ := strconv.Atoi(portStr)

			// StartClient handles everything. If the node fills up while this
			// dial is in transit, NewPeer will safely reject and close it.
			err := StartClient(host, port, false, Manager)

			if err != nil {

				if errors.Is(err, ErrOutboundFull) || errors.Is(err, ErrOutboundDuplicate) {
					return
				}
				discovery.RemovePeer(addr)
			} else {
				found.Add(1)
			}

		}(addrStr)
	}

	wg.Wait()

	finalCount := found.Load()
	if finalCount > 0 {
		globalLog(fmt.Sprintf("Topology Manager: Successfully connected to %d new peers", finalCount))
	} else {
		globalLog("Topology Manager: Found 0 active peers in this batch.")
	}
}
