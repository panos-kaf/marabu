package storage

import (
	"encoding/json"
	"marabu/internal/messages"
)

type UTXOKey = messages.UTXOKey
type UTXOSet = messages.UTXOSet

type UTXORecord struct {
	Key   UTXOKey             `json:"key"`
	Value messages.T_TxOutput `json:"value"`
}

type UTXOStorageData struct {
	BlockID T_HashID     `json:"blockid"`
	Height  uint64       `json:"height"`
	Records []UTXORecord `json:"records"`
}

func (s *Store) GetUTXO(blockid T_HashID) (UTXOSet, error) {
	key := []byte("utxo-" + string(blockid))
	utxoRaw, err := s.db.Get(key, nil)
	if err != nil {
		return UTXOSet{}, err
	}

	var data UTXOStorageData
	err = json.Unmarshal(utxoRaw, &data)
	if err != nil {
		return UTXOSet{}, err
	}

	utxo := UTXOSet{
		BlockID: data.BlockID,
		Height:  data.Height,
		UTXOs:   make(map[UTXOKey]messages.T_TxOutput),
	}

	for _, record := range data.Records {
		utxo.UTXOs[record.Key] = record.Value
	}

	return utxo, nil
}

func (s *Store) PutUTXO(blockid T_HashID, utxos UTXOSet) error {
	key := []byte("utxo-" + string(blockid))

	var records []UTXORecord
	for k, v := range utxos.UTXOs {
		records = append(records, UTXORecord{Key: k, Value: v})
	}

	// Marshal the complete data wrapper
	data := UTXOStorageData{
		BlockID: utxos.BlockID,
		Height:  utxos.Height,
		Records: records,
	}

	utxoRaw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.db.Put(key, utxoRaw, nil)
}
