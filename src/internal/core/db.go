package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"marabu/internal/crypto"
	"marabu/internal/serialization"
	"marabu/internal/types"

	"github.com/syndtr/goleveldb/leveldb"
)

type PendingBlock struct {
	Block     *types.Block
	Timestamp time.Time
	Peer      string
}

// database is private
type database struct {
	db *leveldb.DB

	pendingFinds  map[types.HashID][]chan types.Object
	pendingMutex  sync.Mutex
	pendingBlocks map[types.HashID][]PendingBlock

	mempool            map[types.HashID]MempoolEntry
	mempoolSpentInputs map[OutpointKey]types.HashID
	mempoolMutex       sync.RWMutex
}

func newDatabase(path string) (*database, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &database{
		db:                 db,
		pendingFinds:       make(map[types.HashID][]chan types.Object),
		pendingBlocks:      make(map[types.HashID][]PendingBlock),
		mempool:            make(map[types.HashID]MempoolEntry),
		mempoolSpentInputs: make(map[OutpointKey]types.HashID),
	}, nil
}

// Internal Data Structures

type OutpointKey struct {
	Txid  types.HashID
	Index int
}

type UTXOSet struct {
	BlockID types.HashID
	UTXOs   map[OutpointKey]types.TxOutput
	Height  uint64
}

type UTXORecord struct {
	Key   OutpointKey    `json:"key"`
	Value types.TxOutput `json:"value"`
}

type UTXOStorageData struct {
	BlockID types.HashID `json:"blockid"`
	Height  uint64       `json:"height"`
	Records []UTXORecord `json:"records"`
}

type MempoolEntry struct {
	TxID      types.HashID       `json:"txid"`
	Tx        *types.Transaction `json:"tx"`
	Fee       types.Picabu       `json:"fee"`
	Timestamp time.Time          `json:"timestamp"`
}

type Chaintip struct {
	BlockID types.HashID `json:"blockid"`
	Height  uint64       `json:"height"`
}

var chaintipKey = []byte("chaintip")

// Private CRUD Methods

func (d *database) existsObject(id types.HashID) (bool, error) {
	return d.db.Has([]byte(id), nil)
}

func (d *database) getObject(id types.HashID) (types.Object, error) {
	data, err := d.db.Get([]byte(id), nil)
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
			var cb types.CoinbaseTransaction
			if err := json.Unmarshal(data, &cb); err != nil {
				return nil, err
			}
			return &cb, nil
		} else {
			var tx types.Transaction
			if err := json.Unmarshal(data, &tx); err != nil {
				return nil, err
			}
			return &tx, nil
		}
	case "block":
		var blk types.Block
		if err := json.Unmarshal(data, &blk); err != nil {
			return nil, err
		}
		return &blk, nil

	default:
		return nil, fmt.Errorf("retrieved object of unknown type: %s", probe.Type)
	}
}

func (d *database) putObject(object types.Object) (types.HashID, error) {
	canon, err := serialization.Canonicalize(object)
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

	if err := d.db.Put([]byte(id), data, nil); err != nil {
		return "", err
	}

	return types.HashID(id), nil
}

func (d *database) getAllObjectIDs() ([]types.HashID, error) {
	var ids []types.HashID

	// Create an iterator over the entire database
	iter := d.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		keyStr := string(iter.Key())

		// Filter out our internal state tracking keys
		if keyStr == "chaintip" || strings.HasPrefix(keyStr, "utxo-") || strings.HasPrefix(keyStr, "fee-") {
			continue
		}

		ids = append(ids, types.HashID(keyStr))
	}

	return ids, iter.Error()
}

func (d *database) findObject(id types.HashID) (types.Object, error) {
	obj, err := d.getObject(id)
	if err == nil {
		return obj, nil
	}

	d.pendingMutex.Lock()
	ch := make(chan types.Object, 1)
	d.pendingFinds[id] = append(d.pendingFinds[id], ch)
	d.pendingMutex.Unlock()

	result, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("find for object %s was cancelled", id)
	}
	return result, nil
}

