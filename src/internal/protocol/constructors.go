package protocol

import (
	"encoding/json"
	"marabu/internal/serialization"
	"marabu/internal/types"
)

// -- Constructor functions for messages --

func MakeHello() (string, error) {

	version := types.Version("0.10.0")
	agent := types.BuString("marabobos")

	return serialization.CanonicalizeMessage(&Hello{
		Type:      types.MSG_HELLO,
		Version: version,
		Agent:     &agent,
	})
}

func MakeError(name types.ErrorCode, description types.BuString) (string, error) {
	return serialization.CanonicalizeMessage(&Error{
		Type:        types.MSG_ERROR,
		Name:        name,
		Description: description,
	})
}

func MakeGetPeers() (string, error) {
	return serialization.CanonicalizeMessage(&GetPeers{
		Type: types.MSG_GETPEERS,
	})
}

func MakePeers(peers types.Peers) (string, error) {
	return serialization.CanonicalizeMessage(&Peers{
		Type:    types.MSG_PEERS,
		Peers: peers,
	})
}

func MakeGetObject(objectID types.HashID) (string, error) {
	return serialization.CanonicalizeMessage(&GetObject{
		Type:     types.MSG_GETOBJECT,
		ObjectID: objectID,
	})
}

func MakeIHaveObject(objectID types.HashID) (string, error) {
	return serialization.CanonicalizeMessage(&IHaveObject{
		Type:     types.MSG_IHAVEOBJECT,
		ObjectID: objectID,
	})
}

func MakeObject(obj types.Object) (string, error) {
	raw, err := serialization.Canonicalize(obj)
	if err != nil {
		return "", err
	}
	return serialization.CanonicalizeMessage(&Object{
		Type:      types.MSG_OBJECT,
		RawObject: json.RawMessage(raw),
	})
}

func MakeGetMempool() (string, error) {
	return serialization.CanonicalizeMessage(&GetMempool{
		Type: types.MSG_GETMEMPOOL,
	})
}

func MakeMempool(Txids types.HashIDs) (string, error) {
	return serialization.CanonicalizeMessage(&Mempool{
		Type:  types.MSG_MEMPOOL,
		Txids: Txids,
	})
}

func MakeGetChainTip() (string, error) {
	return serialization.CanonicalizeMessage(&GetChainTip{
		Type: types.MSG_GETCHAINTIP,
	})
}

func MakeChainTip(BlockID types.HashID) (string, error) {
	return serialization.CanonicalizeMessage(&ChainTip{
		Type:    types.MSG_CHAINTIP,
		BlockID: BlockID,
	})
}
