package main

import (
	"bufio"
	"fmt"
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"net"
	"os"
)

func send(conn net.Conn, msg string) {
	fmt.Fprintf(conn, "%s\n", msg)
}

func receive(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	resp, _ := reader.ReadString('\n')
	return resp
}

func exchangeObject(objectID messages.T_HashID, objectMsg string, conn net.Conn, resp string) {
	// 1. Send ihaveobject
	ihaveMsg, _ := messages.MakeIHaveObjectMessage(objectID)
	send(conn, ihaveMsg)
	fmt.Println("Sent ihaveobject")

	// 2. Expect getobject
	resp = receive(conn)
	fmt.Println("Received:", resp)
	// Parse and check for getobject

	// 3. Send object
	send(conn, objectMsg)
	fmt.Println("Sent object")

	// 4. Expect ihaveobject gossip (optional, if you have multiple peers)
	resp = receive(conn)
	fmt.Println("Received:", resp)

	// 5. Send getobject for known object
	getObjMsg, _ := messages.MakeGetObjectMessage(objectID)
	send(conn, getObjMsg)
	fmt.Println("Sent getobject")

	// 6. Expect object response
	resp = receive(conn)
	fmt.Println("Received:", resp)

}

func main() {
	serverAddr := "localhost:18018" // Change to your server address
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 0. Greet the server
	helloMsg, _ := messages.MakeHelloMessage()
	send(conn, helloMsg)
	fmt.Println("Sent hello")

	resp := receive(conn)
	fmt.Println("Received:", resp)
	// Parse and check for hello response

	resp = receive(conn)
	fmt.Println("Received:", resp)
	// Parse and check for getpeers response

	height := messages.T_BuInt(0)
	val := messages.NewPicabu(50000000000)

	// 1. Coinbase transaction
	coinbaseTx := messages.T_CoinbaseTransaction{
		Type:   messages.OBJ_TRANSACTION,
		Height: &height,
		Outputs: []messages.T_TxOutput{
			{
				Pubkey: "958f8add086cc348e229a3b6590c71b7d7754e42134a127a50648bf07969d9a0",
				Value:  &val,
			},
		},
	}
	coinbaseIDstr, _ := crypto.HashObject(coinbaseTx)
	coinbaseID := messages.T_HashID(coinbaseIDstr)
	coinbaseMsg, _ := messages.MakeObjectMessage(coinbaseTx)
	fmt.Println("\n--- Coinbase T_Transaction Exchange ---")
	fmt.Printf("Coinbase object message:\n%s\n\n", coinbaseMsg)
	exchangeObject(coinbaseID, coinbaseMsg, conn, resp)

	// 2. Regular transaction
	sig := messages.T_Signature("060bf7cbe141fecfebf6dafbd6ebbcff25f82e729a7770f4f3b1f81a7ec8a0ce4b287597e609b822111bbe1a83d682ef14f018f8a9143cef25ecc9a8b0c1c405")
	idx := messages.T_BuInt(0)
	val2 := messages.NewPicabu(10)

	input := messages.T_TxInput{
		Outpoint: messages.T_Outpoint{Txid: coinbaseID, Index: &idx},
		Sig:        &sig,
	}

	val2 = messages.NewPicabu(10)

	output := messages.T_TxOutput{
		Pubkey: "958f8add086cc348e229a3b6590c71b7d7754e42134a127a50648bf07969d9a0",
		Value:  &val2,
	}

	regularTx := messages.T_Transaction{
		Type:    messages.OBJ_TRANSACTION,
		Inputs:  []messages.T_TxInput{input},
		Outputs: []messages.T_TxOutput{output},
	}

	regularIDstr, _ := crypto.HashObject(regularTx)
	regularID := messages.T_HashID(regularIDstr)
	regularMsg, _ := messages.MakeObjectMessage(regularTx)
	fmt.Println("\n--- Regular T_Transaction Exchange ---")
	fmt.Printf("Regular object message:\n%s\n\n", regularMsg)
	exchangeObject(regularID, regularMsg, conn, resp)
}
