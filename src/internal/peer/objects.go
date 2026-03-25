package peer

import (
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"math/big"

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
		feestr := (*big.Int)(&fee).String()
		p.log(MSG_OBJECT, E_NONE, fmt.Sprintf("Coinbase transaction %s is valid with fee %s", objID, feestr))
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

	// 1. Create a quick struct to hold the data we need for crypto later
	type sigData struct {
		pubkey string
		sig    string
	}
	var verifyQueue []sigData

	// input/output transaction validity checks
	for i, input := range tx.Inputs {
		outpoint := input.Outpoint

		exists, err := p.objectManager.Exists(outpoint.Txid)
		if !exists || err != nil {
			return ZERO_PICABU, E_UNKNOWN_OBJECT, fmt.Errorf("Referenced transaction %s for input %d does not exist", outpoint.Txid, i)
		}

		obj, err := p.objectManager.Get(outpoint.Txid)
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

	blockid, err := crypto.HashObject(blk)
	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to hash block for validation: %v", err)
	}

	isValid, err := crypto.VerifyPoW(blockid)
	if err != nil {
		return E_INTERNAL_ERROR, fmt.Errorf("Failed to verify PoW for block %s: %v", blockid, err)
	}
	if !isValid {
		return E_INVALID_BLOCK_POW, fmt.Errorf("Invalid PoW for block %s", blockid)
	}

	hasCoinbase := false
	var cbTx *T_CoinbaseTransaction
	var cbID T_HashID

	fees := new(big.Int)

	for index, txid := range blk.Txids {
		exists, err := p.objectManager.Exists(txid)
		if err != nil {
			return E_INTERNAL_ERROR, fmt.Errorf("Error checking existence of transaction %s: %v", txid, err)
		}
		if !exists {
			p.objectManager.AddPendingBlock(p.addr, txid, blk)
			BroadcastGetObject(txid)
			return E_UNKNOWN_OBJECT, fmt.Errorf("Block references transactions we don't have. Asked peers for them")
		} else {

			tx, err := p.objectManager.Get(txid)
			if err != nil {
				return E_INTERNAL_ERROR, fmt.Errorf("Failed to fetch referenced transaction %s: %v", txid, err)
			}
			switch tx := tx.(type) {
			case *T_Transaction:
				fee, err := p.objectManager.GetFee(txid)
				if err != nil {
					p.logInfo(fmt.Sprintf("Fee cache miss for transaction %s: %v", txid, err))

					recalculatedFee, errorCode, err := p.ValidateTransaction(tx)
					if err != nil {
						return errorCode, fmt.Errorf("Failed to validate transaction %s while processing block: %v", txid, err)
					}
					fee = recalculatedFee

					err = p.objectManager.PutFee(txid, fee)
					if err != nil {
						p.err(MSG_OBJECT, E_NONE, fmt.Sprintf("Failed to cache fee for transaction %s: %v", txid, err))
					}
				}
				fees.Add(fees, (*big.Int)(&fee))

				for _, input := range tx.Inputs {
					outpoint := input.Outpoint

					if hasCoinbase && outpoint.Txid == cbID {
						return E_INVALID_TX_OUTPOINT, fmt.Errorf("Cannot spend coinbase transaction in the same block")
					}

				}
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
			default:
				return E_INTERNAL_ERROR, fmt.Errorf("Referenced object is of unknown type")
			}

			// at this point, the TX is structurally valid
			// so we will try to add it to the UTXO set.
			// appendToUTXO(txid)

		}
	}

	coinbaseVal := (*big.Int)(cbTx.Outputs[0].Value)

	totalOutput := BlockRewardBigInt()
	totalOutput.Add(totalOutput, fees)

	if coinbaseVal.Cmp(totalOutput) == 1 {
		return E_INVALID_BLOCK_COINBASE, fmt.Errorf("Coinbase transaction value %d exceeds allowed reward+fees of %d", coinbaseVal, totalOutput)
	}

	return E_NONE, nil
}
