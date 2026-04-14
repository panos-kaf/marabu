package protocol

import (
	"encoding/json"
	"marabu/internal/types"
)

type (
	Hello struct {
		Type    types.MessageType `json:"type"`
		Version types.Version     `json:"version"`
		Agent   *types.BuString   `json:"agent,omitempty"`
	}

	Error struct {
		Type        types.MessageType `json:"type"`
		Name        types.ErrorCode   `json:"name"`
		Description types.BuString    `json:"description"`
	}

	GetPeers struct {
		Type types.MessageType `json:"type"`
	}

	Peers struct {
		Type  types.MessageType `json:"type"`
		Peers types.Peers       `json:"peers"`
	}

	GetObject struct {
		Type     types.MessageType `json:"type"`
		ObjectID types.HashID      `json:"objectid"`
	}

	IHaveObject struct {
		Type     types.MessageType `json:"type"`
		ObjectID types.HashID      `json:"objectid"`
	}

	Object struct {
		Type types.MessageType `json:"type"`

		// The raw, unparsed JSON of the object.
		RawObject json.RawMessage `json:"object"`

		// This field is not part of the JSON schema but is used internally to hold the deserialized object after validation
		Object types.Object `json:"-"`
	}

	GetMempool struct {
		Type types.MessageType `json:"type"`
	}

	Mempool struct {
		Type  types.MessageType `json:"type"`
		Txids types.HashIDs     `json:"txids"`
	}

	GetChainTip struct {
		Type types.MessageType `json:"type"`
	}

	ChainTip struct {
		Type  types.MessageType `json:"type"`
		Block types.HashID      `json:"block"`
	}
)

// -- message type getters --

func (h *Hello) MessageType() types.MessageType {
	return types.MSG_HELLO
}
func (e *Error) MessageType() types.MessageType {
	return types.MSG_ERROR
}
func (g *GetPeers) MessageType() types.MessageType {
	return types.MSG_GETPEERS
}
func (p *Peers) MessageType() types.MessageType {
	return types.MSG_PEERS
}
func (g *GetObject) MessageType() types.MessageType {
	return types.MSG_GETOBJECT
}
func (i *IHaveObject) MessageType() types.MessageType {
	return types.MSG_IHAVEOBJECT
}
func (o *Object) MessageType() types.MessageType {
	return types.MSG_OBJECT
}
func (g *GetMempool) MessageType() types.MessageType {
	return types.MSG_GETMEMPOOL
}
func (m *Mempool) MessageType() types.MessageType {
	return types.MSG_MEMPOOL
}
func (g *GetChainTip) MessageType() types.MessageType {
	return types.MSG_GETCHAINTIP
}
func (c *ChainTip) MessageType() types.MessageType {
	return types.MSG_CHAINTIP
}
