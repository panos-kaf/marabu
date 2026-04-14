package storage

import (
	"encoding/json"
	"marabu/internal/types"
)

type Chaintip struct {
	BlockID types.HashID `json:"blockid"`
	Height  uint64       `json:"height"`
}

var chaintipKey = []byte("chaintip")

func (s *Store) ExistsChaintip() (bool, error) {
	return s.db.Has(chaintipKey, nil)
}

func (s *Store) GetChaintip() (blkid types.HashID, height uint64, err error) {

	chaintipRaw, err := s.db.Get(chaintipKey, nil)
	if err != nil {
		return types.DUMMY_HASH, 0, err
	}

	var chaintip Chaintip
	err = json.Unmarshal(chaintipRaw, &chaintip)
	if err != nil {
		return types.DUMMY_HASH, 0, err
	}

	return chaintip.BlockID, chaintip.Height, nil
}

func (s *Store) PutChaintip(blkid types.HashID, height uint64) error {

	chaintip := Chaintip{
		BlockID: blkid,
		Height:  height,
	}

	chaintipRaw, err := json.Marshal(chaintip)
	if err != nil {
		return err
	}

	return s.db.Put(chaintipKey, chaintipRaw, nil)
}
