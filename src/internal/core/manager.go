package core

import (
	"context"
	"errors"
	"fmt"
	"marabu/internal/logs"
	"marabu/internal/types"
	"sync"
	"sync/atomic"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

type NodeConfig struct {
	DBPath      string
	ServerPort  int
	AgentName   types.BuString
	Note        types.BuString
	StudentIDs  types.BuStrings
	MiningCores int
	PubKey      types.HashID
}

type Manager struct {
	db *database

	isSynced          atomic.Bool
	chaintipsReceived atomic.Int32
	lastSyncProgress  atomic.Int64

	minerCancel context.CancelFunc

	miningActive atomic.Bool
	miningStart  atomic.Int64
	miningHashes atomic.Uint64

	configMutex sync.RWMutex // Protects the config at runtimes
	config      NodeConfig
}

var ErrNotFound = errors.New("object not found in database")

func NewManager(config NodeConfig) *Manager {
	db, err := newDatabase(config.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to create database: %v", err))
	}

	m := &Manager{db: db, config: config}

	m.isSynced.Store(false)

	return m

}

func (m *Manager) SetNote(note types.BuString) {
	m.configMutex.Lock()
	m.config.Note = note
	m.configMutex.Unlock()
}

func (m *Manager) Config() NodeConfig {
	m.configMutex.RLock()
	defer m.configMutex.RUnlock()
	return m.config
}

func (m *Manager) IsSynced() bool {
	return m.isSynced.Load()
}

func (m *Manager) IncrementChaintipsReceived() {
	m.chaintipsReceived.Add(1)
}

// CommitObject applies state changes (DB and Mempool) and returns a boolean
// indicating if the object should be gossiped to the network.
func (m *Manager) CommitObject(obj types.Object, result ValidationResult) (bool, types.ErrorCode, error) {

	// store the raw object to the hard drive
	if _, err := m.db.putObject(obj); err != nil {
		return false, types.E_INTERNAL_ERROR, fmt.Errorf("failed to store object: %v", err)
	}

	switch o := obj.(type) {
	case *types.Transaction:
		return m.commitTransaction(o, result)
	case *types.CoinbaseTransaction:
		// Coinbase transactions are not gossiped, so we can skip the mempool logic and just return
		return false, types.E_NONE, nil
	case *types.Block:
		return m.commitBlock(o, result)
	default:
		return false, types.E_INTERNAL_ERROR, fmt.Errorf("unknown object type")
	}
}

func (m *Manager) commitTransaction(tx *types.Transaction, result ValidationResult) (bool, types.ErrorCode, error) {
	// Store the fee
	if err := m.db.putFee(result.ObjID, result.Fee); err != nil {
		return false, types.E_INTERNAL_ERROR, fmt.Errorf("error storing fee: %v", err)
	}

	// Check for double spends in current mempool
	isMempoolConflict := false
	for _, input := range tx.Inputs {
		outpoint := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
		if m.IsInputSpent(outpoint) {
			isMempoolConflict = true
			break
		}
	}

	// Decide if it goes into the mempool
	if !isMempoolConflict {
		// Ensure it belongs to our active chaintip
		if m.IsInputInUTXO(tx) {
			m.AddToMempool(tx, result.Fee)
			return true, types.E_NONE, nil // Gossip!
		} else {
			logs.GlobalLog(fmt.Sprintf("Tx %s saved, but rejected from mempool (not on active chain).", result.ObjID))
			return false, types.E_INVALID_TX_OUTPOINT, nil
		}
	} else {
		logs.GlobalLog(fmt.Sprintf("Tx %s saved, but withheld from mempool due to conflict.", result.ObjID))
		return false, types.E_INVALID_TX_OUTPOINT, nil
	}

	// return false, nil // Do not gossip
}

func (m *Manager) commitBlock(blk *types.Block, result ValidationResult) (bool, types.ErrorCode, error) {
	if result.NewUTXOSet != nil {
		if err := m.db.putUTXO(result.ObjID, *result.NewUTXOSet); err != nil {
			return false, types.E_INTERNAL_ERROR, fmt.Errorf("failed to save UTXO state: %v", err)
		}
	}

	// Check for Reorg / New Tip
	var isReorg bool
	var oldTip types.HashID

	if result.IsNewTip {
		oldTip, _, err := m.GetChaintip()
		if err == nil && oldTip != *blk.Previd {
			isReorg = true
		}

		if err := m.db.putChaintip(result.ObjID, result.NewHeight); err != nil {
			return false, types.E_INTERNAL_ERROR, fmt.Errorf("failed to update chaintip: %v", err)
		}

		if m.minerCancel != nil {
			m.minerCancel() // Stop the miner if it's running. It will be restarted on the next loop.
		}
	}

	// Clean up the mempool
	if isReorg {
		m.HandleReorg(oldTip, result.ObjID)
	} else {
		for _, txid := range blk.Txids {
			m.RemoveFromMempool(txid)
		}
	}

	return true, types.E_NONE, nil // Always gossip valid blocks
}

func (m *Manager) SetMiningCancel(cancelFunc context.CancelFunc) {
	m.minerCancel = cancelFunc
}

func (m *Manager) ExistsObject(id types.HashID) (bool, error) {
	return m.db.existsObject(id)
}

func (m *Manager) StoreObject(object types.Object) error {
	_, err := m.db.putObject(object)
	return err
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

// IsInputInUTXO checks if a transaction's inputs exist in the active UTXOSet
func (m *Manager) IsInputInUTXO(tx *types.Transaction) bool {
	tip, _, err := m.GetChaintip()
	if err != nil {
		return false // No tip yet. nothing is spendable
	}

	activeUTXOs, err := m.GetUTXO(tip)
	if err != nil {
		return false
	}

	for _, input := range tx.Inputs {
		key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
		if _, exists := activeUTXOs.UTXOs[key]; !exists {
			return false // Input is not in the longest chain
		}
	}

	return true
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
		result := m.ValidateTransaction(entry.Tx)

		if result.Error != nil || result.ErrorCode != types.E_NONE || !m.IsInputInUTXO(entry.Tx) {
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

func (m *Manager) GetMempoolTxids() []types.HashID {
	return m.db.getMempoolTxids()
}

func (m *Manager) IsInputSpent(outpoint OutpointKey) bool {

	m.db.mempoolMutex.RLock()
	_, isPendingSpend := m.db.mempoolSpentInputs[outpoint]
	m.db.mempoolMutex.RUnlock()

	return isPendingSpend
}

func (m *Manager) AddPendingBlock(peer string, missingID types.HashID, block *types.Block) {
	m.db.addPendingBlock(peer, missingID, block)
}

func (m *Manager) IsNeededForPendingBlock(id types.HashID) bool {
	return m.db.isNeededForPendingBlock(id)
}

func (m *Manager) FetchPendingBlocks(resolvedObjID types.HashID) []PendingBlock {
	return m.db.fetchAndClearPendingBlocks(resolvedObjID)
}

func (m *Manager) ResetSyncTimer() {
	m.lastSyncProgress.Store(time.Now().Unix())
}

func (m *Manager) TimeSinceLastProgress() time.Duration {
	last := m.lastSyncProgress.Load()
	if last == 0 {
		return 5 * time.Minute // Force timeout if never set
	}
	return time.Since(time.Unix(last, 0))
}

func (m *Manager) CleanupPendingBlocks(notifyPeer func(peerAddr string, txid types.HashID)) {
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {

		if m.TimeSinceLastProgress() < 2*time.Minute {
			continue
		}

		expiredBlocks := m.db.checkPendingBlocks()

		for _, expired := range expiredBlocks {
			// Trigger the callback! The Manager doesn't care what happens next.
			notifyPeer(expired.Peer, expired.Txid)
		}
	}
}

func (m *Manager) GetMissingDependencies() []types.HashID {
	m.db.pendingMutex.RLock()
	defer m.db.pendingMutex.RUnlock()

	var missing []types.HashID
	for missingID := range m.db.pendingBlocks {
		missing = append(missing, missingID)
	}
	return missing
}

func (m *Manager) SyncNodeState(requestMempool func()) {

	// give the node 30 seconds to find a peer and start syncing
	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			// Check if we've received enough chaintips (from distinct peers) to be confident we're syncing from active peers, and that we've finished downloading all pending blocks.
			hasTips := m.chaintipsReceived.Load() > 3
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

		if utxoOld == nil || utxoNew == nil {
			logs.GlobalLog(fmt.Sprintf("Error during reorg handling: missing UTXO for %s or %s", currOld, currNew))
			return
		}

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

					result := m.ValidateTransaction(tx)
					// Add it to the DB (we don't care if it's invalid right now, the sweep will catch it)
					m.AddToMempool(tx, result.Fee)
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

func (m *Manager) SetMiningState(active bool) {
	m.miningActive.Store(active)
	if active {
		// Reset the clock and the counter when a new block starts
		m.miningStart.Store(time.Now().UnixNano())
		m.miningHashes.Store(0)
	}
}

func (m *Manager) SetMiningCores(cores int) {
	m.configMutex.Lock()
	m.config.MiningCores = cores
	m.configMutex.Unlock()

	// THE MAGIC TRICK:
	// Instantly kill the current mining job so it restarts with the new core count!
	if m.minerCancel != nil {
		m.minerCancel()
	}
}

// Update your getter so the Miner safely reads it
func (m *Manager) GetMiningCores() int {
	m.configMutex.RLock()
	defer m.configMutex.RUnlock()
	return m.config.MiningCores
}

func (m *Manager) AddMiningHashes(count uint64) {
	m.miningHashes.Add(count)
}

func (m *Manager) GetMiningStats() (active bool, elapsed time.Duration, hashrate float64) {
	active = m.miningActive.Load()
	if !active {
		return false, 0, 0
	}

	start := time.Unix(0, m.miningStart.Load())
	elapsed = time.Since(start)
	hashes := m.miningHashes.Load()

	if elapsed.Seconds() > 0 {
		hashrate = float64(hashes) / elapsed.Seconds()
	}

	return active, elapsed, hashrate
}
