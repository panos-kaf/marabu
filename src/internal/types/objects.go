package types

// -- Object sub-type definitions --

type (

	// An object is either a Tx, a coinbase Tx, or a block
	Object interface {
		ObjectType() ObjectType
	}
	ObjectType string

	Outpoint struct {
		Txid  HashID `json:"txid"`
		Index *BuInt `json:"index"`
	}

	TxInputs []TxInput
	TxInput  struct {
		Outpoint Outpoint `json:"outpoint"`

		// 64byte (128-character) hexadecimal string, handle as simple string for now...
		Sig *Signature `json:"sig"`
	}

	TxOutputs []TxOutput
	TxOutput  struct {
		Pubkey HashID  `json:"pubkey"`
		Value  *Picabu `json:"value"`
	}
)

const (
	OBJ_BLOCK       ObjectType = "block"
	OBJ_TRANSACTION ObjectType = "transaction"
)

type (
	Transaction struct {
		Type    ObjectType `json:"type"`
		Inputs  TxInputs   `json:"inputs"`
		Outputs TxOutputs  `json:"outputs"`
	}

	CoinbaseTransaction struct {
		Type    ObjectType `json:"type"`
		Height  *BuInt     `json:"height"`
		Outputs TxOutputs  `json:"outputs"`
	}

	Block struct {
		Type       ObjectType `json:"type"`
		T          HashID     `json:"T"`
		Created    BuInt      `json:"created"`
		Miner      *BuString  `json:"miner,omitempty"`
		Nonce      HashID     `json:"nonce"`
		Note       *BuString  `json:"note,omitempty"`
		Previd     *HashID    `json:"previd"` //nullable
		Studentids *BuStrings `json:"studentids,omitempty"`
		Txids      HashIDs    `json:"txids"`
	}
)

func (t Transaction) ObjectType() ObjectType {
	return OBJ_TRANSACTION
}

func (c CoinbaseTransaction) ObjectType() ObjectType {
	return OBJ_TRANSACTION
}

func (b Block) ObjectType() ObjectType {
	return OBJ_BLOCK
}

// -- object constructors --

func makeTxInput(txid HashID, index BuInt, sig Signature) TxInput {
	return TxInput{
		Outpoint: Outpoint{
			Txid:  txid,
			Index: &index,
		},
		Sig: &sig,
	}
}

func makeTxOutput(pubkey HashID, value Picabu) TxOutput {
	return TxOutput{
		Pubkey: pubkey,
		Value:  &value,
	}
}

func makeTransaction(inputs TxInputs, outputs TxOutputs) Transaction {
	return Transaction{
		Type:    OBJ_TRANSACTION,
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func makeCoinbaseTransaction(height BuInt, outputs TxOutputs) CoinbaseTransaction {
	return CoinbaseTransaction{
		Type:    OBJ_TRANSACTION,
		Height:  &height,
		Outputs: outputs,
	}
}

func makeBlock(T HashID, created BuInt, miner *BuString, nonce HashID, note *BuString, previd *HashID, studentids *BuStrings, txids HashIDs) Block {
	return Block{
		Type:       OBJ_BLOCK,
		T:          T,
		Created:    created,
		Miner:      miner,
		Nonce:      nonce,
		Note:       note,
		Previd:     previd,
		Studentids: studentids,
		Txids:      txids,
	}
}
