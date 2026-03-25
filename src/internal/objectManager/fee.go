package objectManager

import (
	"fmt"
	"marabu/internal/messages"
	"math/big"
)

// PutFee caches the calculated fee for a transaction
func (om *ObjectManager) PutFee(id T_HashID, fee messages.T_Picabu) error {
	feeStr := (*big.Int)(&fee).String()
	key := []byte("fee-" + string(id))
	return om.db.Put(key, []byte(feeStr), nil)
}

// GetFee retrieves the cached fee
func (om *ObjectManager) GetFee(id T_HashID) (messages.T_Picabu, error) {
	key := []byte("fee-" + string(id))
	data, err := om.db.Get(key, nil)
	if err != nil {
		return messages.ZERO_PICABU, fmt.Errorf("fee not found for tx %s", id)
	}
	fee := new(big.Int)
	_, ok := fee.SetString(string(data), 10)
	if !ok {
		return messages.ZERO_PICABU, fmt.Errorf("invalid fee format for tx %s: %s", id, string(data))
	}
	return messages.T_Picabu(*fee), nil
}
