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

/* -------------------------
   PEER STRUCT
--------------------------*/

type Peer struct {
	conn   net.Conn
	reader *bufio.Reader
	name   string
}

func newPeer(conn net.Conn, name string) *Peer {
	return &Peer{
		conn:   conn,
		reader: bufio.NewReader(conn),
		name:   name,
	}
}

func (p *Peer) send(msg string) {
	fmt.Printf("[%s] --> %s\n", p.name, msg)
	fmt.Fprintf(p.conn, "%s\n", msg)
}

func (p *Peer) receive() (string, error) {
	p.conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	resp, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	resp = strings.TrimSpace(resp)
	fmt.Printf("[%s] <-- %s\n", p.name, resp)
	return resp, nil
}

/* -------------------------
   HELPERS
--------------------------*/

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func waitFor(p *Peer, expected string) {
	for {
		resp, err := p.receive()
		if err != nil {
			fmt.Printf("❌ Timeout waiting for %s\n", expected)
			return
		}
		if strings.Contains(resp, expected) {
			fmt.Println("✅ OK:", expected)
			return
		}
	}
}

func waitForAny(p *Peer, expected ...string) string {
	for {
		resp, err := p.receive()
		if err != nil {
			fmt.Println("❌ Timeout waiting for messages")
			return ""
		}
		for _, e := range expected {
			if strings.Contains(resp, e) {
				fmt.Println("✅ OK:", e)
				return resp
			}
		}
	}
}

/* -------------------------
   HANDSHAKE
--------------------------*/

func handshake(p *Peer) {

	agent := types.BuString("object-test-client")

	p.send(must(protocol.MakeHello(&agent)))

	seenHello := false
	seenGetPeers := false

	for !(seenHello && seenGetPeers) {
		resp, err := p.receive()
		if err != nil {
			fmt.Println("❌ Handshake failed:", err)
			return
		}

		if strings.Contains(resp, "hello") {
			seenHello = true
			fmt.Println("✅ OK: hello")
		}
		if strings.Contains(resp, "getpeers") {
			seenGetPeers = true
			fmt.Println("✅ OK: getpeers")
		}
	}
}

/* -------------------------
   MSG_OBJECT EXCHANGE TESTS
--------------------------*/

// 1a
func testSelfObjectRetrieval(p *Peer, objID types.HashID, objMsg string) {
	fmt.Println("\n[Test 1a] Self object retrieval")

	p.send(objMsg)
	p.send(must(protocol.MakeGetObject(objID)))

	waitFor(p, "object")
}

// 1d
func testIHaveFlow(p *Peer, objID types.HashID) {
	fmt.Println("\n[Test 1d] ihaveobject -> getobject")

	p.send(must(protocol.MakeIHaveObject(objID)))
	waitFor(p, "getobject")
}

// 1b + 1c
func testGossip(p1, p2 *Peer, objID types.HashID, objMsg string) {
	fmt.Println("\n[Test 1b/1c] Gossip between peers")

	p1.send(objMsg)

	// p2 should get ihaveobject
	waitFor(p2, "ihaveobject")

	// request object
	p2.send(must(protocol.MakeGetObject(objID)))
	waitFor(p2, "object")
}

/* -------------------------
   VALIDATION TESTS
--------------------------*/

func expectError(p *Peer, expected string) {
	resp := waitForAny(p, "error")
	if !strings.Contains(resp, expected) {
		fmt.Printf("❌ Expected error %s, got %s\n", expected, resp)
	} else {
		fmt.Println("✅ OK:", expected)
	}
}

