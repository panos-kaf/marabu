package messages

// -- Object sub-type definitions --

type (

	// An object is either a Tx, a coinbase Tx, or a block
	Object interface {
		ObjectType() ObjectType
		// Validate() (error, ErrorCode)
	}
	ObjectType string

	T_Outpoint struct {
		Txid  T_HashID `json:"txid"`
		Index *T_BuInt `json:"index"`
	}

	T_TxInputs []T_TxInput
	T_TxInput  struct {
		Outpoint T_Outpoint `json:"outpoint"`

		// 64byte (128-character) hexadecimal string, handle as simple string for now...
		Sig *T_Signature `json:"sig"`
	}

	T_TxOutputs []T_TxOutput
	T_TxOutput  struct {
		Pubkey T_HashID  `json:"pubkey"`
		Value  *T_Picabu `json:"value"`
	}
)

const (
	OBJ_BLOCK       ObjectType = "block"
	OBJ_TRANSACTION ObjectType = "transaction"
)

type (
	T_Transaction struct {
		Type    ObjectType  `json:"type"`
		Inputs  T_TxInputs  `json:"inputs"`
		Outputs T_TxOutputs `json:"outputs"`
	}

	T_CoinbaseTransaction struct {
		Type    ObjectType  `json:"type"`
		Height  *T_BuInt    `json:"height"`
		Outputs T_TxOutputs `json:"outputs"`
	}

	T_Block struct {
		Type       ObjectType   `json:"type"`
		T          T_HashID     `json:"T"`
		Created    T_BuInt      `json:"created"`
		Miner      *T_BuString  `json:"miner,omitempty"`
		Nonce      T_HashID     `json:"nonce"`
		Note       *T_BuString  `json:"note,omitempty"`
		Previd     *T_HashID    `json:"previd"` //nullable
		Studentids *T_BuStrings `json:"studentids,omitempty"`
		Txids      T_HashIDs    `json:"txids"`
	}
)

func (t T_Transaction) ObjectType() ObjectType {
	return OBJ_TRANSACTION
}

func (c T_CoinbaseTransaction) ObjectType() ObjectType {
	return OBJ_TRANSACTION
}

func (b T_Block) ObjectType() ObjectType {
	return OBJ_BLOCK
}

// func (t T_Transaction) Validate() (error, ErrorCode) {
// 	return nil, E_NONE
// }

// func (c T_CoinbaseTransaction) Validate() (error, ErrorCode) {
// 	return nil, E_NONE
// }

// func (b T_Block) Validate() (error, ErrorCode) {
// 	return nil, E_NONE
// }

// -- object constructors --

func makeTxInput(txid T_HashID, index T_BuInt, sig T_Signature) T_TxInput {
	return T_TxInput{
		Outpoint: T_Outpoint{
			Txid:  txid,
			Index: &index,
		},
		Sig: &sig,
	}
}

func makeTxOutput(pubkey T_HashID, value T_Picabu) T_TxOutput {
	return T_TxOutput{
		Pubkey: pubkey,
		Value:  &value,
	}
}

func makeTransaction(inputs T_TxInputs, outputs T_TxOutputs) T_Transaction {
	return T_Transaction{
		Type:    OBJ_TRANSACTION,
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func makeCoinbaseTransaction(height T_BuInt, outputs T_TxOutputs) T_CoinbaseTransaction {
	return T_CoinbaseTransaction{
		Type:    OBJ_TRANSACTION,
		Height:  &height,
		Outputs: outputs,
	}
}

func makeBlock(T T_HashID, created T_BuInt, miner *T_BuString, nonce T_HashID, note *T_BuString, previd *T_HashID, studentids *T_BuStrings, txids T_HashIDs) T_Block {
	return T_Block{
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
