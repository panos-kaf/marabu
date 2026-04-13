package storage

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

type Store struct {
	db           *leveldb.DB
	pendingFinds map[T_HashID][]chan messages.Object
	mutex        sync.Mutex

	PendingBlocks map[T_HashID][]PendingBlock
}

func NewStore(path string) (*Store, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:            db,
		pendingFinds:  make(map[T_HashID][]chan messages.Object),
		PendingBlocks: make(map[T_HashID][]PendingBlock),
	}, nil
}

func (s *Store) ExistsObject(id T_HashID) (bool, error) {
	return s.db.Has([]byte(id), nil)
}

func (s *Store) GetObject(id T_HashID) (messages.Object, error) {
	data, err := s.db.Get([]byte(id), nil)
	if err != nil {
		return nil, err
	}

	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}

	switch probe.Type {
	case "transaction":
		var heightProbe struct {
			Height *int `json:"height"`
		}
		if err := json.Unmarshal(data, &heightProbe); err != nil {
			return nil, err
		}

		if heightProbe.Height != nil {
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

func (s *Store) PutObject(object messages.Object) (T_HashID, error) {
	canon, err := messages.Canonicalize(object)
	if err != nil {
		return "", err
	}
	id, err := crypto.HashString(canon)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(object)
	if err != nil {
		return "", err
	}

	if err := s.db.Put([]byte(id), data, nil); err != nil {
		return "", err
	}

	return T_HashID(id), nil
}

func (s *Store) FindObject(id T_HashID) (messages.Object, error) {
	obj, err := s.GetObject(id)
	if err == nil {
		return obj, nil
	}

	s.mutex.Lock()
	ch := make(chan messages.Object, 1)
	s.pendingFinds[id] = append(s.pendingFinds[id], ch)
	s.mutex.Unlock()

	result, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("find for object %s was cancelled", id)
	}
	return result, nil
}

func (s *Store) notifyWaiters(id T_HashID, obj messages.Object) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, ch := range s.pendingFinds[id] {
		ch <- obj
		close(ch)
	}
	delete(s.pendingFinds, id)
}

func (s *Store) AddPendingBlock(peer string, missingID T_HashID, block *messages.T_Block) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.PendingBlocks[missingID] = append(s.PendingBlocks[missingID], PendingBlock{
		Block:     block,
		Timestamp: time.Now(),
		Peer:      peer,
	})
}

func (s *Store) CheckPendingBlocks() []struct {
	Peer  string
	Block *messages.T_Block
	Txid  T_HashID
} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	var expired []struct {
		Peer  string
		Block *messages.T_Block
		Txid  T_HashID
	}

	timeout := 5 * time.Second

	for txid, blocks := range s.PendingBlocks {
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
			delete(s.PendingBlocks, txid)
		} else {
			s.PendingBlocks[txid] = stillPending
		}
	}
	return expired
}