// 2a(i)
func testUnknownObject(p *Peer) {
	fmt.Println("\n[Test 2a(i)] UNKNOWN_OBJECT")

	idx := types.BuInt(0)
	tx := types.Transaction{
		Type: types.OBJ_TRANSACTION,
		Inputs: []types.TxInput{
			{
				Outpoint: types.Outpoint{
					Txid:  types.HashID("abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
					Index: &idx,
				},
			},
		},
	}

	p.send(must(protocol.MakeObject(tx)))
	expectError(p, "UNKNOWN_OBJECT")
}

// {"object":{
// 	"height":0,
// 	"outputs":[
// 		{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4",
// 		"value":50000000000}
// 		],
// 	"type":"transaction"
// 	},
// "type":"object"}

// 2a(ii)
func testInvalidSignature(p *Peer, coinbaseID types.HashID) {
	fmt.Println("\n[Test 2a(ii)] INVALID_TX_SIGNATURE")

	v := types.NewPicabu(10)
	idx := types.BuInt(0)

	tx := types.Transaction{
		Type: types.OBJ_TRANSACTION,
		Inputs: []types.TxInput{
			{
				Outpoint: types.Outpoint{Txid: coinbaseID, Index: &idx},
				Sig:      nil,
			},
		},
		Outputs: []types.TxOutput{
			{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &v},
		},
	}

	p.send(must(protocol.MakeObject(tx)))
	expectError(p, "INVALID_TX_SIGNATURE")
}

// 2a(iii)
func testInvalidOutpoint(p *Peer, coinbaseID types.HashID) {
	fmt.Println("\n[Test 2a(iii)] INVALID_TX_OUTPOINT")

	v := types.NewPicabu(10)
	idx := types.BuInt(999)

	tx := types.Transaction{
		Type: types.OBJ_TRANSACTION,
		Inputs: []types.TxInput{
			{
				Outpoint: types.Outpoint{Txid: coinbaseID, Index: &idx},
			},
		},
		Outputs: []types.TxOutput{
			{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &v},
		},
	}

	p.send(must(protocol.MakeObject(tx)))
	expectError(p, "INVALID_TX_OUTPOINT")
}

// 2a(iv)
func testConservation(p *Peer, coinbaseID types.HashID, sig types.Signature) {
	fmt.Println("\n[Test 2a(iv)] INVALID_TX_CONSERVATION")

	v := types.NewPicabu(50000000001)
	idx := types.BuInt(0)

	tx := types.Transaction{
		Type: types.OBJ_TRANSACTION,
		Inputs: []types.TxInput{
			{
				Outpoint: types.Outpoint{Txid: coinbaseID, Index: &idx},
				Sig:      &sig,
			},
		},
		Outputs: []types.TxOutput{
			{Pubkey: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", Value: &v},
		},
	}

	p.send(must(protocol.MakeObject(tx)))
	expectError(p, "INVALID_TX_CONSERVATION")
}

// 2a(v)
func testInvalidFormat(p *Peer) {
	fmt.Println("\n[Test 2a(v)] INVALID_FORMAT")

	p.send("{invalid json}")
	expectError(p, "INVALID_FORMAT")
}

/* -------------------------
   MAIN
--------------------------*/

func main() {
	addr := "localhost:18018"

	conn1, _ := net.Dial("tcp", addr)
	conn2, _ := net.Dial("tcp", addr)

	p1 := newPeer(conn1, "G1")
	p2 := newPeer(conn2, "G2")

	defer conn1.Close()
	defer conn2.Close()

	fmt.Println("Connected two graders")

	handshake(p1)
	handshake(p2)

	/* -------------------------
	   Coinbase
	--------------------------*/
	h := types.BuInt(0)
	v := types.NewPicabu(50000000000)

	coinbase := types.CoinbaseTransaction{
		Type:   types.OBJ_TRANSACTION,
		Height: &h,
		Outputs: []types.TxOutput{
			{Pubkey: "39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4", Value: &v},
		},
	}

	coinbaseIDstr, _ := crypto.HashObject(coinbase)
	coinbaseID := types.HashID(coinbaseIDstr)
	coinbaseMsg := must(protocol.MakeObject(coinbase))

	p1.send(coinbaseMsg)

	// wait until node processes it (consume anything)
	waitForAny(p1, "ihaveobject", "object", "ok")

	/* -------------------------
	   Object exchange
	--------------------------*/
	testSelfObjectRetrieval(p1, coinbaseID, coinbaseMsg)
	dummyHash := types.HashID("0000000000000000000000000000000000000000000000000000000000001234")
	testIHaveFlow(p1, dummyHash)
	testGossip(p1, p2, coinbaseID, coinbaseMsg)

	/* -------------------------
	   Validation
	--------------------------*/
	sig := types.Signature("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	testUnknownObject(p1)
	testInvalidSignature(p1, coinbaseID)
	testInvalidOutpoint(p1, coinbaseID)
	testConservation(p1, coinbaseID, sig)
	testInvalidFormat(p1)

	fmt.Println("\nAll tests executed")
}
