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

func (m *Manager) ValidateObject(obj types.Object) ValidationResult {

	objIDstr, err := crypto.HashObject(obj)
	if err != nil {
		return ValidationResult{ErrorCode: types.E_INTERNAL_ERROR, Error: fmt.Errorf("Failed to hash object: %v", err)}
	}

	objID := types.HashID(objIDstr)

	var result ValidationResult

	switch o := obj.(type) {
	case *types.Transaction:
		result = m.ValidateTransaction(o)

	case *types.CoinbaseTransaction:
		result = m.ValidateCoinbase(o)

	case *types.Block:
		result = m.ValidateBlock(o)

	default:
		return ValidationResult{ErrorCode: types.E_INTERNAL_ERROR, Error: fmt.Errorf("Unknown object type: %T", obj)}
	}

	result.ObjID = objID
	return result
}

func (m *Manager) ValidateTransaction(tx *types.Transaction) ValidationResult {

	result := ValidationResult{ObjID: types.DUMMY_HASH, Fee: types.ZERO_PICABU, ErrorCode: types.E_NONE, Error: nil}

	if tx.Type != types.OBJ_TRANSACTION {
		result.ErrorCode = types.E_INTERNAL_ERROR
		result.Error = fmt.Errorf("Invalid object type for transaction: %s", tx.Type)
		return result
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

		// Check for duplicate outpoints
		key := OutpointKey{Txid: outpoint.Txid, Index: idx}
		if outpoints[key] {
			result.ErrorCode = types.E_INVALID_TX_OUTPOINT
			result.Error = fmt.Errorf("Multiple inputs have the same outpoint.")
			return result
		}
		outpoints[key] = true

		if m.IsInputSpent(key) {
			result.ErrorCode = types.E_INVALID_TX_OUTPOINT
			result.Error = fmt.Errorf("Input %d is already being spent by another transaction in the mempool", i)
			return result
		}

		obj, err := m.db.getObject(outpoint.Txid)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				result.ErrorCode = types.E_UNKNOWN_OBJECT
				
				result.MissingID = outpoint.Txid

				result.Error = fmt.Errorf("Referenced transaction %s for input %d does not exist", outpoint.Txid, i)
				return result
			}
			result.ErrorCode = types.E_UNKNOWN_OBJECT
			result.Error = fmt.Errorf("Failed to fetch referenced transaction")
			return result
		}

		var outputs []types.TxOutput
		switch txObj := obj.(type) {
		case *types.Transaction:
			outputs = txObj.Outputs
		case *types.CoinbaseTransaction:
			outputs = txObj.Outputs
		default:
			result.ErrorCode = types.E_INTERNAL_ERROR
			result.Error = fmt.Errorf("Referenced object is of unknown type")
			return result
		}

		if idx < 0 || idx >= len(outputs) {
			result.ErrorCode = types.E_INVALID_TX_OUTPOINT
			result.Error = fmt.Errorf("Invalid output index")
			return result
		}

		output := outputs[idx]
		sumInputs.Add(sumInputs, (*big.Int)(output.Value))

		if input.Sig == nil {
			result.ErrorCode = types.E_INVALID_TX_SIGNATURE
			result.Error = fmt.Errorf("Missing signature")
			return result
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
		result.ErrorCode = types.E_INVALID_TX_CONSERVATION
		result.Error = fmt.Errorf("Output value %d exceeds input value %d", sumOutputs, sumInputs)
		return result
	}

	msg := serialization.TxMessageForSignature(tx)
	for i, data := range verifyQueue {
		if !crypto.Verify(data.pubkey, msg, data.sig) {
			result.ErrorCode = types.E_INVALID_TX_SIGNATURE
			result.Error = fmt.Errorf("Invalid signature for input %d", i)
			return result
		}
	}

	fee := new(big.Int).Sub(sumInputs, sumOutputs)
	result.ErrorCode = types.E_NONE
	result.Fee = types.Picabu(*fee)
	return result
}

func (m *Manager) ValidateCoinbase(cb *types.CoinbaseTransaction) ValidationResult {
	return ValidationResult{ErrorCode: types.E_NONE, Fee: types.ZERO_PICABU}
}

