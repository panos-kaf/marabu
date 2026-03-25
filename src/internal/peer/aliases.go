package peer

import (
	"marabu/internal/messages"
	"math/big"
)

const (
	sent = true
	recv = false
)

var (
	MSG_NONE        = messages.MSG_NONE
	MSG_HELLO       = messages.MSG_HELLO
	MSG_ERROR       = messages.MSG_ERROR
	MSG_GETPEERS    = messages.MSG_GETPEERS
	MSG_PEERS       = messages.MSG_PEERS
	MSG_GETOBJECT   = messages.MSG_GETOBJECT
	MSG_IHAVEOBJECT = messages.MSG_IHAVEOBJECT
	MSG_OBJECT      = messages.MSG_OBJECT
	MSG_GETMEMPOOL  = messages.MSG_GETMEMPOOL
	MSG_MEMPOOL     = messages.MSG_MEMPOOL
	MSG_GETCHAINTIP = messages.MSG_GETCHAINTIP
	MSG_CHAINTIP    = messages.MSG_CHAINTIP

	E_NONE                    = messages.E_NONE
	E_INTERNAL_ERROR          = messages.E_INTERNAL_ERROR
	E_INVALID_FORMAT          = messages.E_INVALID_FORMAT
	E_UNKNOWN_OBJECT          = messages.E_UNKNOWN_OBJECT
	E_UNFINDABLE_OBJECT       = messages.E_UNFINDABLE_OBJECT
	E_INVALID_HANDSHAKE       = messages.E_INVALID_HANDSHAKE
	E_INVALID_TX_OUTPOINT     = messages.E_INVALID_TX_OUTPOINT
	E_INVALID_TX_SIGNATURE    = messages.E_INVALID_TX_SIGNATURE
	E_INVALID_TX_CONSERVATION = messages.E_INVALID_TX_CONSERVATION
	E_INVALID_BLOCK_COINBASE  = messages.E_INVALID_BLOCK_COINBASE
	E_INVALID_BLOCK_TIMESTAMP = messages.E_INVALID_BLOCK_TIMESTAMP
	E_INVALID_BLOCK_POW       = messages.E_INVALID_BLOCK_POW
	E_INVALID_GENESIS         = messages.E_INVALID_GENESIS

	OBJ_BLOCK       = messages.OBJ_BLOCK
	OBJ_TRANSACTION = messages.OBJ_TRANSACTION

	PEER_INVALID = messages.PEER_INVALID
	TARGET       = messages.TARGET
	DUMMY_HASH   = messages.DUMMY_HASH
	ZERO_PICABU  = messages.ZERO_PICABU
)

func BlockReward() messages.T_Picabu {
	return messages.BlockReward()
}

func BlockRewardBigInt() *big.Int {
	return messages.BlockRewardBigInt()
}

type (

	// Interfaces
	Message = messages.Message
	Object  = messages.Object

	// Message Field Types
	MessageType = messages.MessageType
	ErrorCode   = messages.ErrorCode

	T_Version   = messages.T_Version
	T_Peer      = messages.T_Peer
	T_Peers     = messages.T_Peers
	T_HashID    = messages.T_HashID
	T_HashIDs   = messages.T_HashIDs
	T_Signature = messages.T_Signature
	T_Picabu    = messages.T_Picabu

	T_BuInt     = messages.T_BuInt
	T_BuString  = messages.T_BuString
	T_BuInts    = messages.T_BuInts
	T_BuStrings = messages.T_BuStrings

	// Object-specific Field Types
	T_Transaction         = messages.T_Transaction
	T_CoinbaseTransaction = messages.T_CoinbaseTransaction
	T_Block               = messages.T_Block

	// Transaction-specific Field Types
	T_TxOutput = messages.T_TxOutput
	T_TxInput  = messages.T_TxInput
	T_Outpoint = messages.T_Outpoint

	// Message Schemas
	HelloMessage       = messages.HelloMessage
	ErrorMessage       = messages.ErrorMessage
	GetPeersMessage    = messages.GetPeersMessage
	PeersMessage       = messages.PeersMessage
	GetObjectMessage   = messages.GetObjectMessage
	IHaveObjectMessage = messages.IHaveObjectMessage
	ObjectMessage      = messages.ObjectMessage
	GetMempoolMessage  = messages.GetMempoolMessage
	MempoolMessage     = messages.MempoolMessage
	GetChainTipMessage = messages.GetChainTipMessage
	ChainTipMessage    = messages.ChainTipMessage
)
