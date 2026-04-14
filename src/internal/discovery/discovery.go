package discovery

import (
	"encoding/csv"
	"errors"
	"fmt"
	"marabu/internal/logs"
	"marabu/internal/types"

	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	BOOTSTRAP_PEERS = types.Peers{
		"95.179.158.137:18018",
		"95.179.132.22:18018",
		"45.32.235.245:18018",
	}
	PEERS_FILE      = filepath.Join(".", "db", "peers.csv")
	KnownPeers      = make(map[types.Peer]string)
	KnownPeersMutex sync.Mutex
)

func init() {
	loadPeers()
	if _, err := os.Stat(PEERS_FILE); errors.Is(err, os.ErrNotExist) {
		savePeers()
	}
}

// Load peers from file and bootstrap list
func loadPeers() {
	KnownPeersMutex.Lock()
	for _, peer := range BOOTSTRAP_PEERS {
		KnownPeers[peer] = "bootstrap"
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
		KnownPeers[types.Peer(rec[0])] = rec[1]
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
	w.Write([]string{"Address", "Source"})
	for peer, source := range KnownPeers {
		if peer != types.PEER_INVALID {
			w.Write([]string{string(peer), string(source)})
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

// Add new peers
func AppendPeers(peers types.Peers, source string) {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()
	newPeers := 0
	for _, peer := range peers {
		if peer != types.PEER_INVALID {
			if _, exists := KnownPeers[peer]; !exists {
				newPeers++
				KnownPeers[peer] = source
				logs.GlobalLog(fmt.Sprintf("Added new peer: %s from source %s", peer, source))
			}
		}
	}
	if newPeers > 0 {
		savePeers()
		logs.GlobalLog(fmt.Sprintf("Saved %d peers to disk...", newPeers))
	} else {
		logs.GlobalLog("No new peers to add.")
	}
}

// Select random peers per source
func SelectRandomPeersPerSource(count int) []string {
	KnownPeersMutex.Lock()
	defer KnownPeersMutex.Unlock()

	peersBySource := make(map[string][]string)
	for peer, source := range KnownPeers {
		if peer != types.PEER_INVALID {
			peersBySource[source] = append(peersBySource[source], string(peer))
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
