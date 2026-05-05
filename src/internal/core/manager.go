package core

import (
	"errors"
	"fmt"
	"marabu/internal/logs"
	"marabu/internal/types"
	"sync/atomic"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

type Manager struct {
	db *database

	isSynced          atomic.Bool
	chaintipsReceived atomic.Int32
}

var ErrNotFound = errors.New("object not found in database")

func NewManager(dbPath string) *Manager {
	db, err := newDatabase(dbPath)
	if err != nil {
		panic(fmt.Errorf("failed to create database: %v", err))
	}

	m := &Manager{db: db}

	m.isSynced.Store(false)

	return m

}

func (m *Manager) IsSynced() bool {
	return m.isSynced.Load()
}

func (m *Manager) IncrementChaintipsReceived() {
	m.chaintipsReceived.Add(1)
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

func (m *Manager) InitializeMempool() {

	savedMempool, err := m.db.loadMempool()
	if err != nil || len(savedMempool) == 0 {
		return
	}

	cnt := 0
	for _, entry := range savedMempool {

		// This guarantees the UTXOs are still unspent on the longest chain.
		_, errCode, err := m.ValidateTransaction(entry.Tx)

		if err != nil || errCode != types.E_NONE {
			m.db.removeMempoolEntry(entry.TxID)
		} else {
			m.db.mempoolMutex.Lock()

			m.db.mempool[entry.TxID] = entry
			for _, input := range entry.Tx.Inputs {
				key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
				m.db.mempoolSpentInputs[key] = entry.TxID
			}

			m.db.mempoolMutex.Unlock()
			cnt++
		}
	}

	logs.GlobalLog(fmt.Sprintf("Mempool initialized: Loaded %d valid transactions from disk.", cnt))
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

func (m *Manager) IsInputSpent(outpoint OutpointKey) bool {

	m.db.mempoolMutex.RLock()
	_, isPendingSpend := m.db.mempoolSpentInputs[outpoint]
	m.db.mempoolMutex.RUnlock()

	return isPendingSpend
}

func (m *Manager) IsNeededForPendingBlock(id types.HashID) bool {
	return m.db.isNeededForPendingBlock(id)
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

func (m *Manager) SyncNodeState(requestMempool func()) {

	// give the node 30 seconds to find a peer and start syncing
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			hasTips := m.chaintipsReceived.Load() > 0
			isDoneDownloading := m.db.getPendingBlocksCount() == 0

			if hasTips && isDoneDownloading {
				m.isSynced.Store(true)
				logs.GlobalLog("=== INITIAL BLOCK DOWNLOAD COMPLETE. Node is Synced. ===")

				// broadcast get mempool
				requestMempool()
				return
			}

		case <-timeout:
			if m.db.getPendingBlocksCount() == 0 {
				m.isSynced.Store(true)
				logs.GlobalLog("=== SYNC TIMEOUT: Assuming local tip is the highest. ===")
				requestMempool()
				return
			}
		}
	}
}

func (m *Manager) HandleReorg(oldTip types.HashID, newTip types.HashID) {
	logs.GlobalLog(fmt.Sprintf("=== CHAIN REORG DETECTED ===\nSwitching from %s to %s\n", oldTip, newTip))

	var deadBlocks []types.HashID
	var newBlocks []types.HashID

	currOld := oldTip
	currNew := newTip

	// find the common ancestor
	for currOld != currNew {
		utxoOld, _ := m.GetUTXO(currOld)
		utxoNew, _ := m.GetUTXO(currNew)

		if utxoOld.Height > utxoNew.Height {
			deadBlocks = append(deadBlocks, currOld)
			oldBlkObj, _ := m.GetObject(currOld)
			currOld = *oldBlkObj.(*types.Block).Previd
		} else if utxoNew.Height > utxoOld.Height {
			newBlocks = append(newBlocks, currNew)
			newBlkObj, _ := m.GetObject(currNew)
			currNew = *newBlkObj.(*types.Block).Previd
		} else {
			deadBlocks = append(deadBlocks, currOld)
			newBlocks = append(newBlocks, currNew)
			oldBlkObj, _ := m.GetObject(currOld)
			newBlkObj, _ := m.GetObject(currNew)
			currOld = *oldBlkObj.(*types.Block).Previd
			currNew = *newBlkObj.(*types.Block).Previd
		}
	}

	// push dead transactions back to the hard drive
	for _, blkID := range deadBlocks {
		blkObj, _ := m.GetObject(blkID)
		for _, txid := range blkObj.(*types.Block).Txids {
			txObj, err := m.GetObject(txid)
			if err == nil {
				if tx, ok := txObj.(*types.Transaction); ok {

					fee, _, _ := m.ValidateTransaction(tx)
					// Add it to the DB (we don't care if it's invalid right now, the sweep will catch it)
					m.AddToMempool(tx, fee)
				}
			}
		}
	}

	// Remove the new chains transactions from the hard drive
	for _, blkID := range newBlocks {
		blkObj, _ := m.GetObject(blkID)
		for _, txid := range blkObj.(*types.Block).Txids {
			m.RemoveFromMempool(txid)
		}
	}

	// flush the mempool struct and let InitializeMempool rebuild it
	m.db.mempoolMutex.Lock()
	m.db.mempool = make(map[types.HashID]MempoolEntry)
	m.db.mempoolSpentInputs = make(map[OutpointKey]types.HashID)
	m.db.mempoolMutex.Unlock()

	fmt.Println("Sweeping mempool against new chaintip...")
	m.InitializeMempool()
}
