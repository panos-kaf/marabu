package miner

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/logs"
	"marabu/internal/peer"
	"marabu/internal/serialization"
	"marabu/internal/types"
	"marabu/internal/utils"
	"math/big"
	"time"

	"golang.org/x/crypto/blake2s"
)

type Miner struct {
	Manager    *core.Manager
	Pubkey     types.HashID
	Agent      types.BuString
	StudentIDs types.BuStrings
}

func NewMiner(manager *core.Manager, pubkey types.HashID) *Miner {
	return &Miner{
		Manager:    manager,
		Pubkey:     pubkey,
		Agent:      manager.Config().AgentName,
		StudentIDs: manager.Config().StudentIDs,
	}
}

func (m *Miner) BuildBlock(note types.BuString) (*types.Block, *types.CoinbaseTransaction, error) {

	timestamp := types.BuInt(time.Now().Unix())

	var block types.Block

	block.Type = types.OBJ_BLOCK
	block.T = types.TARGET

	if m.Agent != "" {
		block.Miner = &m.Agent
	}

	if len(m.StudentIDs) > 0 {
		block.Studentids = &m.StudentIDs
	}

	block.Created = timestamp

	isGen := false
	previd, height, err := m.Manager.GetChaintip()
	if err == core.ErrNotFound {
		isGen = true
	} else if err != nil {
		return nil, nil, err
	}

	if !isGen {
		block.Previd = &previd
		height += 1
	}

	mempool := m.Manager.GetMempoolEntries()

	var txids types.HashIDs

	// Insert placeholder to replace with the coinbase after calculating reward
	txids = append(txids, types.DUMMY_HASH)

	reward := types.BlockReward()

	for _, entry := range mempool {
		txids = append(txids, entry.TxID)
		reward = reward.Add(entry.Fee)
	}

	output := types.MakeTxOutput(m.Pubkey, reward)

	coinbase := types.MakeCoinbaseTransaction(types.BuInt(height), types.TxOutputs{output})

	res := m.Manager.ValidateObject(&coinbase)
	if res.Error != nil {
		return nil, nil, fmt.Errorf("miner generated invalid coinbase: %v", res.Error)
	}
	coinbaseHashStr, err := crypto.HashObject(coinbase)
	if err != nil {
		return nil, nil, fmt.Errorf("miner failed to commit coinbase: %v", err)
	}

	txids[0] = types.HashID(coinbaseHashStr)
	block.Txids = txids

	block.Note = &note

	return &block, &coinbase, nil

}

func (m *Miner) SetNonce(block *types.Block, nonce big.Int) {
	hexNonce := fmt.Sprintf("%064x", &nonce)
	block.Nonce = types.Nonce(hexNonce)
}

func (m *Miner) mineWorker(
	ctx context.Context,
	templateBytes []byte,
	nonceOffset int,
	target *big.Int,
	resultCh chan<- string,
	hashTracker chan<- uint64,
) {
	// Give worker its own private memory
	workerBytes := make([]byte, len(templateBytes))
	copy(workerBytes, templateBytes)

	hasher, _ := blake2s.New256(nil)
	nonce := utils.GenerateRandomNonce()
	one := big.NewInt(1)
	var localHashes uint64 = 0

	rawNonceBytes := make([]byte, 32)

	hashResult := make([]byte, 32)

	hashInt := new(big.Int)

	for {
		// Check if we should die (because another core or the network found a block)
		select {
		case <-ctx.Done():
			hashTracker <- localHashes // Report leftover hashes before dying
			return
		default:
		}

		nonce.Add(&nonce, one)

		// Fill our pre-allocated array with the binary math (padded to exactly 32 bytes)
		nonce.FillBytes(rawNonceBytes)

		// Directly encode those raw bytes as a 64-character hex string
		// straight into our worker's template array
		hex.Encode(workerBytes[nonceOffset:nonceOffset+64], rawNonceBytes)

		hasher.Reset()
		hasher.Write(workerBytes)

		hashResult = hasher.Sum(hashResult[:0])

		hashInt.SetBytes(hashResult)

		localHashes++

		// Batch report telemetry to avoid channel bottleneck
		if localHashes%1000 == 0 {
			hashTracker <- 1000
			localHashes = 0
		}

		// found a valid nonce
		if hashInt.Cmp(target) < 0 {
			hashTracker <- localHashes
			resultCh <- fmt.Sprintf("%064x", &nonce) // Send the winning nonce back to the orchestrator
			return
		}
	}
}

func (m *Miner) Mine(ctx context.Context) error {

	m.Manager.SetMiningState(true)
	defer m.Manager.SetMiningState(false) // Automatically reset to idle when this function exits

	target := types.TARGET_BIGINT()

	block, coinbase, err := m.BuildBlock(types.BuString("a noteworthy note"))
	if err != nil {
		logs.GlobalError(fmt.Sprintf("Miner failed to build block template: %v", err))
		return err
	}

	dummyNonce := types.Nonce(types.DUMMY_HASH)
	block.Nonce = dummyNonce

	// Canonicalize the block
	templateStr, err := serialization.Canonicalize(block)
	if err != nil {
		return fmt.Errorf("failed to canonicalize template: %v", err)
	}
	templateBytes := []byte(templateStr)

	// Find the exact memory offset where our dummy nonce starts
	nonceOffset := bytes.Index(templateBytes, []byte(dummyNonce))
	if nonceOffset == -1 {
		return fmt.Errorf("failed to find nonce offset in canonicalized block")
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers() // Ensure workers are killed no matter how this function exits

	numCores := m.Manager.GetMiningCores()
	logs.GlobalLog(fmt.Sprintf("Starting mining with %d cores...", numCores))

	resultCh := make(chan string)
	hashTracker := make(chan uint64, numCores*2) // Buffered channel for telemetry

	for range numCores {
		go m.mineWorker(workerCtx, templateBytes, nonceOffset, target, resultCh, hashTracker)
	}

	// Start a background telemetry aggregator
	go func() {
		for h := range hashTracker {
			m.Manager.AddMiningHashes(h)
		}
	}()

	var winningNonce string

	select {
	case <-ctx.Done():
		// The network found a block first (triggered by m.Manager.minerCancel)
		logs.GlobalLog("Network found a block first. Aborting current mining job.")
		return fmt.Errorf("mining interrupted")

	case winningNonce = <-resultCh:
		// One of our cores found the solution
		cancelWorkers() // Instantly kill all other cores
		block.Nonce = types.Nonce(winningNonce)
	}

	finalHash, _ := crypto.HashObject(block)
	logs.GlobalLog(fmt.Sprintf("Found nonce for block %s!", finalHash))

	m.Manager.StoreObject(coinbase)

	res := m.Manager.ValidateObject(block)
	if res.Error != nil {
		logs.GlobalError(fmt.Sprintf("Miner generated an invalid block %v", res.Error))
		return res.Error
	}

	gossip, errCode, err := m.Manager.CommitObject(block, res)
	if err != nil || errCode != types.E_NONE {
		logs.GlobalError(fmt.Sprintf("Miner failed to commit block: %v", err))
		return err
	}

	if gossip {
		peer.BroadcastObject(block)
	}

	return nil
}

func (m *Miner) StartMining() {
	logs.GlobalLog("Miner orchestrator started...")

	for {
		ctx, cancel := context.WithCancel(context.Background())

		m.Manager.SetMiningCancel(cancel)

		err := m.Mine(ctx)
		if err != nil {
			logs.GlobalLog(fmt.Sprintf("Mining round ended: %v", err))
		}

	}
}
