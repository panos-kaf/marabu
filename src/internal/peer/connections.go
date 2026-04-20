package peer

import (
	"sync"
)

// ConnectionManager handles thread-safe tracking of all active TCP connections.
type ConnectionManager struct {
	peers map[string]*Peer
	mutex sync.RWMutex

	inboundCount    int
	outboundCount   int
	persistentCount int

	// Strictly incrementing
	peerCounter int
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		peers: make(map[string]*Peer),
	}
}

var ConnManager = NewConnectionManager()

// Add safely registers a new peer and increments the correct counters
func (cm *ConnectionManager) Add(p *Peer) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.peers[p.addr] = p

	cm.peerCounter++
	p.ID = cm.peerCounter

	if p.isPersistent {
		cm.persistentCount++
	} else if p.role == "server" {
		cm.inboundCount++
	} else if p.role == "client" {
		cm.outboundCount++
	}
}

// Remove safely deletes a peer and decrements the correct counters
func (cm *ConnectionManager) Remove(p *Peer) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Ensure we don't double-delete
	if _, exists := cm.peers[p.addr]; !exists {
		return
	}

	delete(cm.peers, p.addr)

	if p.isPersistent {
		cm.persistentCount--
	} else if p.role == "server" {
		cm.inboundCount--
	} else if p.role == "client" {
		cm.outboundCount--
	}
}

func (cm *ConnectionManager) Exists(addr string) (*Peer, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	p, exists := cm.peers[addr]

	return p, exists

}

// Fetch returns a list of all active peers (useful for broadcasting)
func (cm *ConnectionManager) GetAll() []*Peer {
	cm.mutex.RLock() // Use RLock for reading!
	defer cm.mutex.RUnlock()

	list := make([]*Peer, 0, len(cm.peers))
	for _, p := range cm.peers {
		list = append(list, p)
	}
	return list
}

func (cm *ConnectionManager) GetDisposable() []*Peer {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	list := make([]*Peer, 0, len(cm.peers))
	for _, p := range cm.peers {
		if p.role == "client" && !p.isPersistent {
			list = append(list, p)
		}
	}
	return list
}

// GetCounts returns the current stats in O(1) time
func (cm *ConnectionManager) GetCounts() (inbound, outbound, persistent int) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.inboundCount, cm.outboundCount, cm.persistentCount
}