func (m *Manager) ValidateBlock(blk *types.Block) ValidationResult {

	result := ValidationResult{ErrorCode: types.E_NONE}

	now := time.Now().Unix()
	if int64(blk.Created) > now {
		result.ErrorCode = types.E_INVALID_BLOCK_TIMESTAMP
		result.Error = fmt.Errorf("Block timestamp %d comes from the future :o", blk.Created)
		return result
	}

	blockid, err := crypto.HashObject(blk)
	if err != nil {
		result.ErrorCode = types.E_INTERNAL_ERROR
		result.Error = fmt.Errorf("Failed to hash block for validation: %v", err)
		return result
	}

	isValid, err := crypto.VerifyPoW(blockid)

	// REMOVE THIS AFTER TESTING
	// isValid = true // TEMPORARY: Disable PoW verification for testing

	if err != nil {
		result.ErrorCode = types.E_INTERNAL_ERROR
		result.Error = fmt.Errorf("Failed to verify PoW for block %s: %v", blockid, err)
		return result
	}
	if !isValid {
		result.ErrorCode = types.E_INVALID_BLOCK_POW
		result.Error = fmt.Errorf("Invalid PoW for block %s", blockid)
		return result
	}

	isGenesis := blk.Previd == nil
	if !isGenesis {
		prevObj, err := m.GetObject(*blk.Previd)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				result.ErrorCode = types.E_UNKNOWN_OBJECT
				result.MissingID = *blk.Previd
				result.Error = fmt.Errorf("Parent block %s not found. Asked peers for it", *blk.Previd)
				return result
			}
			result.ErrorCode = types.E_INTERNAL_ERROR
			result.Error = fmt.Errorf("Failed to fetch parent block %s: %v", *blk.Previd, err)
			return result
		}
		prevBlock := prevObj.(*types.Block)

		if blk.Created <= prevBlock.Created {
			result.ErrorCode = types.E_INVALID_BLOCK_TIMESTAMP
			result.Error = fmt.Errorf("Block timestamp %d is earlier than its parent's timestamp %d", blk.Created, prevBlock.Created)
			return result
		}
	}

	utxos := make(map[OutpointKey]types.TxOutput)
	var height uint64

	if isGenesis {
		if blockid != genesisHash {
			result.ErrorCode = types.E_INVALID_GENESIS
			result.Error = fmt.Errorf("invalid genesis block")
			return result
		}
		height = 0
	} else {
		UTXO, err := m.db.getUTXO(*blk.Previd)
		if err != nil {
			result.ErrorCode = types.E_UNKNOWN_OBJECT
			result.MissingID = *blk.Previd
			result.Error = fmt.Errorf("Could not find UTXO for block %s: %v", *blk.Previd, err)
			return result
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
				result.ErrorCode = types.E_UNKNOWN_OBJECT
				result.MissingID = txid
				result.Error = fmt.Errorf("Block references transactions we don't have")
				return result
			}

			result.ErrorCode = types.E_INTERNAL_ERROR
			result.Error = fmt.Errorf("Failed to fetch referenced transaction: %v", err)
			return result
		}
		switch t := tx.(type) {
		case *types.Transaction:
			sumIn := new(big.Int)
			sumOut := new(big.Int)

			for _, input := range t.Inputs {
				if hasCoinbase && input.Outpoint.Txid == cbID {
					result.ErrorCode = types.E_INVALID_TX_OUTPOINT
					result.Error = fmt.Errorf("Cannot spend coinbase transaction in the same block")
					return result
				}
				key := OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
				spentOutput, exists := utxos[key]
				if !exists {
					result.ErrorCode = types.E_INVALID_TX_OUTPOINT
					result.Error = fmt.Errorf("Invalid input: %s (not found in UTXO set)", txid)
					return result
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
				result.ErrorCode = types.E_INVALID_BLOCK_COINBASE
				result.Error = fmt.Errorf("Only the first transaction in the block can be a coinbase, found one at index %d", index)
				return result
			}
			if uint64(*t.Height) != height {
				result.ErrorCode = types.E_INVALID_BLOCK_COINBASE
				result.Error = fmt.Errorf("Coinbase transaction has height %d, while its block has height %d", *t.Height, height)
				return result
			}
			utxos[OutpointKey{Txid: cbID, Index: 0}] = t.Outputs[0]
		default:
			result.ErrorCode = types.E_INTERNAL_ERROR
			result.Error = fmt.Errorf("Referenced object is of unknown type")
			return result
		}
	}

	coinbaseVal := new(big.Int)
	if hasCoinbase {
		coinbaseVal = (*big.Int)(cbTx.Outputs[0].Value)
	}

	totalOutput := new(big.Int).Add(types.BlockRewardBigInt(), fees)
	if coinbaseVal.Cmp(totalOutput) == 1 {
		result.ErrorCode = types.E_INVALID_BLOCK_COINBASE
		result.Error = fmt.Errorf("Coinbase transaction value %d exceeds allowed reward+fees of %d", coinbaseVal, totalOutput)
		return result
	}

	// Build the proposed UTXO set, but DO NOT save it
	result.NewUTXOSet = &UTXOSet{UTXOs: utxos, BlockID: types.HashID(blockid), Height: height}

	var curHeight uint64
	hasTip := true

	_, curHeight, err = m.GetChaintip()

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// we don't have a tip yet (e.g. genesis)
			hasTip = false
		} else {
			// database error
			result.ErrorCode = types.E_INTERNAL_ERROR
			result.Error = fmt.Errorf("Failed to get current chaintip for block validation: %v", err)
			return result
		}
	}

	// Calculate if this is a new tip
	isNewTip := !hasTip || height > curHeight

	// Return the proposed state instead of saving it
	result.ErrorCode = types.E_NONE
	result.NewHeight = height
	result.IsNewTip = isNewTip
	result.Error = nil
	return result
}
