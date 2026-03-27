package main

import (
	"bufio"
	"fmt"
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"net"
	"strings"
	"time"
)

type Peer struct {
	conn   net.Conn
	reader *bufio.Reader
	name   string
}

func newPeer(conn net.Conn, name string) *Peer {
	return &Peer{conn: conn, reader: bufio.NewReader(conn), name: name}
}

func (p *Peer) send(msg string) {
	fmt.Fprintf(p.conn, "%s\n", msg)
}

func waitForError(p *Peer, expected string) {
	p.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		resp, err := p.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Timeout waiting for %s\n", expected)
			return
		}
		if strings.Contains(resp, "error") {
			if strings.Contains(resp, expected) {
				fmt.Printf("Passed: Received expected %s\n", expected)
			} else {
				fmt.Printf("Failed: Expected %s, got: %s\n", expected, strings.TrimSpace(resp))
			}
			return
		}
	}
}

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func main() {
	conn, _ := net.Dial("tcp", "localhost:18018")
	defer conn.Close()
	p := newPeer(conn, "Tester")

	// Handshake
	p.send(must(messages.MakeHelloMessage()))
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n--- Starting UTXO & Block Validation Tests ---")

	// Helper pointers
	h := messages.T_BuInt(0)
	dummyTarget := messages.T_HashID("00000000abc00000000000000000000000000000000000000000000000000000")

	var prevID *messages.T_HashID = nil // Assume genesis

	// ---------------------------------------------------------
	// Test 1: Law of Conservation (Coinbase too big)
	// ---------------------------------------------------------
	fmt.Print("Test 1: Law of Conservation... ")
	tooMuch := messages.NewPicabu(50000000000001) // 1 over the limit
	cb1 := messages.T_CoinbaseTransaction{
		Type:   messages.OBJ_TRANSACTION,
		Height: &h,
		Outputs: messages.T_TxOutputs{
			{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				Value: &tooMuch}},
	}
	cb1Msg := must(messages.MakeObjectMessage(cb1))
	cb1IDstr, _ := crypto.HashObject(cb1)

	blk1 := messages.T_Block{
		Type:    messages.OBJ_BLOCK,
		T:       dummyTarget,
		Created: messages.T_BuInt(time.Now().Unix()),
		Previd:  prevID,
		Nonce:   dummyTarget,
		Txids:   messages.T_HashIDs{messages.T_HashID(cb1IDstr)},
	}

	p.send(cb1Msg)
	time.Sleep(100 * time.Millisecond)
	p.send(must(messages.MakeObjectMessage(blk1)))

	waitForError(p, "INVALID_BLOCK_COINBASE")

	// ---------------------------------------------------------
	// Test 2: Two Coinbase Transactions
	// ---------------------------------------------------------
	fmt.Print("Test 2: Two Coinbases... ")
	validAmount := messages.NewPicabu(50000000000)
	cbValid := messages.T_CoinbaseTransaction{
		Type: messages.OBJ_TRANSACTION, Height: &h,
		Outputs: messages.T_TxOutputs{{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &validAmount}},
	}
	cbValidMsg := must(messages.MakeObjectMessage(cbValid))
	cbValidIDstr, _ := crypto.HashObject(cbValid)
	cbID := messages.T_HashID(cbValidIDstr)

	blk2 := messages.T_Block{
		Type:    messages.OBJ_BLOCK,
		T:       dummyTarget,
		Previd:  prevID,
		Nonce:   dummyTarget,
		Created: messages.T_BuInt(time.Now().Unix()),
		Txids:   messages.T_HashIDs{cbID, cbID}, // Two coinbases!
	}

	p.send(cbValidMsg)
	time.Sleep(100 * time.Millisecond)
	p.send(must(messages.MakeObjectMessage(blk2)))

	waitForError(p, "INVALID_BLOCK_COINBASE")

	// ---------------------------------------------------------
	// Test 3: Coinbase spent in the same block
	// ---------------------------------------------------------
	fmt.Print("Test 3: Coinbase spent in same block... ")

	dummySig := messages.T_Signature("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	outIndex := messages.T_BuInt(0)
	illegalTx := messages.T_Transaction{
		Type: messages.OBJ_TRANSACTION,
		Inputs: messages.T_TxInputs{{
			Outpoint: messages.T_Outpoint{
				Txid:  cbID,
				Index: &outIndex},
			Sig: &dummySig}},
		Outputs: messages.T_TxOutputs{{Pubkey: "0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba", Value: &validAmount}},
	}
	illegalTxMsg := must(messages.MakeObjectMessage(illegalTx))
	illegalTxIDstr, _ := crypto.HashObject(illegalTx)

	blk3 := messages.T_Block{
		Type:    messages.OBJ_BLOCK,
		T:       dummyTarget,
		Previd:  prevID,
		Nonce:   dummyTarget,
		Created: messages.T_BuInt(time.Now().Unix()),
		Txids:   messages.T_HashIDs{cbID, messages.T_HashID(illegalTxIDstr)},
	}

	p.send(cbValidMsg)   // Ensure node has cb
	p.send(illegalTxMsg) // Ensure node has the tx
	time.Sleep(100 * time.Millisecond)
	p.send(must(messages.MakeObjectMessage(blk3)))

	waitForError(p, "INVALID_TX_OUTPOINT")
}
