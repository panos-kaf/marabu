package serialization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"marabu/internal/types"
	"sort"
)

// Canonicalize takes an arbitrary Go value,
// marshals it to JSON,
// and then re-marshals it in a canonical form
// with sorted keys and consistent formatting.
func canonicalizeJSON(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := bytes.NewBufferString("{")
		for i, k := range keys {
			if i > 0 {
				buf.WriteString(",")
			}
			keyBytes, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf.Write(keyBytes)
			buf.WriteString(":")
			valueBytes, err := canonicalizeJSON(val[k])
			if err != nil {
				return nil, err
			}
			buf.Write(valueBytes)

		}
		buf.WriteString("}")
		return buf.Bytes(), nil

	case []interface{}:
		buf := bytes.NewBufferString("[")
		for i, elem := range val {
			if i > 0 {
				buf.WriteString(",")
			}
			elemBytes, err := canonicalizeJSON(elem)
			if err != nil {
				return nil, err
			}
			buf.Write(elemBytes)
		}
		buf.WriteString("]")
		return buf.Bytes(), nil

	default:
		return json.Marshal(val)
	}

}

func Canonicalize(v interface{}) (string, error) {
	var obj interface{}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return "", err
	}
	canon, err := canonicalizeJSON(obj)
	if err != nil {
		return "", err
	}
	return string(canon), nil
}

// wraps the canonicalization process for the Message interface,
// readying the message for message contrsuction to be sent over the network.
func CanonicalizeMessage(msg types.Message) (string, error) {

	canon, err := Canonicalize(msg)
	if err != nil {
		return "", fmt.Errorf("Error parsing message: %w", err)
	}
	return canon + "\n", nil
}

// Creates a canonical transaction with nil signatures for signing/verification
func TxMessageForSignature(tx *types.Transaction) []byte {
	// Create a copy of the transaction with empty signatures for signing/verification
	txCopy := *tx
	txCopy.Inputs = make([]types.TxInput, len(tx.Inputs))
	copy(txCopy.Inputs, tx.Inputs)
	for i := range txCopy.Inputs {
		txCopy.Inputs[i].Sig = nil
	}
	// Canonicalize the transaction copy to get the message bytes
	msgBytes, _ := (Canonicalize(txCopy))
	return []byte(msgBytes)
}
