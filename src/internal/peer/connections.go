package peer

import (
	"errors"
	"fmt"
	"marabu/internal/types"
	"net"
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

	mutedPeers  map[string]bool
	bannedPeers map[string]bool
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		peers:       make(map[string]*Peer),
		mutedPeers:  make(map[string]bool),
		bannedPeers: make(map[string]bool),
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

// Mute spammy peers

func (cm *ConnectionManager) MutePeer(peer string) {

	globalLog(fmt.Sprintf("Muted peer %s", peer))

	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.mutedPeers[peer] = true
}

func (cm *ConnectionManager) UnmutePeer(peer string) {

	globalLog(fmt.Sprintf("Unmuted peer %s", peer))

	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.mutedPeers, peer)
}

func (cm *ConnectionManager) IsMuted(peer string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.mutedPeers[peer]
}

type PeerRecord struct {
	Target string
	IP     string
	Agent  string
}

func (cm *ConnectionManager) GetMuted() []PeerRecord {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var records []PeerRecord
	for target := range cm.mutedPeers {

		record := PeerRecord{
			Target: target,
			IP:     "Offline / Unknown",
			Agent:  "Offline / Unknown",
		}

		// guess what the rule is based on formatting
		if net.ParseIP(target) != nil {
			record.IP = target
		} else {
			record.Agent = target
		}

		// Scan active peers to fill in the blanks!
		for _, p := range cm.peers {

			if p.host == target || p.agent == target {
				record.IP = p.host
				if p.agent != "" {
					record.Agent = p.agent
				}
				break
			}
		}

		records = append(records, record)
	}

	return records
}

func (cm *ConnectionManager) BanPeer(target string) {
	cm.mutex.Lock()
	cm.bannedPeers[target] = true

	// Find any currently connected peers that match this new ban rule
	var violators []*Peer
	for _, p := range cm.peers {
		if p.host == target || p.agent == target {
			violators = append(violators, p)
		}
	}
	cm.mutex.Unlock() // Unlock before kicking to prevent deadlocks!

	// Kick them out immediately
	for _, p := range violators {
		p.Disconnect()
	}

	globalLog(fmt.Sprintf("Banned %s", target))
}

func (cm *ConnectionManager) UnbanPeer(target string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.bannedPeers, target)

	globalLog(fmt.Sprintf("Unbanned %s", target))

}

func (cm *ConnectionManager) IsBanned(target string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.bannedPeers[target]
}

func (cm *ConnectionManager) GetBanned() []PeerRecord {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var records []PeerRecord
	for target := range cm.bannedPeers {
		record := PeerRecord{
			Target: target,
			IP:     "Offline / Unknown",
			Agent:  "Offline / Unknown",
		}

		if net.ParseIP(target) != nil {
			record.IP = target
		} else {
			record.Agent = target
		}

		// Check if they are somehow connected (they shouldn't be, but just in case)
		for _, p := range cm.peers {
			if p.host == target || p.agent == target {
				record.IP = p.host
				if p.agent != "" {
					record.Agent = p.agent
				}
				break
			}
		}
		records = append(records, record)
	}
	return records
}

// --- Bulk Clears ---

func (cm *ConnectionManager) UnmuteAll() int {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	count := len(cm.mutedPeers)
	cm.mutedPeers = make(map[string]bool) // Instantly flush the map
	return count
}

func (cm *ConnectionManager) UnbanAll() int {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	count := len(cm.bannedPeers)
	cm.bannedPeers = make(map[string]bool) // Instantly flush the map
	return count
}
