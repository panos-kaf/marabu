package peer

import (
	"errors"
	"marabu/internal/types"
	"sync"
)

var (
	ErrOutboundFull      = errors.New("outbound connections capped")
	ErrOutboundDuplicate = errors.New("outbound connection already exists")
)

// ConnectionManager handles thread-safe tracking of all active TCP connections
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

// Add safely registers a new peer and increments the counters
func (cm *ConnectionManager) Add(p *Peer) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if p.origin == types.Outbound && !p.isPersistent && cm.outboundCount >= MaxOutbound {
		return ErrOutboundFull
	}

	if _, exists := cm.peers[p.addr]; exists {
		return ErrOutboundDuplicate
	}
	cm.peers[p.addr] = p

	cm.peerCounter++
	p.ID = cm.peerCounter

	if p.isPersistent {
		cm.persistentCount++
	}

	switch p.origin {
	case types.Inbound:
		cm.inboundCount++
	case types.Outbound:
		cm.outboundCount++
	}

	return nil
}

// Remove safely deletes a peer and decrements the counters
func (cm *ConnectionManager) Remove(p *Peer) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Ensure we dont double delete
	if _, exists := cm.peers[p.addr]; !exists {
		return
	}

	delete(cm.peers, p.addr)

	if p.isPersistent {
		cm.persistentCount--
	}

	switch p.origin {
	case types.Inbound:
		cm.inboundCount--
	case types.Outbound:
		cm.outboundCount--
	}
}

func (cm *ConnectionManager) Exists(addr string) (*Peer, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	p, exists := cm.peers[addr]

	return p, exists

}

// Fetch returns a list of all active peers (useful for broadcast)
func (cm *ConnectionManager) GetAll() []*Peer {
	cm.mutex.RLock() // Use RLock for reading!
	defer cm.mutex.RUnlock()

	list := make([]*Peer, 0, len(cm.peers))
	for _, p := range cm.peers {
		list = append(list, p)
	}
	return list
}

// GetActiveIPs returns a map of all currently connected IP addresses
func (cm *ConnectionManager) GetActiveIPs() map[string]bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	activeIPs := make(map[string]bool, len(cm.peers))
	for ip := range cm.peers {
		activeIPs[ip] = true
	}

	return activeIPs
}

// GetDisposable returns all peers that are not persistent (aka the bootstrap peers)
func (cm *ConnectionManager) GetDisposable() []*Peer {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	list := make([]*Peer, 0, len(cm.peers))
	for _, p := range cm.peers {
		if p.origin == types.Outbound && !p.isPersistent {
			list = append(list, p)
		}
	}
	return list
}

// GetCounts returns the current stats
func (cm *ConnectionManager) GetCounts() (inbound, outbound, persistent int) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.inboundCount, cm.outboundCount, cm.persistentCount
}
