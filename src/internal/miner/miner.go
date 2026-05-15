package miner

import (
	"context"
	"fmt"
	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/logs"
	"marabu/internal/peer"
	"marabu/internal/types"
	"marabu/internal/utils"
	"math/big"
	"time"
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
		Agent:      manager.Agent(),
		StudentIDs: manager.StudentIDs(),
	}
}

func (m *Miner) BlockTemplate() *types.Block {

	miner := types.BuString(m.Manager.Agent())
	studentids := m.Manager.StudentIDs()

	return &types.Block{
		Type:       types.OBJ_BLOCK,
		T:          types.TARGET,
		Miner:      &miner,
		Studentids: &studentids,
	}
}

func (m *Miner) BuildBlock(note types.BuString) (*types.Block, *types.CoinbaseTransaction, error) {

	timestamp := types.BuInt(time.Now().Unix())

	block := m.BlockTemplate()

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

	return block, &coinbase, nil

}

func (m *Miner) SetNonce(block *types.Block, nonce big.Int) {
	hexNonce := fmt.Sprintf("%064x", &nonce)
	block.Nonce = types.Nonce(hexNonce)
}

func (m *Miner) Mine(ctx context.Context) error {

	target := types.TARGET_BIGINT()

	nonce := utils.GenerateRandomNonce()

	one := big.NewInt(1)

	block, coinbase, err := m.BuildBlock(types.BuString("a noteworthy note"))
	if err != nil {
		logs.GlobalError(fmt.Sprintf("Miner failed to build block template: %v", err))
		return err
	}

	hash, err := crypto.HashObjectBigInt(block)

	if err != nil {
		logs.GlobalLog(fmt.Sprintf("Error hashing block: %v", err))
		return err
	}

	for {

		select {
		case <-ctx.Done():
			logs.GlobalLog("Network found a block first. Aborting current mining job.")
			return fmt.Errorf("mining interrupted")
		default:
			// continue mining
		}
		nonce.Add(&nonce, one)
		m.SetNonce(block, nonce)

		hash, err = crypto.HashObjectBigInt(block)
		if err != nil {
			return err
		}
		if hash.Cmp(target) < 0 {
			break
		}
	}

	logs.GlobalLog(fmt.Sprintf("Found nonce for block %s!", hash))

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
		// 1. Create a fresh kill-switch for THIS specific block attempt
		ctx, cancel := context.WithCancel(context.Background())

		// 2. Hand the kill-switch to the Manager so it can pull the trigger if a network block arrives
		m.Manager.SetMiningCancel(cancel)

		// 3. Start hashing! This will block until we win, or we get killed.
		err := m.Mine(ctx)
		if err != nil {
			logs.GlobalLog(fmt.Sprintf("Mining round ended: %v", err))
		}

		// 4. We either won the block or lost the race.
		// The loop instantly restarts, creates a new context, and tries again for the NEXT height!
	}
}
