package messages

import (
	"encoding/json"
)

type (
	Message interface {
		MessageType() MessageType

		// schema validation now happens during unmarshaling
		// Validate() (error, ErrorCode)
	}

	MessageType string
	ErrorCode   string
	T_Version   string
	T_Peer      string
	T_Peers     []T_Peer
	T_HashID    string // 32byte (64-character) hex string
	T_HashIDs   []T_HashID
	T_Signature string // 64byte (128-character) hex string

	T_BuInt     int
	T_BuString  string
	T_BuInts    []T_BuInt
	T_BuStrings []T_BuString
)

const (
	MSG_NONE        MessageType = ""
	MSG_HELLO       MessageType = "hello"
	MSG_ERROR       MessageType = "error"
	MSG_GETPEERS    MessageType = "getpeers"
	MSG_PEERS       MessageType = "peers"
	MSG_GETOBJECT   MessageType = "getobject"
	MSG_IHAVEOBJECT MessageType = "ihaveobject"
	MSG_OBJECT      MessageType = "object"
	MSG_GETMEMPOOL  MessageType = "getmempool"
	MSG_MEMPOOL     MessageType = "mempool"
	MSG_GETCHAINTIP MessageType = "getchaintip"
	MSG_CHAINTIP    MessageType = "chaintip"

	E_NONE                    ErrorCode = ""
	E_INTERNAL_ERROR          ErrorCode = "INTERNAL_ERROR"
	E_INVALID_FORMAT          ErrorCode = "INVALID_FORMAT"
	E_UNKNOWN_OBJECT          ErrorCode = "UNKNOWN_OBJECT"
	E_UNFINDABLE_OBJECT       ErrorCode = "UNFINDABLE_OBJECT"
	E_INVALID_HANDSHAKE       ErrorCode = "INVALID_HANDSHAKE"
	E_INVALID_TX_OUTPOINT     ErrorCode = "INVALID_TX_OUTPOINT"
	E_INVALID_TX_SIGNATURE    ErrorCode = "INVALID_TX_SIGNATURE"
	E_INVALID_TX_CONSERVATION ErrorCode = "INVALID_TX_CONSERVATION"
	E_INVALID_BLOCK_COINBASE  ErrorCode = "INVALID_BLOCK_COINBASE"
	E_INVALID_BLOCK_TIMESTAMP ErrorCode = "INVALID_BLOCK_TIMESTAMP"
	E_INVALID_BLOCK_POW       ErrorCode = "INVALID_BLOCK_POW"
	E_INVALID_GENESIS         ErrorCode = "INVALID_GENESIS"

	// Peer sanitization fail value
	PEER_INVALID = ""
)

type (
	HelloMessage struct {
		Type      MessageType `json:"type"`
		T_Version T_Version   `json:"version"`
		Agent     *T_BuString `json:"agent,omitempty"`
	}

	ErrorMessage struct {
		Type        MessageType `json:"type"`
		Name        ErrorCode   `json:"name"`
		Description T_BuString  `json:"description"`
	}

	GetPeersMessage struct {
		Type MessageType `json:"type"`
	}

	PeersMessage struct {
		Type    MessageType `json:"type"`
		T_Peers T_Peers     `json:"peers"`
	}

	GetObjectMessage struct {
		Type     MessageType `json:"type"`
		ObjectID T_HashID    `json:"objectid"`
	}

	IHaveObjectMessage struct {
		Type     MessageType `json:"type"`
		ObjectID T_HashID    `json:"objectid"`
	}

	ObjectMessage struct {
		Type MessageType `json:"type"`

		// The raw, unparsed JSON of the object.
		RawObject json.RawMessage `json:"object"`

		// This field is not part of the JSON schema but is used internally to hold the deserialized object after validation
		Object Object `json:"-"`
	}

	GetMempoolMessage struct {
		Type MessageType `json:"type"`
	}

	MempoolMessage struct {
		Type  MessageType `json:"type"`
		Txids T_HashIDs   `json:"txids"`
	}

	GetChainTipMessage struct {
		Type MessageType `json:"type"`
	}

	ChainTipMessage struct {
		Type    MessageType `json:"type"`
		T_Block T_HashID    `json:"block"`
	}
)

// -- message type getters --

func (h *HelloMessage) MessageType() MessageType {
	return MSG_HELLO
}
func (e *ErrorMessage) MessageType() MessageType {
	return MSG_ERROR
}
func (g *GetPeersMessage) MessageType() MessageType {
	return MSG_GETPEERS
}
func (p *PeersMessage) MessageType() MessageType {
	return MSG_PEERS
}
func (g *GetObjectMessage) MessageType() MessageType {
	return MSG_GETOBJECT
}
func (i *IHaveObjectMessage) MessageType() MessageType {
	return MSG_IHAVEOBJECT
}
func (o *ObjectMessage) MessageType() MessageType {
	return MSG_OBJECT
}
func (g *GetMempoolMessage) MessageType() MessageType {
	return MSG_GETMEMPOOL
}
func (m *MempoolMessage) MessageType() MessageType {
	return MSG_MEMPOOL
}
func (g *GetChainTipMessage) MessageType() MessageType {
	return MSG_GETCHAINTIP
}
func (c *ChainTipMessage) MessageType() MessageType {
	return MSG_CHAINTIP
}
