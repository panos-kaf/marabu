package discovery

import (
	"encoding/csv"
	"errors"
	"fmt"
	"marabu/internal/logs"
	"marabu/internal/types"
	"sort"
	"strconv"

	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PeerRecord struct {
	Source   string
	Agent    string
	LastSeen int64
}

var (
	BOOTSTRAP_PEERS = types.Peers{
		"95.179.158.137:18018",
		"95.179.132.22:18018",
		"45.32.235.245:18018",
	}
	PEERS_FILE = filepath.Join(".", "db", "peers.csv")

	KnownPeers = make(map[types.Peer]PeerRecord)
	DeadPeers  = make(map[types.Peer]time.Time)

	KnownPeersMutex sync.Mutex
)

func init() {
	loadPeers()
	if _, err := os.Stat(PEERS_FILE); errors.Is(err, os.ErrNotExist) {
		savePeers()
	}

	go DeadPeersTimer()
}

// Load peers from file and bootstrap list
func loadPeers() {
	KnownPeersMutex.Lock()
	for _, peer := range BOOTSTRAP_PEERS {
		KnownPeers[peer] = PeerRecord{Source: "bootstrap", Agent: "unknown"}
	}
	KnownPeersMutex.Unlock()
	file, err := os.Open(PEERS_FILE)
	if err != nil {
		return
	}
	defer file.Close()
	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return
	}
	for _, rec := range records {
		if len(rec) < 2 || rec[0] == "Address" {
			continue
		}

		// Safely handle the 3rd column (for backwards compatibility with your old CSV!)
		agent := "unknown"
		if len(rec) > 2 {
			agent = rec[2]
		}

		var lastSeen int64 = 0
		if len(rec) > 3 {
			lastSeen, _ = strconv.ParseInt(rec[3], 10, 64)
		}

		KnownPeers[types.Peer(rec[0])] = PeerRecord{Source: rec[1], Agent: agent, LastSeen: lastSeen}
	}
}

// Save peers to file
func savePeers() {
	file, err := os.Create(PEERS_FILE)
	if err != nil {
		logs.GlobalLog(fmt.Sprintf("Failed to save peers file: %v", err))
		return
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()
	w.Write([]string{"Address", "Source", "Agent", "LastSeen"})
	for peer, record := range KnownPeers {
		if peer != types.PEER_INVALID {
			w.Write([]string{
				string(peer),
				record.Source,
				record.Agent,
				strconv.FormatInt(record.LastSeen, 10),
			})
		}
	}
}

// Get all known peers
func GetKnownPeers() types.Peers {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()
	keys := make(types.Peers, 0, len(KnownPeers))
	for k := range KnownPeers {
		keys = append(keys, k)
	}
	return keys
}

func GetKnownPeersCount() int {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	return len(KnownPeers)
}

// Add new peers
func AppendPeers(peers types.Peers, source string) {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()
	newPeers := 0
	for _, peer := range peers {
		if peer != types.PEER_INVALID {

			if _, isDead := DeadPeers[peer]; isDead {
				continue
			}

			if _, exists := KnownPeers[peer]; !exists {
				newPeers++
				KnownPeers[peer] = PeerRecord{Source: source, Agent: "unknown"}
				logs.GlobalLog(fmt.Sprintf("Discovered new peer: %s from %s", peer, source))
			}
		}
	}
	if newPeers > 0 {
		savePeers()
		logs.GlobalLog(fmt.Sprintf("Saved %d peers to disk...", newPeers))
	} else {
		// logs.GlobalLog("No new peers to store.")
	}
}

func RemovePeer(peerAddr string) {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	p := types.Peer(peerAddr)
	delete(KnownPeers, p)
	DeadPeers[p] = time.Now()
}

func DeadPeersTimer() {

	ticker := time.NewTicker(15 * time.Minute)

	for range ticker.C {
		KnownPeersMutex.Lock()

		now := time.Now()
		cleared := 0

		for peer, timeOfDeath := range DeadPeers {
			if now.Sub(timeOfDeath) > 12*time.Hour {
				delete(DeadPeers, peer)
				cleared++
			}
		}

		KnownPeersMutex.Unlock()

		if cleared > 0 {
			logs.GlobalLog(fmt.Sprintf("Discovery: Swept %d expired tombstones from the graveyard.", cleared))
		}
	}
}

func UpdatePeer(peerAddr string, agent string) {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	p := types.Peer(peerAddr)
	record, exists := KnownPeers[p]

	if !exists {
		// Just in case they dial us before we discovered them via gossiping
		KnownPeers[p] = PeerRecord{Source: "direct"}
	}
	record.Agent = agent
	record.LastSeen = time.Now().Unix()

	KnownPeers[p] = record
	savePeers()
}

func SelectRandomPeers(count int, ignoreIPs map[string]bool) []string {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	var validPeers []string
	for peer := range KnownPeers {
		if peer != types.PEER_INVALID {
			peerStr := string(peer)

			if !ignoreIPs[peerStr] {
				validPeers = append(validPeers, peerStr)
			}
		}
	}

	if len(validPeers) <= count {
		return validPeers
	}

	selected := make([]string, 0, count)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	perm := rng.Perm(len(validPeers))
	for i := range count {
		selected = append(selected, validPeers[perm[i]])
	}

	return selected
}

// Select count random peers per source
func SelectRandomPeersPerSource(count int, ignoreIPs map[string]bool) []string {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	peersBySource := make(map[string][]string)
	for peer, record := range KnownPeers {
		if peer != types.PEER_INVALID {
			peerStr := string(peer)

			if ignoreIPs[peerStr] {
				continue
			}
			peersBySource[record.Source] = append(peersBySource[record.Source], peerStr)
		}
	}
	selected := []string{}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, peers := range peersBySource {
		if len(peers) <= count {
			selected = append(selected, peers...)
		} else {
			perm := rng.Perm(len(peers))
			for i := range count {
				selected = append(selected, peers[perm[i]])
			}
		}
	}
	return selected
}

// Select count peers, 80% of which are the most recently seen peers, and 20% are randomly selected
func SelectRecentPeers(count int, ignoreIPs map[string]bool) []string {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	type peerData struct {
		addr     string
		lastSeen int64
	}
	var validPeers []peerData

	for peer, record := range KnownPeers {
		if peer != types.PEER_INVALID {
			peerStr := string(peer)
			if !ignoreIPs[peerStr] {
				validPeers = append(validPeers, peerData{addr: peerStr, lastSeen: record.LastSeen})
			}
		}
	}

	// Sort (newest LastSeen first)
	sort.Slice(validPeers, func(i, j int) bool {
		return validPeers[i].lastSeen > validPeers[j].lastSeen
	})

	selected := make([]string, 0, count)

	if len(validPeers) <= count {
		for _, p := range validPeers {
			selected = append(selected, p.addr)
		}
		return selected
	}

	recentCount := int(float64(count) * 0.8)
	randomCount := count - recentCount

	for i := range recentCount {
		selected = append(selected, validPeers[i].addr)
	}

	remaining := validPeers[recentCount:]
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	perm := rng.Perm(len(remaining))

	for i := range randomCount {
		selected = append(selected, remaining[perm[i]].addr)
	}

	return selected
}
