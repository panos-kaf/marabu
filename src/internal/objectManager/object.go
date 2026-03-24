package objectManager

import (
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"time"

	"encoding/json"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

type T_HashID = messages.T_HashID

type PendingBlock struct {
	Block     *messages.T_Block
	Timestamp time.Time
	Peer      string
}

type ObjectManager struct {
	db           *leveldb.DB
	pendingFinds map[T_HashID][]chan messages.Object
	mutex        sync.Mutex

	PendingBlocks map[T_HashID][]PendingBlock
}

func NewObjectManager(path string) (*ObjectManager, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &ObjectManager{
		db:            db,
		pendingFinds:  make(map[T_HashID][]chan messages.Object),
		PendingBlocks: make(map[T_HashID][]PendingBlock),
	}, nil
}

func (om *ObjectManager) Exists(id T_HashID) (bool, error) {
	return om.db.Has([]byte(id), nil)
}

func (om *ObjectManager) Get(id T_HashID) (messages.Object, error) {
	data, err := om.db.Get([]byte(id), nil)
	if err != nil {
		return nil, err
	}

	// Probe the type field
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}

	switch probe.Type {
	case "transaction":
		var probe struct {
			Height *int `json:"height"`
		}
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil, err
		}

		if probe.Height != nil {
			var cb messages.T_CoinbaseTransaction
			if err := json.Unmarshal(data, &cb); err != nil {
				return nil, err
			}
			return &cb, nil
		} else {
			var tx messages.T_Transaction
			if err := json.Unmarshal(data, &tx); err != nil {
				return nil, err
			}
			return &tx, nil
		}
	case "block":
		var blk messages.T_Block
		if err := json.Unmarshal(data, &blk); err != nil {
			return nil, err
		}
		return &blk, nil

	default:
		return nil, fmt.Errorf("Retrieved object of unknown type: %s", probe.Type)
	}
}

func (om *ObjectManager) Put(object messages.Object) (T_HashID, error) {
	canon, err := messages.Canonicalize(object)
	if err != nil {
		return "", err
	}
	id, err := crypto.HashString(canon)
	if err != nil {
		return "", err
	}

	// Marshal and store
	data, err := json.Marshal(object)
	if err != nil {
		return "", err
	}

	if err := om.db.Put([]byte(id), data, nil); err != nil {
		return "", err
	}

	hashID := T_HashID(id)
	return hashID, nil
}

// Implement FindObject with channels for pending requests
func (om *ObjectManager) FindObject(id T_HashID) (messages.Object, error) {
	// First, try to get the object immediately
	obj, err := om.Get(id)
	if err == nil {
		return obj, nil
	}

	// If not found, set up a pending channel
	om.mutex.Lock()
	ch := make(chan messages.Object, 1)
	om.pendingFinds[id] = append(om.pendingFinds[id], ch)
	om.mutex.Unlock()

	// Wait for the object to be provided by someone else (e.g., after a network fetch)
	result, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("find for object %s was cancelled", id)
	}
	return result, nil
}

// When you later receive the object (e.g., after a network fetch and Put):
func (om *ObjectManager) notifyWaiters(id T_HashID, obj messages.Object) {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	for _, ch := range om.pendingFinds[id] {
		ch <- obj
		close(ch)
	}
	delete(om.pendingFinds, id)
}

// peer received a block that waits for object with id missingID.
func (om *ObjectManager) AddPendingBlock(peer string, missingID T_HashID, block *messages.T_Block) {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.PendingBlocks[missingID] = append(om.PendingBlocks[missingID], PendingBlock{
		Block:     block,
		Timestamp: time.Now(),
		Peer:      peer,
	})
}

func (om *ObjectManager) CheckPendingBlocks() []struct {
	Peer  string
	Block *messages.T_Block
	Txid  T_HashID
} {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	now := time.Now()
	var expired []struct {
		Peer  string
		Block *messages.T_Block
		Txid  T_HashID
	}

	timeout := 5 * time.Second

	for txid, blocks := range om.PendingBlocks {
		var stillPending []PendingBlock
		for _, blk := range blocks {
			if now.Sub(blk.Timestamp) > timeout {
				expired = append(expired, struct {
					Peer  string
					Block *messages.T_Block
					Txid  T_HashID
				}{blk.Peer, blk.Block, txid})
			} else {
				stillPending = append(stillPending, blk)
			}
		}
		if len(stillPending) == 0 {
			delete(om.PendingBlocks, txid)
		} else {
			om.PendingBlocks[txid] = stillPending
		}
	}
	return expired
}
