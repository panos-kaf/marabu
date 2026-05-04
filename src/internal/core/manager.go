package core

import (
	"errors"
	"fmt"
	"marabu/internal/types"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

type Manager struct {
	db *database
}

var ErrNotFound = errors.New("object not found in database")

func NewManager(dbPath string) *Manager {
	db, err := newDatabase(dbPath)
	if err != nil {
		panic(fmt.Errorf("failed to create database: %v", err))
	}

	return &Manager{db: db}
}

// CommitObject safely writes the validated object to the database.
func (m *Manager) CommitObject(obj types.Object, result ValidationResult) error {
	// Store the raw object
	if _, err := m.db.putObject(obj); err != nil {
		return fmt.Errorf("failed to store object: %v", err)
	}

	// Apply type-specific database updates
	switch o := obj.(type) {
	case *types.Transaction:
		if err := m.db.putFee(result.ObjID, result.Fee); err != nil {
			return fmt.Errorf("error storing fee: %v", err)
		}

	case *types.Block:
		if result.NewUTXOSet != nil {
			if err := m.db.putUTXO(result.ObjID, *result.NewUTXOSet); err != nil {
				return fmt.Errorf("failed to save UTXO state: %v", err)
			}
		}
		if result.IsNewTip {
			if err := m.db.putChaintip(result.ObjID, result.NewHeight); err != nil {
				return fmt.Errorf("failed to update chaintip: %v", err)
			}
		}

		// we commited the block to the database, so we can remove its transactions from the mempool
		for _, txid := range o.Txids {
			if err := m.db.removeMempoolEntry(txid); err != nil {
				return fmt.Errorf("failed to remove transaction from mempool: %v", err)
			}
		}
	}
	return nil
}

func (m *Manager) ExistsObject(id types.HashID) (bool, error) {
	return m.db.existsObject(id)
}

// ExistsChaintip lets the network layer know if we have a chaintip yet
func (m *Manager) ExistsChaintip() (bool, error) {
	return m.db.existsChaintip()
}

// GetObject allows the network layer to fetch an object to send to a peer
func (m *Manager) GetObject(id types.HashID) (types.Object, error) {
	obj, err := m.db.getObject(id)

	// Check if it's a not found error, and return our sentinel error instead
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, ErrNotFound
	}

	return obj, err
}

func (m *Manager) GetAllObjectIDs() ([]types.HashID, error) {
	return m.db.getAllObjectIDs()
}

func (m *Manager) GetUTXO(id types.HashID) (*UTXOSet, error) {
	UTXOSet, err := m.db.getUTXO(id)

	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("DB error while fetching UTXO: %v", err)
	}
	return &UTXOSet, nil
}

// GetChaintip allows the network layer to fetch the current tip to gossip
func (m *Manager) GetChaintip() (types.HashID, uint64, error) {
	chaintip, height, err := m.db.getChaintip()
	if errors.Is(err, leveldb.ErrNotFound) {
		return types.DUMMY_HASH, 0, ErrNotFound
	} else if err != nil {
		return types.DUMMY_HASH, 0, fmt.Errorf("DB error while fetching chaintip: %v", err)
	}
	return chaintip, height, nil
}

func (m *Manager) AddToMempool(tx *types.Transaction, fee types.Picabu) error {
	if err := m.db.addMempoolEntry(tx, fee); err != nil {
		return fmt.Errorf("failed to add transaction to mempool: %v", err)
	}
	return nil
}

func (m *Manager) RemoveFromMempool(txid types.HashID) error {
	if err := m.db.removeMempoolEntry(txid); err != nil {
		return fmt.Errorf("failed to remove transaction from mempool: %v", err)
	}
	return nil
}

func (m *Manager) ExistsInMempool(txid types.HashID) (bool, error) {
	return m.db.existsInMempool(txid)
}

func (m *Manager) GetMempoolEntries() []MempoolEntry {
	return m.db.getMempoolEntries()
}

func (m *Manager) FetchPendingBlocks(resolvedObjID types.HashID) []PendingBlock {
	return m.db.fetchAndClearPendingBlocks(resolvedObjID)
}

func (m *Manager) CleanupPendingBlocks(notifyPeer func(peerAddr string, txid types.HashID)) {
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {
		expiredBlocks := m.db.checkPendingBlocks()

		for _, expired := range expiredBlocks {
			// Trigger the callback! The Manager doesn't care what happens next.
			notifyPeer(expired.Peer, expired.Txid)
		}
	}
}