func (d *database) putFee(id types.HashID, fee types.Picabu) error {
	feeStr := (*big.Int)(&fee).String()
	key := []byte("fee-" + string(id))
	return d.db.Put(key, []byte(feeStr), nil)
}

func (d *database) getFee(id types.HashID) (types.Picabu, error) {
	key := []byte("fee-" + string(id))
	data, err := d.db.Get(key, nil)
	if err != nil {
		return types.ZERO_PICABU, fmt.Errorf("fee not found for tx %s", id)
	}
	fee := new(big.Int)
	_, ok := fee.SetString(string(data), 10)
	if !ok {
		return types.ZERO_PICABU, fmt.Errorf("invalid fee format for tx %s: %s", id, string(data))
	}
	return types.Picabu(*fee), nil
}

func (d *database) getUTXO(blockid types.HashID) (UTXOSet, error) {
	key := []byte("utxo-" + string(blockid))
	utxoRaw, err := d.db.Get(key, nil)
	if err != nil {
		return UTXOSet{}, err
	}

	var data UTXOStorageData
	err = json.Unmarshal(utxoRaw, &data)
	if err != nil {
		return UTXOSet{}, err
	}

	utxo := UTXOSet{
		BlockID: data.BlockID,
		Height:  data.Height,
		UTXOs:   make(map[OutpointKey]types.TxOutput),
	}

	for _, record := range data.Records {
		utxo.UTXOs[record.Key] = record.Value
	}

	return utxo, nil
}

func (d *database) putUTXO(blockid types.HashID, utxos UTXOSet) error {
	key := []byte("utxo-" + string(blockid))

	var records []UTXORecord
	for k, v := range utxos.UTXOs {
		records = append(records, UTXORecord{Key: k, Value: v})
	}

	data := UTXOStorageData{
		BlockID: utxos.BlockID,
		Height:  utxos.Height,
		Records: records,
	}

	utxoRaw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return d.db.Put(key, utxoRaw, nil)
}

func (d *database) existsChaintip() (bool, error) {
	return d.db.Has(chaintipKey, nil)
}

func (d *database) getChaintip() (blkid types.HashID, height uint64, err error) {
	chaintipRaw, err := d.db.Get(chaintipKey, nil)
	if err != nil {
		return types.DUMMY_HASH, 0, err
	}

	var chaintip Chaintip
	err = json.Unmarshal(chaintipRaw, &chaintip)
	if err != nil {
		return types.DUMMY_HASH, 0, err
	}

	return chaintip.BlockID, chaintip.Height, nil
}

func (d *database) putChaintip(blkid types.HashID, height uint64) error {
	chaintip := Chaintip{
		BlockID: blkid,
		Height:  height,
	}

	chaintipRaw, err := json.Marshal(chaintip)
	if err != nil {
		return err
	}

	return d.db.Put(chaintipKey, chaintipRaw, nil)
}

func (d *database) notifyWaiters(id types.HashID, obj types.Object) {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()
	for _, ch := range d.pendingFinds[id] {
		ch <- obj
		close(ch)
	}
	delete(d.pendingFinds, id)
}

func (d *database) addMempoolEntry(tx *types.Transaction, fee types.Picabu) error {

	txid, err := crypto.HashObject(tx)
	if err != nil {
		return fmt.Errorf("error hashing transaction: %v", err)
	}

	entry := MempoolEntry{
		TxID:      types.HashID(txid),
		Tx:        tx,
		Fee:       fee,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error serializing mempool entry: %v", err)
	}

	key := []byte("mempool-" + string(entry.TxID))
	if err := d.db.Put(key, data, nil); err != nil {
		return fmt.Errorf("error storing mempool entry: %v", err)
	}

	if err := d.putFee(entry.TxID, fee); err != nil {
		return fmt.Errorf("error storing fee for mempool entry: %v", err)
	}

	d.mempoolMutex.Lock()
	defer d.mempoolMutex.Unlock()

	d.mempool[entry.TxID] = entry

	for _, input := range tx.Inputs {
		key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
		d.mempoolSpentInputs[key] = types.HashID(txid)
	}

	return nil
}

