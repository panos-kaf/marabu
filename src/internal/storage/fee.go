package storage

import (
	"fmt"
	"marabu/internal/types"
	"math/big"
)

// PutFee caches the calculated fee for a transaction
func (s *Store) PutFee(id types.HashID, fee types.Picabu) error {
	feeStr := (*big.Int)(&fee).String()
	key := []byte("fee-" + string(id))
	return s.db.Put(key, []byte(feeStr), nil)
}

// GetFee retrieves the cached fee
func (s *Store) GetFee(id types.HashID) (types.Picabu, error) {
	key := []byte("fee-" + string(id))
	data, err := s.db.Get(key, nil)
	if err != nil {
		return types.ZERO_PICABU, fmt.Errorf("fee not found for tx %s", id)
	}
	fee := new(big.Int)
	_, ok := fee.SetString(string(data), 10)
	if !ok {
		return types.ZERO_PICABU, fmt.Errorf("invalid fee format for tx %s: %s", id, string(data))
	}
	return types.Picabu(*fee), nil
}
