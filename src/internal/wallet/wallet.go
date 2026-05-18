package wallet

import (
	"crypto/ed25519"
	"fmt"

	"marabu/internal/core"
	"marabu/internal/crypto"
	"marabu/internal/peer"
	"marabu/internal/serialization"
	"marabu/internal/types"
)

type Wallet struct {
	manager *core.Manager
	privKey ed25519.PrivateKey
	pubKey  types.HashID

	aliases       map[string]types.HashID
	reverseLookup map[types.HashID]string
	anonCounter   int
}

func New(m *core.Manager, priv ed25519.PrivateKey, pub types.HashID) *Wallet {
	w := &Wallet{
		manager:       m,
		privKey:       priv,
		pubKey:        pub,
		aliases:       make(map[string]types.HashID),
		reverseLookup: make(map[types.HashID]string),
		anonCounter:   1,
	}

	w.AddAlias("me", pub)
	return w
}

func (w *Wallet) AddAlias(name string, pubkey types.HashID) {
	w.aliases[name] = pubkey
	w.reverseLookup[pubkey] = name
}

func (w *Wallet) GetAliases() map[string]types.HashID {
	return w.aliases
}

func (w *Wallet) ResolveAddress(input string) types.HashID {
	// If the input matches a saved alias, return the saved pubkey
	if pubkey, exists := w.aliases[input]; exists {
		return pubkey
	}
	// Otherwise, assume they just pasted a raw pubkey and return it as-is
	return types.HashID(input)
}

// AutoAliasUTXOs scans the current blockchain state and assigns names to unknown keys
func (w *Wallet) AutoAliasUTXOs() int {
	tip, _, err := w.manager.GetChaintip()
	if err != nil {
		return 0
	}

	activeUTXOs, err := w.manager.GetUTXO(tip)
	if err != nil {
		return 0
	}

	newAliases := 0

	for _, output := range activeUTXOs.UTXOs {
		pubkey := output.Pubkey

		// If this pubkey isn't in our reverse lookup map yet, it's new!
		if _, exists := w.reverseLookup[pubkey]; !exists {
			// Generate a programmatic name
			newName := fmt.Sprintf("Anon-%d", w.anonCounter)

			// Save it
			w.AddAlias(newName, pubkey)

			w.anonCounter++
			newAliases++
		}
	}

	return newAliases
}

// GetBalance asks the core Manager for the UTXO set and calculates the total balance
func (w *Wallet) GetBalance() (types.Picabu, map[core.OutpointKey]types.TxOutput) {
	tip, _, err := w.manager.GetChaintip()
	if err != nil {
		return types.ZERO_PICABU, nil
	}

	activeUTXOs, err := w.manager.GetUTXO(tip)
	if err != nil {
		return types.ZERO_PICABU, nil
	}

	total := types.NewPicabu(0)
	myUTXOs := make(map[core.OutpointKey]types.TxOutput)

	for key, output := range activeUTXOs.UTXOs {
		if output.Pubkey == w.pubKey {
			if !w.manager.IsInputSpent(key) {
				myUTXOs[key] = output
				total = total.Add(*output.Value)
			}
		}
	}

	return total, myUTXOs
}

// SendPicabus builds the transaction, signs it, commits it to the mempool, and broadcasts it
func (w *Wallet) SendPicabus(targetPubkey types.HashID, amount types.Picabu, fee types.Picabu) (*types.Transaction, error) {

	needed := amount.Add(fee)

	balance, myUTXOs := w.GetBalance()

	if balance.Cmp(needed) < 0 {
		return nil, fmt.Errorf("insufficient funds. Have %s, need %s", balance.String(), needed.String())
	}

	var inputs types.TxInputs
	collected := types.NewPicabu(0)

	for key, output := range myUTXOs {
		index := types.BuInt(key.Index)
		input := types.MakeTxInput(key.Txid, index, types.Signature(""))
		inputs = append(inputs, input)

		collected = collected.Add(*output.Value)
		if collected.Cmp(needed) >= 0 {
			break
		}
	}

	var outputs types.TxOutputs
	outputs = append(outputs, types.MakeTxOutput(targetPubkey, amount))

	change := collected.Sub(needed)
	if change.Cmp(types.ZERO_PICABU) > 0 {
		outputs = append(outputs, types.MakeTxOutput(w.pubKey, change))
	}

	tx := types.MakeTransaction(inputs, outputs)
	msgBytes := serialization.TxMessageForSignature(&tx)

	for i := range tx.Inputs {
		sigStr, err := crypto.Sign(msgBytes, w.privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign input: %v", err)
		}
		sig := types.Signature(sigStr)
		tx.Inputs[i].Sig = &sig
	}

	result := w.manager.ValidateObject(&tx)
	if result.Error != nil {
		return nil, fmt.Errorf("transaction failed validation: %v", result.Error)
	}

	gossip, errCode, err := w.manager.CommitObject(&tx, result)
	if err != nil || errCode != types.E_NONE {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	if gossip {
		peer.BroadcastObject(&tx)
	}

	return &tx, nil
}