func (d *database) removeMempoolEntry(txid types.HashID) error {
	key := []byte("mempool-" + string(txid))
	if err := d.db.Delete(key, nil); err != nil {
		return fmt.Errorf("error removing mempool entry: %v", err)
	}
	if err := d.db.Delete([]byte("fee-"+string(txid)), nil); err != nil {
		return fmt.Errorf("error removing fee for mempool entry: %v", err)
	}

	d.mempoolMutex.Lock()
	defer d.mempoolMutex.Unlock()

	delete(d.mempool, txid)
	if entry, exists := d.mempool[txid]; exists {
		for _, input := range entry.Tx.Inputs {
			key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
			delete(d.mempoolSpentInputs, key)
		}
	}

	return nil
}

func (d *database) existsInMempool(txid types.HashID) (bool, error) {
	d.mempoolMutex.RLock()
	defer d.mempoolMutex.RUnlock()

	_, exists := d.mempool[txid]
	return exists, nil
}

func (d *database) getMempoolEntries() []MempoolEntry {
	d.mempoolMutex.RLock()
	defer d.mempoolMutex.RUnlock()

	var entries []MempoolEntry
	for _, entry := range d.mempool {
		entries = append(entries, entry)
	}
	return entries
}

func (d *database) loadMempool() ([]MempoolEntry, error) {

	var mempool []MempoolEntry

	iter := d.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		// Only look at keys that start with the mempool prefix
		if strings.HasPrefix(string(iter.Key()), "mempool-") {
			var entry MempoolEntry
			if err := json.Unmarshal(iter.Value(), &entry); err == nil {
				mempool = append(mempool, entry)
			}
		}
	}
	return mempool, iter.Error()
}

// --- Thread-Safe Pending Block Management ---

func (d *database) addPendingBlock(peer string, missingID types.HashID, block *types.Block) {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()

	d.pendingBlocks[missingID] = append(d.pendingBlocks[missingID], PendingBlock{
		Block:     block,
		Timestamp: time.Now(),
		Peer:      peer,
	})
}

func (d *database) getPendingBlocksCount() int {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()
	return len(d.pendingBlocks)
}

func (d *database) isNeededForPendingBlock(id types.HashID) bool {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()
	_, exists := d.pendingBlocks[id]
	return exists
}

// Safely fetches pending blocks and clears them from the map in one locked action
func (d *database) fetchAndClearPendingBlocks(resolvedObjID types.HashID) []PendingBlock {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()

	blocks := d.pendingBlocks[resolvedObjID]
	delete(d.pendingBlocks, resolvedObjID)

	return blocks
}

func (d *database) checkPendingBlocks() []struct {
	Peer  string
	Block *types.Block
	Txid  types.HashID
} {
	d.pendingMutex.Lock()
	defer d.pendingMutex.Unlock()
	now := time.Now()
	var expired []struct {
		Peer  string
		Block *types.Block
		Txid  types.HashID
	}

	timeout := 2 * time.Second

	for txid, blocks := range d.pendingBlocks {
		var stillPending []PendingBlock
		for _, blk := range blocks {
			if now.Sub(blk.Timestamp) > timeout {
				expired = append(expired, struct {
					Peer  string
					Block *types.Block
					Txid  types.HashID
				}{blk.Peer, blk.Block, txid})
			} else {
				stillPending = append(stillPending, blk)
			}
		}
		if len(stillPending) == 0 {
			delete(d.pendingBlocks, txid)
		} else {
			d.pendingBlocks[txid] = stillPending
		}
	}
	return expired
}
