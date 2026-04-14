package main

import (
	"bufio"
	"fmt"
	"marabu/internal/crypto"
	"marabu/internal/protocol"
	"marabu/internal/types"
	"net"
	"strings"
	"time"
)

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func send(conn net.Conn, msg string) {
	fmt.Fprintf(conn, "%s\n", msg)
}

// NEW: Smart Reader. It only returns if it finds the exact error we are looking for!
func waitForSpecificError(reader *bufio.Reader, timeout time.Duration, conn net.Conn, targetError string, alternateError string) string {
	conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		resp, err := reader.ReadString('\n')
		if err != nil {
			// Check if the node actively kicked us out or crashed!
			if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
				return fmt.Sprintf("CONNECTION_DROPPED (%v)", err)
			}
			return "" // Legitimate timeout
		}

		resp = strings.TrimSpace(resp)

		// Check if this line contains our target error
		if strings.Contains(resp, `"error"`) {
			if strings.Contains(resp, targetError) || (alternateError != "" && strings.Contains(resp, alternateError)) {
				return resp
			}
			// If it's an error from a previous test, we just ignore it and keep looping!
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:18018")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// --- Handshake ---
	send(conn, `{"type":"hello","version":"0.10.0","agent":"MasterTester"}`)
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n========================================")
	fmt.Println("  MARABU NODE MASTER TEST SUITE")
	fmt.Println("========================================")

	h := types.BuInt(0)
	dummyTarget := types.HashID("00000000abc00000000000000000000000000000000000000000000000000000")
	var prevID *types.HashID = nil // Proper Genesis PrevID

	// ---------------------------------------------------------
	// Test 1: Genesis Block Ingestion
	// ---------------------------------------------------------
	fmt.Print("[Test 1] Ingesting Empty Genesis Block... ")
	genesisMsg := `{"object":{"T":"00000000abc00000000000000000000000000000000000000000000000000000","created":1771159355,"miner":"Marabu","nonce":"00dd82159556175752d9ba7349df67bddd237b59183747383f7b720e85c32347","note":"Master Test Genesis","previd":null,"txids":[],"type":"block"},"type":"object"}`
	send(conn, genesisMsg)

	// We expect NO error here, so we wait 1 second for "INVALID_FORMAT". If it times out (""), it passed!
	resp := waitForSpecificError(reader, 1*time.Second, conn, "INVALID_FORMAT", "")
	if strings.Contains(resp, "CONNECTION_DROPPED") {
		fmt.Println("❌ FAILED: Node closed the connection!")
		return
	} else if resp != "" {
		fmt.Println("❌ FAILED: Node rejected Genesis block.")
	} else {
		fmt.Println("✅ PASSED")
	}

	// ---------------------------------------------------------
	// Test 2: Law of Conservation (Coinbase too big)
	// ---------------------------------------------------------
	fmt.Print("[Test 2] Law of Conservation (50T+1)... ")
	tooMuch := types.NewPicabu(50000000000001) // 50 Trillion + 1
	cb1 := types.CoinbaseTransaction{
		Type:   types.OBJ_TRANSACTION,
		Height: &h,
		Outputs: types.TxOutputs{
			{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &tooMuch}},
	}
	cb1Msg := must(protocol.MakeObject(cb1))
	cb1IDstr, _ := crypto.HashObject(cb1)

	blk1 := types.Block{
		Type:    types.OBJ_BLOCK,
		T:       dummyTarget,
		Nonce:   dummyTarget,
		Created: types.BuInt(time.Now().Unix()),
		Previd:  prevID,
		Txids:   types.HashIDs{types.HashID(cb1IDstr)},
	}

	send(conn, cb1Msg)
	time.Sleep(100 * time.Millisecond)
	send(conn, must(protocol.MakeObject(blk1)))

	resp = waitForSpecificError(reader, 2*time.Second, conn, "INVALID_BLOCK_COINBASE", "")
	if strings.Contains(resp, "INVALID_BLOCK_COINBASE") {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Printf("❌ FAILED: Expected INVALID_BLOCK_COINBASE, got: %s\n", resp)
	}

	// ---------------------------------------------------------
	// Test 3: Two Coinbase Transactions
	// ---------------------------------------------------------
	fmt.Print("[Test 3] Two Coinbases in one block... ")
	validAmount := types.NewPicabu(50000000000000) // Exactly 50 Trillion
	cbValid := types.CoinbaseTransaction{
		Type: types.OBJ_TRANSACTION, Height: &h,
		Outputs: types.TxOutputs{{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &validAmount}},
	}
	cbValidMsg := must(protocol.MakeObject(cbValid))
	cbValidIDstr, _ := crypto.HashObject(cbValid)
	cbID := types.HashID(cbValidIDstr)

	blk2 := types.Block{
		Type: types.OBJ_BLOCK, T: dummyTarget, Nonce: dummyTarget, Previd: prevID, Created: types.BuInt(time.Now().Unix()),
		Txids: types.HashIDs{cbID, cbID}, // Two coinbases!
	}

	send(conn, cbValidMsg)
	time.Sleep(100 * time.Millisecond)
	send(conn, must(protocol.MakeObject(blk2)))

	// It might throw INVALID_BLOCK_COINBASE or INVALID_TX_OUTPOINT if it processes it weirdly.
	resp = waitForSpecificError(reader, 2*time.Second, conn, "INVALID_BLOCK_COINBASE", "")
	if strings.Contains(resp, "INVALID_BLOCK_COINBASE") {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Printf("❌ FAILED: Expected INVALID_BLOCK_COINBASE, got: %s\n", resp)
	}

	// ---------------------------------------------------------
	// Test 4: Coinbase spent in the same block
	// ---------------------------------------------------------
	fmt.Print("[Test 4] Coinbase spent in same block... ")
	outIndex := types.BuInt(0)
	dummySig := types.Signature("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // 64-character signature

	illegalTx := types.Transaction{
		Type:    types.OBJ_TRANSACTION,
		Inputs:  types.TxInputs{{Outpoint: types.Outpoint{Txid: cbID, Index: &outIndex}, Sig: &dummySig}},
		Outputs: types.TxOutputs{{Pubkey: "0987654321fedcba0987654321fedcba0987654321fedcba0987654321fedcba", Value: &validAmount}},
	}
	illegalTxMsg := must(protocol.MakeObject(illegalTx))
	illegalTxIDstr, _ := crypto.HashObject(illegalTx)

	blk3 := types.Block{
		Type: types.OBJ_BLOCK, T: dummyTarget, Nonce: dummyTarget, Previd: prevID, Created: types.BuInt(time.Now().Unix()),
		Txids: types.HashIDs{cbID, types.HashID(illegalTxIDstr)},
	}

	send(conn, cbValidMsg)
	send(conn, illegalTxMsg)
	time.Sleep(100 * time.Millisecond)
	send(conn, must(protocol.MakeObject(blk3)))

	resp = waitForSpecificError(reader, 2*time.Second, conn, "INVALID_TX_OUTPOINT", "INVALID_TX_SIGNATURE")
	if strings.Contains(resp, "INVALID_TX_OUTPOINT") || strings.Contains(resp, "INVALID_TX_SIGNATURE") {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Printf("❌ FAILED: Expected outpoint/sig error, got: %s\n", resp)
	}

	// ---------------------------------------------------------
	// Test 5: The Ghost Transaction (UNFINDABLE_OBJECT)
	// ---------------------------------------------------------
	fmt.Print("[Test 5] Unfindable Object Timeout (Waiting 5-8s)... ")
	fakeTxID := types.HashID("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	blkFakeTx := types.Block{
		Type:    types.OBJ_BLOCK,
		T:       dummyTarget,
		Nonce:   dummyTarget,
		Created: types.BuInt(time.Now().Unix()),
		Previd:  prevID,
		Txids:   types.HashIDs{fakeTxID},
	}

	send(conn, must(protocol.MakeObject(blkFakeTx)))

	// Wait for the 5-second timeout
	resp = waitForSpecificError(reader, 8*time.Second, conn, "UNFINDABLE_OBJECT", "unfindable")

	if strings.Contains(resp, "UNFINDABLE_OBJECT") || strings.Contains(resp, "unfindable") {
		fmt.Println("✅ PASSED")
	} else if resp == "" {
		fmt.Println("❌ FAILED: Node timed out without sending UNFINDABLE_OBJECT.")
	} else {
		fmt.Printf("❌ FAILED: Unexpected response: %s\n", resp)
	}

	fmt.Println("========================================")
}
