package core

import (
	"errors"
	"fmt"
	"marabu/internal/crypto"
	"marabu/internal/serialization"
	"marabu/internal/types"
	"math/big"
	"time"
)

const genesisHash = "00000000522473196b73bc619a8b18472c4cb4c6caf785a13fa32aaae7222ff6"

type ValidationResult struct {
	ObjID     types.HashID
	Fee       types.Picabu
	ErrorCode types.ErrorCode
	Error     error
	MissingID types.HashID

	// Proposed State
	NewUTXOSet *UTXOSet
	IsNewTip   bool
	NewHeight  uint64
}

func (m *Manager) ValidateObject(obj types.Object, peerAddr string) ValidationResult {
	result := ValidationResult{
		ObjID:     types.DUMMY_HASH,
		Fee:       types.ZERO_PICABU,
		ErrorCode: types.E_NONE,
		Error:     nil,
		MissingID: types.DUMMY_HASH,
	}
	objIDstr, err := crypto.HashObject(obj)
	if err != nil {
		result.ErrorCode = types.E_INTERNAL_ERROR
		result.Error = fmt.Errorf("Failed to hash object for validation: %v", err)
		return result
	}
	objID := types.HashID(objIDstr)

	switch o := obj.(type) {
	case *types.Transaction:
		fee, errorCode, err := m.ValidateTransaction(o)
		result.ObjID = objID
		result.Fee = fee
		result.ErrorCode = errorCode
		result.Error = err
		return result

	case *types.CoinbaseTransaction:
		fee, errorCode, err := m.ValidateCoinbase(o)
		result.ObjID = objID
		result.Fee = fee
		result.ErrorCode = errorCode
		result.Error = err
		return result

	case *types.Block:
		errorCode, missingID, newUTXO, isNewTip, newHeight, err := m.ValidateBlock(o, peerAddr)

		result.ObjID = objID
		result.ErrorCode = errorCode
		result.MissingID = missingID
		result.Error = err

		result.NewUTXOSet = newUTXO
		result.IsNewTip = isNewTip
		result.NewHeight = newHeight

		return result

	default:
		result.ErrorCode = types.E_INTERNAL_ERROR
		result.Error = fmt.Errorf("Unknown object type: %T", obj)
		return result
	}
}

func (m *Manager) ValidateTransaction(tx *types.Transaction) (types.Picabu, types.ErrorCode, error) {
	if tx.Type != types.OBJ_TRANSACTION {
		return types.ZERO_PICABU, types.E_INTERNAL_ERROR, fmt.Errorf("Invalid object type for transaction: %s", tx.Type)
	}

	sumInputs := new(big.Int)
	sumOutputs := new(big.Int)

	type sigData struct {
		pubkey string
		sig    string
	}
	var verifyQueue []sigData

	outpoints := make(map[OutpointKey]bool)

	for i, input := range tx.Inputs {
		outpoint := input.Outpoint
		idx := int(*outpoint.Index)

		key := OutpointKey{Txid: outpoint.Txid, Index: idx}
		if m.IsInputSpent(key) {
			return types.ZERO_PICABU, types.E_INVALID_TX_OUTPOINT, fmt.Errorf("Input %d is already being spent by another transaction in the mempool", i)
		}

		obj, err := m.db.getObject(outpoint.Txid)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return types.ZERO_PICABU, types.E_UNKNOWN_OBJECT, fmt.Errorf("Referenced transaction %s for input %d does not exist", outpoint.Txid, i)
			}
			return types.ZERO_PICABU, types.E_UNKNOWN_OBJECT, fmt.Errorf("Failed to fetch referenced transaction")
		}

		var outputs []types.TxOutput
		switch txObj := obj.(type) {
		case *types.Transaction:
			outputs = txObj.Outputs
		case *types.CoinbaseTransaction:
			outputs = txObj.Outputs
		default:
			return types.ZERO_PICABU, types.E_INTERNAL_ERROR, fmt.Errorf("Referenced object is of unknown type")
		}

		if idx < 0 || idx >= len(outputs) {
			return types.ZERO_PICABU, types.E_INVALID_TX_OUTPOINT, fmt.Errorf("Invalid output index")
		}

		// Check for duplicate outpoints
		if outpoints[key] {
			return types.ZERO_PICABU, types.E_INVALID_TX_OUTPOINT, fmt.Errorf("Multiple inputs have the same outpoint.")
		}
		outpoints[key] = true

		output := outputs[idx]
		sumInputs.Add(sumInputs, (*big.Int)(output.Value))

		if input.Sig == nil {
			return types.ZERO_PICABU, types.E_INVALID_TX_SIGNATURE, fmt.Errorf("Missing signature")
		}

		verifyQueue = append(verifyQueue, sigData{
			pubkey: string(output.Pubkey),
			sig:    string(*input.Sig),
		})
	}

	for _, output := range tx.Outputs {
		sumOutputs.Add(sumOutputs, (*big.Int)(output.Value))
	}

	if sumOutputs.Cmp(sumInputs) == 1 {
		return types.ZERO_PICABU, types.E_INVALID_TX_CONSERVATION, fmt.Errorf("Output value %d exceeds input value %d", sumOutputs, sumInputs)
	}

	msg := serialization.TxMessageForSignature(tx)
	for i, data := range verifyQueue {
		if !crypto.Verify(data.pubkey, msg, data.sig) {
			return types.ZERO_PICABU, types.E_INVALID_TX_SIGNATURE, fmt.Errorf("Invalid signature for input %d", i)
		}
	}

	fee := new(big.Int).Sub(sumInputs, sumOutputs)
	return types.Picabu(*fee), types.E_NONE, nil
}

