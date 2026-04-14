package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

/* -------------------------
   PEER STRUCT & HELPERS
   (Kept exactly as you wrote them)
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

func expectError(p *Peer, expected string) {
	resp := waitForAny(p, "error")
	if !strings.Contains(resp, expected) {
		fmt.Printf("❌ Expected error %s, got %s\n", expected, resp)
	} else {
		fmt.Println("✅ OK:", expected)
	}
}

func handshake(p *Peer) {
	p.send(`{"agent":"Grader Test","type":"hello","version":"0.10.0"}`)
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
		}
		if strings.Contains(resp, "getpeers") {
			seenGetPeers = true
		}
	}
	fmt.Println("✅ Handshake complete for", p.name)
}

/* -------------------------
   EXACT GRADER TEST RECREATION
--------------------------*/

func main() {
	addr := "localhost:18018"

	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	p1 := newPeer(conn1, "Grader 1")
	p2 := newPeer(conn2, "Grader 2")

	defer conn1.Close()
	defer conn2.Close()

	fmt.Println("--- Starting Grader Recreation ---")
	handshake(p1)
	handshake(p2)

	// Test 1: Valid Coinbase Gossip
	fmt.Println("\n[Test 1] Valid Coinbase Gossip")
	coinbaseStr := `{"object":{"height":0,"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":50000000000}],"type":"transaction"},"type":"object"}`
	coinbaseTxID := "ba46a995539f59b1095953117c3899c7396925f2da0d0cf0e0b572b506a14e7a"
	p1.send(coinbaseStr)

	// Grader 2 must receive ihaveobject
	waitFor(p2, fmt.Sprintf(`"objectid":"%s"`, coinbaseTxID))
	p2.send(fmt.Sprintf(`{"objectid":"%s","type":"getobject"}`, coinbaseTxID))
	waitFor(p2, `"type":"object"`) // Should receive the full object

	// Test 2: Valid Spending TX Gossip
	fmt.Println("\n[Test 2] Valid Spending TX Gossip")
	validTxStr := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ba46a995539f59b1095953117c3899c7396925f2da0d0cf0e0b572b506a14e7a"},"sig":"5ccb106250d4dbd0921b3fdea79b8d7865f163f1e1000184e42c7cf19f7f6deec3f78c982992613a2bc5435e454740721a850172ed884f64b4620f7822b57e04"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":10}],"type":"transaction"},"type":"object"}`
	validTxID := "ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"
	p1.send(validTxStr)

	waitFor(p2, fmt.Sprintf(`"objectid":"%s"`, validTxID))
	p2.send(fmt.Sprintf(`{"objectid":"%s","type":"getobject"}`, validTxID))
	waitFor(p2, `"type":"object"`)

	// Test 3: Must reply with getobject when receiving ihaveobject
	fmt.Println("\n[Test 3] Reply to ihaveobject with getobject")
	dummyHash := "2d58067b3c6ec7f3a0ecfe5da40fa0fdc80a23e6db97db5c04d979ae1022e34a"
	p1.send(fmt.Sprintf(`{"objectid":"%s","type":"ihaveobject"}`, dummyHash))
	waitFor(p1, `"type":"getobject"`)

	// Test 4: UNKNOWN_OBJECT Error (Typo in TXID: ada666... instead of ada665...)
	fmt.Println("\n[Test 4] UNKNOWN_OBJECT Error")
	unknownObjTx := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ada6666818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"c8e5d64844ff82cfc0bd1f280090a8bc661fb7ab7105d06ea96141559451405eb0d77f9e857b207cc7e60c0996f798c29633d899076ef4be9fae9e851dcf2208"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":10}],"type":"transaction"},"type":"object"}`
	p1.send(unknownObjTx)
	expectError(p1, "UNKNOWN_OBJECT")

	// Test 5: INVALID_FORMAT Error (Missing 'value' field in output)
	fmt.Println("\n[Test 5] INVALID_FORMAT Error (Missing value)")
	missingValueTx := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"eba200cc97310da19ed88b1dc619c4dda371ae9c5c8aa7d673266b467870a0119c63a5b662381d9a3b3496a2410807c8000858730163cbb06e63951868b0790b"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4"}],"type":"transaction"},"type":"object"}`
	p1.send(missingValueTx)
	expectError(p1, "INVALID_FORMAT")

	// Test 6: INVALID_TX_OUTPOINT Error (Index 1 doesn't exist)
	fmt.Println("\n[Test 6] INVALID_TX_OUTPOINT Error")
	invalidOutpointTx := `{"object":{"inputs":[{"outpoint":{"index":1,"txid":"ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"51b9cef44429547e0061517f6d2b196bfa8b1a2a37870e5832489a6aafe32a7d8adf037170a396e68957c26480d6bca86f0ae85d4a6b4ef7a8752cb6da9ca903"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":10}],"type":"transaction"},"type":"object"}`
	p1.send(invalidOutpointTx)
	expectError(p1, "INVALID_TX_OUTPOINT")

	// Test 7: INVALID_TX_CONSERVATION Error (Outputs > Inputs)
	fmt.Println("\n[Test 7] INVALID_TX_CONSERVATION Error")
	invalidConsTx := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"15889fd5ef467ee0de549147d7bfd6c207da5f7afc1f1ad76f6f26ce937f2301809c78039a40cf370d0a6066dae92127b22831588515b491deeeb57df7c6c104"},{"outpoint":{"index":0,"txid":"ba46a995539f59b1095953117c3899c7396925f2da0d0cf0e0b572b506a14e7a"},"sig":"15889fd5ef467ee0de549147d7bfd6c207da5f7afc1f1ad76f6f26ce937f2301809c78039a40cf370d0a6066dae92127b22831588515b491deeeb57df7c6c104"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":5100000000011}],"type":"transaction"},"type":"object"}`
	p1.send(invalidConsTx)
	expectError(p1, "INVALID_TX_CONSERVATION")

	// Test 8: INVALID_TX_SIGNATURE Error (Bad types.Signature)
	fmt.Println("\n[Test 8] INVALID_TX_SIGNATURE Error")
	badSigTx := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"51b9cef44429547e0061517f6d2b196bfa8b1a2a37870e5832489a6aafe32a7d8adf037170a396e68957c26480d6bca86f0ae85d4a6b4ef7a8752cb6da9ca903"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":10}],"type":"transaction"},"type":"object"}`
	p1.send(badSigTx)
	expectError(p1, "INVALID_TX_SIGNATURE")

	// Test 9: INVALID_FORMAT Error (Negative value)
	fmt.Println("\n[Test 9] INVALID_FORMAT Error (Negative value)")
	negValueTx := `{"object":{"inputs":[{"outpoint":{"index":0,"txid":"ada6656818fec7c7ad431b17adb7b8abb136dd500c9f7b05568ec28f5ba48c63"},"sig":"eba200cc97310da19ed88b1dc619c4dda371ae9c5c8aa7d673266b467870a0119c63a5b662381d9a3b3496a2410807c8000858730163cbb06e63951868b0790b"}],"outputs":[{"pubkey":"39cd95f5cac18db4ca13e9a47b507811da4a6a158ba4a2f89e183e5123c52ae4","value":-10}],"type":"transaction"},"type":"object"}`
	p1.send(negValueTx)
	expectError(p1, "INVALID_FORMAT")

	// Test 10: Unknown Object Request (No Crash)
	fmt.Println("\n[Test 10] Node doesn't crash on getobject for unknown hash")
	unknownGetHash := "29a5302055c50015509bfb67df2c7a02a339e365692d0cd97b3a49ad3bdd23fc"
	p1.send(fmt.Sprintf(`{"objectid":"%s","type":"getobject"}`, unknownGetHash))
	expectError(p1, "UNKNOWN_OBJECT")

	fmt.Println("\n🎉 All Grader tests executed")
}
