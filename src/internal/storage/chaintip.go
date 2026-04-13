package storage

import (
	"encoding/json"
	"marabu/internal/messages"
)

type Chaintip struct {
	BlockID T_HashID `json:"blockid"`
	Height  uint64   `json:"height"`
}

var chaintipKey = []byte("chaintip")

func (s *Store) ExistsChaintip() (bool, error) {
	return s.db.Has(chaintipKey, nil)
}

func (s *Store) GetChaintip() (blkid T_HashID, height uint64, err error) {

	chaintipRaw, err := s.db.Get(chaintipKey, nil)
	if err != nil {
		return messages.DUMMY_HASH, 0, err
	}

	var chaintip Chaintip
	err = json.Unmarshal(chaintipRaw, &chaintip)
	if err != nil {
		return messages.DUMMY_HASH, 0, err
	}

	return chaintip.BlockID, chaintip.Height, nil
}

func (s *Store) PutChaintip(blkid T_HashID, height uint64) error {

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
