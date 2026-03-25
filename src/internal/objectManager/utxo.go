package objectManager

import (
	"encoding/json"
	"marabu/internal/messages"
)

type UTXOKey = messages.UTXOKey
type UTXOSet = messages.UTXOSet

type UTXORecord struct {
	Key   UTXOKey
	Value messages.T_TxOutput
}

func (om *ObjectManager) GetUTXO(blockid T_HashID) (UTXOSet, error) {

	key := []byte("utxo-" + string(blockid))
	utxoRaw, err := om.db.Get(key, nil)
	if err != nil {
		return UTXOSet{}, err
	}

	var records []UTXORecord
	err = json.Unmarshal(utxoRaw, &records)
	if err != nil {
		return UTXOSet{}, err
	}

	utxo := UTXOSet{
		BlockID: blockid,
        UTXOs: make(map[messages.UTXOKey]messages.T_TxOutput),
    }

	for _, record := range records {
		utxo.UTXOs[record.Key] = record.Value
	}

	return utxo, nil
}


func (om *ObjectManager) PutUTXO(blockid T_HashID, utxos UTXOSet) error {
    key := []byte("utxo-" + string(blockid))

    var records []UTXORecord
    for k, v := range utxos.UTXOs {
        records = append(records, UTXORecord{Key: k, Value: v})
    }

    utxoRaw, err := json.Marshal(records)
    if err != nil {
        return err
    }

    return om.db.Put(key, utxoRaw, nil)
}