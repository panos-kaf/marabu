package protocol

import (
	"encoding/json"
	"fmt"
	"marabu/internal/types"
	"reflect"
	"strings"
)

const (
	maxArrLen = 1000
	maxStrLen = 1000
)

var messageTypeRegistry = map[string]reflect.Type{
	string(types.MSG_HELLO):       reflect.TypeOf(Hello{}),
	string(types.MSG_ERROR):       reflect.TypeOf(Error{}),
	string(types.MSG_GETPEERS):    reflect.TypeOf(GetPeers{}),
	string(types.MSG_PEERS):       reflect.TypeOf(Peers{}),
	string(types.MSG_GETOBJECT):   reflect.TypeOf(GetObject{}),
	string(types.MSG_IHAVEOBJECT): reflect.TypeOf(IHaveObject{}),
	string(types.MSG_OBJECT):      reflect.TypeOf(Object{}),
	string(types.MSG_GETMEMPOOL):  reflect.TypeOf(GetMempool{}),
	string(types.MSG_MEMPOOL):     reflect.TypeOf(Mempool{}),
	string(types.MSG_GETCHAINTIP): reflect.TypeOf(GetChainTip{}),
	string(types.MSG_CHAINTIP):    reflect.TypeOf(ChainTip{}),
}

func UnmarshalMessage(raw string) (types.Message, error) {
	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}
	typeVal, ok := probe["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' field in message")
	}

	typ, found := messageTypeRegistry[typeVal]
	if !found {
		return nil, fmt.Errorf("unknown message type: '%s'", typeVal)
	}

	msgPtr := reflect.New(typ)

	// use DisallowUnknownFields to catch extra fields that are not defined in the schemas
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(msgPtr.Interface()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s message: %w", typeVal, err)
	}

	// Return as Message interface
	if m, ok := msgPtr.Interface().(types.Message); ok {
		return m, nil
	}
	return nil, fmt.Errorf("type %s does not implement Message interface", typeVal)
}

// Custom UnmarshalJSON for Object to handle dynamic inner object types (block, transaction, coinbase transaction)
func (o *Object) UnmarshalJSON(data []byte) error {

	type Alias Object

	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var typeProbe struct {
		Type types.ObjectType `json:"type"`
	}
	if err := json.Unmarshal(o.RawObject, &typeProbe); err != nil {
		return fmt.Errorf("failed to probe inner object type: %w", err)
	}

	// populate the Object field based on the type of the inner object
	switch typeProbe.Type {
	case types.OBJ_BLOCK:
		var b types.Block
		if err := json.Unmarshal(o.RawObject, &b); err != nil {
			return fmt.Errorf("failed to unmarshal block object: %w", err)
		}

		if b.T != types.TARGET {
			return fmt.Errorf("invalid block object: T field does not match marabu target")
		}

		o.Object = &b

	case types.OBJ_TRANSACTION:

		var cbProbe struct {
			Height *int `json:"height"`
		}

		json.Unmarshal(o.RawObject, &cbProbe)

		if cbProbe.Height != nil {
			var cb types.CoinbaseTransaction
			if err := json.Unmarshal(o.RawObject, &cb); err != nil {
				return fmt.Errorf("failed to parse coinbase transaction: %w", err)
			}
			o.Object = &cb
		} else {
			var tx types.Transaction
			if err := json.Unmarshal(o.RawObject, &tx); err != nil {
				return fmt.Errorf("failed to parse transaction: %w", err)
			}
			o.Object = &tx
		}
	default:
		return fmt.Errorf("unknown object type: %s", typeProbe.Type)
	}
	return nil
}
