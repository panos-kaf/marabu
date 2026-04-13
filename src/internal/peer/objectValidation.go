package peer

import (
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"math/big"
	"time"

	"fmt"
)

func (p *Peer) ValidateObject(obj Object) (T_HashID, T_Picabu, ErrorCode, error) {

	objIDstr, err := crypto.HashObject(obj)
	if err != nil {
		return T_HashID(""), ZERO_PICABU, E_INTERNAL_ERROR, fmt.Errorf("Failed to hash object for validation: %v", err)
	}
	objID := T_HashID(objIDstr)

	switch o := obj.(type) {
	case *T_Transaction:
		fee, errorCode, err := p.ValidateTransaction(o)
		if err != nil {
			return objID, ZERO_PICABU, errorCode, fmt.Errorf("Validation failed for transaction %s: %v", objID, err)
		}
		feestr := (*big.Int)(&fee).String()
		p.log(MSG_OBJECT, E_NONE, fmt.Sprintf("T_Transaction %s is valid with fee %s", objID, feestr))
		return objID, fee, E_NONE, nil

	case *T_CoinbaseTransaction:
		fee, errorCode, err := p.ValidateCoinbase(o)
		if err != nil {
			return objID, ZERO_PICABU, errorCode, fmt.Errorf("Validation failed for coinbase transaction %s: %v", objID, err)
		}
		// feestr := (*big.Int)(&fee).String()
		p.log(MSG_OBJECT, E_NONE, fmt.Sprintf("Coinbase transaction %s is valid", objID))
		return objID, fee, E_NONE, nil

	case *T_Block:
		errorCode, err := p.ValidateBlock(o)
		if err != nil {
			return objID, ZERO_PICABU, errorCode, fmt.Errorf("Validation failed for block %s: %v", objID, err)
		}
		p.log(MSG_OBJECT, E_NONE, fmt.Sprintf("T_Block %s is valid", objID))
		return objID, ZERO_PICABU, E_NONE, nil

	default:
		return objID, ZERO_PICABU, E_INTERNAL_ERROR, fmt.Errorf("Unknown object type: %T", obj)

	}
}

func (p *Peer) ValidateTransaction(tx *T_Transaction) (T_Picabu, ErrorCode, error) {
	if tx.Type != messages.OBJ_TRANSACTION {
		return ZERO_PICABU, messages.E_INTERNAL_ERROR, fmt.Errorf("Invalid object type for transaction: %s", tx.Type)
	}

	sumInputs := new(big.Int)
	sumOutputs := new(big.Int)

	type sigData struct {
		pubkey string
		sig    string
	}
	var verifyQueue []sigData

	// input/output transaction validity checks
	for i, input := range tx.Inputs {
		outpoint := input.Outpoint

		exists, err := p.Store.ExistsObject(outpoint.Txid)
		if !exists || err != nil {
			return ZERO_PICABU, E_UNKNOWN_OBJECT, fmt.Errorf("Referenced transaction %s for input %d does not exist", outpoint.Txid, i)
		}

		obj, err := p.Store.GetObject(outpoint.Txid)
		if err != nil {
			return ZERO_PICABU, E_UNKNOWN_OBJECT, fmt.Errorf("Failed to fetch referenced transaction")
		}

		var outputs []messages.T_TxOutput
		switch txObj := obj.(type) {
		case *T_Transaction:
			outputs = txObj.Outputs
		case *T_CoinbaseTransaction:
			outputs = txObj.Outputs
		default:
			return ZERO_PICABU, E_INTERNAL_ERROR, fmt.Errorf("Referenced object is of unknown type")
		}

		idx := int(*outpoint.Index)

		if idx < 0 || idx >= len(outputs) {
			return ZERO_PICABU, E_INVALID_TX_OUTPOINT, fmt.Errorf("Invalid output index")
		}

		output := outputs[idx]

		outputValue := (*big.Int)(output.Value)
		sumInputs.Add(sumInputs, outputValue)

		if input.Sig == nil {
			return ZERO_PICABU, E_INVALID_TX_SIGNATURE, fmt.Errorf("Missing signature")
		}

		// cache pubkey and sig for later verification
		verifyQueue = append(verifyQueue, sigData{
			pubkey: string(output.Pubkey),
			sig:    string(*input.Sig),
		})
	}

	// conservation check
	for _, output := range tx.Outputs {
		outputValue := (*big.Int)(output.Value)
		sumOutputs.Add(sumOutputs, outputValue)
	}

	if sumOutputs.Cmp(sumInputs) == 1 {
		return ZERO_PICABU, E_INVALID_TX_CONSERVATION, fmt.Errorf("Output value %d exceeds input value %d", sumOutputs, sumInputs)
	}

	// sig verification
	msg := messages.TxMessageForSignature(tx)

	for i, data := range verifyQueue {
		if !crypto.Verify(data.pubkey, msg, data.sig) {
			return ZERO_PICABU, E_INVALID_TX_SIGNATURE, fmt.Errorf("Invalid signature for input %d", i)
		}
	}

	fee := new(big.Int)
	fee.Sub(sumInputs, sumOutputs)
	return T_Picabu(*fee), E_NONE, nil
}