func (m *Manager) ValidateCoinbase(cb *types.CoinbaseTransaction) (types.Picabu, types.ErrorCode, error) {
	return types.ZERO_PICABU, types.E_NONE, nil
}

func (m *Manager) ValidateBlock(blk *types.Block, peerAddr string) (types.ErrorCode, types.HashID, *UTXOSet, bool, uint64, error) {
	now := time.Now().Unix()
	if int64(blk.Created) > now {
		return types.E_INVALID_BLOCK_TIMESTAMP, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Block timestamp %d comes from the future :o", blk.Created)
	}

	blockid, err := crypto.HashObject(blk)
	if err != nil {
		return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Failed to hash block for validation: %v", err)
	}

	isValid, err := crypto.VerifyPoW(blockid)

	// REMOVE THIS AFTER TESTING
	// isValid = true // TEMPORARY: Disable PoW verification for testing

	if err != nil {
		return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Failed to verify PoW for block %s: %v", blockid, err)
	}
	if !isValid {
		return types.E_INVALID_BLOCK_POW, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Invalid PoW for block %s", blockid)
	}

	isGenesis := blk.Previd == nil
	if !isGenesis {
		prevObj, err := m.GetObject(*blk.Previd)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				m.db.addPendingBlock(peerAddr, *blk.Previd, blk)
				return types.E_UNKNOWN_OBJECT, *blk.Previd, nil, false, 0, fmt.Errorf("Parent block %s not found. Asked peers for it", *blk.Previd)
			}
			return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Failed to fetch parent block %s: %v", *blk.Previd, err)
		}
		prevBlock := prevObj.(*types.Block)

		if blk.Created <= prevBlock.Created {
			return types.E_INVALID_BLOCK_TIMESTAMP, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Block timestamp %d is earlier than its parent's timestamp %d", blk.Created, prevBlock.Created)
		}
	}

	utxos := make(map[OutpointKey]types.TxOutput)
	var height uint64

	if isGenesis {
		if blockid != genesisHash {
			return types.E_INVALID_GENESIS, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("invalid genesis block")
		}
		height = 0
	} else {
		UTXO, err := m.db.getUTXO(*blk.Previd)
		if err != nil {
			return types.E_UNKNOWN_OBJECT, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Could not find UTXO for block %s: %v", *blk.Previd, err)
		}
		utxos = UTXO.UTXOs
		height = UTXO.Height + 1
	}

	hasCoinbase := false
	var cbTx *types.CoinbaseTransaction
	var cbID types.HashID
	fees := new(big.Int)

	for index, txid := range blk.Txids {
		tx, err := m.GetObject(txid)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				m.db.addPendingBlock(peerAddr, txid, blk)
				return types.E_UNKNOWN_OBJECT, txid, nil, false, 0, fmt.Errorf("Block references transactions we don't have")
			}

			return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Failed to fetch referenced transaction: %v", err)
		}
		switch t := tx.(type) {
		case *types.Transaction:
			sumIn := new(big.Int)
			sumOut := new(big.Int)

			for _, input := range t.Inputs {
				if hasCoinbase && input.Outpoint.Txid == cbID {
					return types.E_INVALID_TX_OUTPOINT, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Cannot spend coinbase transaction in the same block")
				}
				key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
				spentOutput, exists := utxos[key]
				if !exists {
					return types.E_INVALID_TX_OUTPOINT, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Invalid input: %s (not found in UTXO set)", txid)
				}
				sumIn.Add(sumIn, (*big.Int)(spentOutput.Value))
				delete(utxos, key)
			}
			for idx, output := range t.Outputs {
				sumOut.Add(sumOut, (*big.Int)(output.Value))
				utxos[OutpointKey{Txid: txid, Index: idx}] = output
			}
			fees.Add(fees, new(big.Int).Sub(sumIn, sumOut))

		case *types.CoinbaseTransaction:
			hasCoinbase = true
			cbTx = t
			id, _ := crypto.HashObject(cbTx)
			cbID = types.HashID(id)

			if index != 0 {
				return types.E_INVALID_BLOCK_COINBASE, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Only the first transaction in the block can be a coinbase, found one at index %d", index)
			}
			if uint64(*t.Height) != height {
				return types.E_INVALID_BLOCK_COINBASE, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Coinbase transaction has height %d, while its block has height %d", *t.Height, height)
			}
			utxos[OutpointKey{Txid: cbID, Index: 0}] = t.Outputs[0]
		default:
			return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Referenced object is of unknown type")
		}
	}

	coinbaseVal := new(big.Int)
	if hasCoinbase {
		coinbaseVal = (*big.Int)(cbTx.Outputs[0].Value)
	}

	totalOutput := new(big.Int).Add(types.BlockRewardBigInt(), fees)
	if coinbaseVal.Cmp(totalOutput) == 1 {
		return types.E_INVALID_BLOCK_COINBASE, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Coinbase transaction value %d exceeds allowed reward+fees of %d", coinbaseVal, totalOutput)
	}

	// Build the proposed UTXO set, but DO NOT save it
	newSet := UTXOSet{UTXOs: utxos, BlockID: types.HashID(blockid), Height: height}

	var curHeight uint64
	hasTip := true

	_, curHeight, err = m.GetChaintip()

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// we don't have a tip yet (e.g. genesis)
			hasTip = false
		} else {
			// database error
			return types.E_INTERNAL_ERROR, types.DUMMY_HASH, nil, false, 0, fmt.Errorf("Failed to get current chaintip for block validation: %v", err)
		}
	}

	// Calculate if this is a new tip
	isNewTip := !hasTip || height > curHeight

	// Return the proposed state instead of saving it
	return types.E_NONE, types.DUMMY_HASH, &newSet, isNewTip, height, nil
}