func (p *Peer) ValidateCoinbase(cb *T_CoinbaseTransaction) (T_Picabu, ErrorCode, error) {
	return ZERO_PICABU, E_NONE, nil
}

func (p *Peer) ValidateBlock(blk *T_Block) (ErrorCode, error) {

	now := time.Now().Unix()
	if int64(blk.Created) > now {
		return E_INVALID_BLOCK_TIMESTAMP, fmt.Errorf("Block timestamp %d comes from the future :o", blk.Created)
	}

	blockid, err := crypto.HashObject(blk)
	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to hash block for validation: %v", err)
	}

	isValid, err := crypto.VerifyPoW(blockid)

	// DUMMY VALIDATION - REMOVE THIS
	// isValid = true

	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to verify PoW for block %s: %v", blockid, err)
	}
	if !isValid {
		return E_INVALID_BLOCK_POW, fmt.Errorf("Invalid PoW for block %s", blockid)
	}

	isGenesis := blk.Previd == nil

	if !isGenesis {
		exists, err := p.Store.ExistsObject(*blk.Previd)
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Error checking existence of parent block %s: %v", *blk.Previd, err)
		}
		if !exists {
			p.Store.AddPendingBlock(p.addr, T_HashID(blockid), blk)
			BroadcastGetObject(T_HashID(blockid))
			return E_UNKNOWN_OBJECT, fmt.Errorf("Parent block %s not found. Asked peers for it", *blk.Previd)
		}
		prevObj, err := p.Store.GetObject(*blk.Previd)
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Failed to fetch parent block %s: %v", *blk.Previd, err)
		}
		prevBlock := prevObj.(*T_Block)

		if blk.Created <= prevBlock.Created {
			return E_INVALID_BLOCK_TIMESTAMP, fmt.Errorf("Block timestamp %d is earlier than its parent's timestamp %d", blk.Created, prevBlock.Created)
		}
	}

	utxos := make(map[UTXOKey]messages.T_TxOutput)
	var height uint64

	if isGenesis {
		p.logInfo(fmt.Sprintf("Block %s is the genesis block, so UTXO is empty", blockid))
		height = 0
	} else {
		UTXO, err := p.Store.GetUTXO(*blk.Previd)
		if err != nil {
			return E_UNKNOWN_OBJECT, fmt.Errorf("Could not find UTXO for block %s: %v", *blk.Previd, err)
		}
		utxos = UTXO.UTXOs
		height = UTXO.Height + 1
	}

	hasCoinbase := false
	var cbTx *T_CoinbaseTransaction
	var cbID T_HashID

	fees := new(big.Int)

	for index, txid := range blk.Txids {
		exists, err := p.Store.ExistsObject(txid)
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Error checking existence of transaction %s: %v", txid, err)
		}
		if !exists {
			p.Store.AddPendingBlock(p.addr, txid, blk)
			BroadcastGetObject(txid)
			return E_UNKNOWN_OBJECT, fmt.Errorf("Block references transactions we don't have. Asked peers for them")
		} else {

			tx, err := p.Store.GetObject(txid)
			if err != nil {
				return E_INTERNAL_ERROR, fmt.Errorf("Failed to fetch referenced transaction %s: %v", txid, err)
			}
			switch tx := tx.(type) {
			case *T_Transaction:

				sumInputs := new(big.Int)
				sumOutputs := new(big.Int)

				for _, input := range tx.Inputs {
					outpoint := input.Outpoint

					if hasCoinbase && outpoint.Txid == cbID {
						return E_INVALID_TX_OUTPOINT, fmt.Errorf("Cannot spend coinbase transaction in the same block")
					}

					inputIndex := int(*outpoint.Index)
					inputTx := outpoint.Txid
					spentOutput, exists := utxos[UTXOKey{Txid: inputTx, Index: inputIndex}]
					if !exists {
						return E_INVALID_TX_OUTPOINT, fmt.Errorf("Invalid input: %s (not found in UTXO set)", txid)
					}

					sumInputs.Add(sumInputs, (*big.Int)(spentOutput.Value))

					// apply the transaction by removing the spent outputs from UTXO set
					delete(utxos, UTXOKey{Txid: inputTx, Index: inputIndex})
				}

				for idx, output := range tx.Outputs {
					// add new outputs to UTXO set

					sumOutputs.Add(sumOutputs, (*big.Int)(output.Value))

					utxos[UTXOKey{Txid: txid, Index: idx}] = output

				}

				fee := new(big.Int)
				fee.Sub(sumInputs, sumOutputs)
				fees.Add(fees, fee)

			case *T_CoinbaseTransaction:
				hasCoinbase = true
				cbTx = tx
				cbIDstr, err := crypto.HashObject(cbTx)
				if err != nil {
					return E_INTERNAL_ERROR, fmt.Errorf("Failed to hash coinbase transaction for validation: %v", err)
				}
				cbID = T_HashID(cbIDstr)

				if index != 0 {
					return E_INVALID_BLOCK_COINBASE, fmt.Errorf("Only the first transaction in the block can be a coinbase, found one at index %d", index)
				}

				if uint64(*cbTx.Height) != height {
					return E_INVALID_BLOCK_COINBASE, fmt.Errorf("Coinbase transaction has height %d, while its block has height %d", *cbTx.Height, height)
				}

				utxos[UTXOKey{Txid: cbID, Index: 0}] = cbTx.Outputs[0]

			default:
				return E_INTERNAL_ERROR, fmt.Errorf("Referenced object is of unknown type")
			}
		}
	}

	var coinbaseVal *big.Int

	if hasCoinbase {
		coinbaseVal = (*big.Int)(cbTx.Outputs[0].Value)
	} else {
		coinbaseVal = new(big.Int) // zero value
	}

	totalOutput := BlockRewardBigInt()
	totalOutput.Add(totalOutput, fees)

	if coinbaseVal.Cmp(totalOutput) == 1 {
		return E_INVALID_BLOCK_COINBASE, fmt.Errorf("Coinbase transaction value %d exceeds allowed reward+fees of %d", coinbaseVal, totalOutput)
	}

	newUTXO := UTXOSet{
		UTXOs:   utxos,
		BlockID: T_HashID(blockid),
		Height:  height,
	}
	err = p.Store.PutUTXO(T_HashID(blockid), newUTXO)
	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to update UTXO set for block %s: %v", blockid, err)
	}

	hasChaintip, err := p.Store.ExistsChaintip()
	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to check for existing chaintip: %v", err)
	}

	var currentHeight uint64

	if hasChaintip {
		_, currentHeight, err = p.Store.GetChaintip()
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Failed to get current chaintip for block validation: %v", err)
		}
	}

	if !hasChaintip || height > currentHeight {
		err = p.Store.PutChaintip(T_HashID(blockid), height)
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Failed to update chaintip for block %s: %v", blockid, err)
		} else {
			p.logInfo(fmt.Sprintf("Updated chaintip to block %s at height %d (previous height: %d)", blockid, height, currentHeight))
		}
	}

	return E_NONE, nil
}
