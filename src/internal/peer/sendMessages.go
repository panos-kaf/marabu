package peer

import (
	"fmt"
	"marabu/internal/crypto"
	"marabu/internal/protocol"
	"marabu/internal/types"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sent = true
	recv = false
)

// SendMessage sends a message to the peer.
// Not a top level function, intended to be paired with message constructors like protocol.MakeHello().
// If mkErr is not nil, it returns that error instead of sending the message.
func (p *Peer) SendMessage(t types.MessageType, code types.ErrorCode, msg string, mkerr error, log string) error {
	if mkerr != nil {
		return fmt.Errorf("Failed to create %s message: %w", t, mkerr)
	}

	p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	_, err := p.conn.Write([]byte(msg))

	p.conn.SetWriteDeadline(time.Time{})

	if err != nil {
		return fmt.Errorf("Failed to send %s message: %w", t, err)
	}

	p.logMessage(t, code, sent, log)

	return nil
}

// Broadcast sends a message to all connected peers.
// Intended to be paired with message constructors like protocol.MakeHello().
func Broadcast(t types.MessageType, code types.ErrorCode, msg string, mkErr error) {
	if mkErr != nil {
		globalError(fmt.Sprintf("Failed to create %s message: %v", t, mkErr))
		return
	}

	peersToBroadcast := ConnManager.GetAll()

	if len(peersToBroadcast) == 0 {
		return
	}

	// Use a WaitGroup to wait for all network calls to finish,
	// and an atomic boolean to safely track if any failed.
	var wg sync.WaitGroup
	var hasErrors atomic.Bool

	// Fire all messages concurrently
	for _, peer := range peersToBroadcast {
		wg.Add(1)

		// launch a goroutine for every peer
		go func(p *Peer) {
			defer wg.Done()

			if err := p.SendMessage(t, code, msg, nil, ""); err != nil {
				p.errInfo(fmt.Sprintf("Failed to broadcast %s message to %s: %v", t, p.addr, err))
				hasErrors.Store(true)
			}
		}(peer)
	}

	// Wait for all goroutines to finish sending
	wg.Wait()

	// Log the final result
	if hasErrors.Load() {
		globalError(fmt.Sprintf("Failed to broadcast %s message to some peers", t))
	} else {
		globalLog(fmt.Sprintf("Successfully broadcasted %s message to all peers", t))
	}
}

// -- Top Level Send Functions for each message type --

func (p *Peer) SendHello() error {

	agent := types.BuString(p.Manager.Config().AgentName)

	msg, err := protocol.MakeHello(&agent)
	return p.SendMessage(types.MSG_HELLO, types.E_NONE, msg, err, "")
}

func (p *Peer) SendError(name types.ErrorCode, description string) error {
	msg, err := protocol.MakeError(name, types.BuString(description))
	return p.SendMessage(types.MSG_ERROR, name, msg, err, "")
}

func (p *Peer) SendGetPeers() error {
	msg, err := protocol.MakeGetPeers()
	return p.SendMessage(types.MSG_GETPEERS, types.E_NONE, msg, err, "")
}

func (p *Peer) SendPeers(peers types.Peers) error {
	msg, err := protocol.MakePeers(peers)
	return p.SendMessage(types.MSG_PEERS, types.E_NONE, msg, err, "")
}

func (p *Peer) SendGetObject(objectID types.HashID) error {
	msg, err := protocol.MakeGetObject(objectID)

	logDetail := "ID: " + string(objectID)
	return p.SendMessage(types.MSG_GETOBJECT, types.E_NONE, msg, err, logDetail)
}

func (p *Peer) SendIHaveObject(objectID types.HashID) error {
	msg, err := protocol.MakeIHaveObject(objectID)

	logDetail := "ID: " + string(objectID)
	return p.SendMessage(types.MSG_IHAVEOBJECT, types.E_NONE, msg, err, logDetail)
}

func (p *Peer) SendObject(obj types.Object) error {
	msg, err := protocol.MakeObject(obj)

	hashID, _ := crypto.GetObjectID(obj)
	logDetail := "ID: " + string(hashID)

	return p.SendMessage(types.MSG_OBJECT, types.E_NONE, msg, err, logDetail)
}

func (p *Peer) SendGetMempool() error {
	msg, err := protocol.MakeGetMempool()
	return p.SendMessage(types.MSG_GETMEMPOOL, types.E_NONE, msg, err, "")
}

func (p *Peer) SendMempool(txIDs []types.HashID) error {
	msg, err := protocol.MakeMempool(txIDs)
	return p.SendMessage(types.MSG_MEMPOOL, types.E_NONE, msg, err, "")
}

func (p *Peer) SendGetChainTip() error {
	msg, err := protocol.MakeGetChainTip()
	return p.SendMessage(types.MSG_GETCHAINTIP, types.E_NONE, msg, err, "")
}

func (p *Peer) SendChainTip(chainTip types.HashID) error {
	msg, err := protocol.MakeChainTip(chainTip)
	return p.SendMessage(types.MSG_CHAINTIP, types.E_NONE, msg, err, string(chainTip))
}

func (p *Peer) Greet() {
	p.SendHello()
	p.SendGetPeers()
	p.SendGetChainTip()

	// we wait to sync before requesting mempool
	// p.SendGetMempool()
}

// -- Top level broadcast functions for each message type --

func BroadcastHello() {
	msg, err := protocol.MakeHello(nil)
	Broadcast(types.MSG_HELLO, types.E_NONE, msg, err)
}

func BroadcastGetPeers() {
	msg, err := protocol.MakeGetPeers()
	Broadcast(types.MSG_GETPEERS, types.E_NONE, msg, err)
}

func BroadcastPeers(peers types.Peers) {
	msg, err := protocol.MakePeers(peers)
	Broadcast(types.MSG_PEERS, types.E_NONE, msg, err)
}

func BroadcastGetObject(objectID types.HashID) {
	msg, err := protocol.MakeGetObject(objectID)
	Broadcast(types.MSG_GETOBJECT, types.E_NONE, msg, err)
}

func BroadcastIHaveObject(objectID types.HashID) {
	msg, err := protocol.MakeIHaveObject(objectID)
	Broadcast(types.MSG_IHAVEOBJECT, types.E_NONE, msg, err)
}

func BroadcastObject(obj types.Object) {
	msg, err := protocol.MakeObject(obj)
	Broadcast(types.MSG_OBJECT, types.E_NONE, msg, err)
}

func BroadcastGetMempool() {
	msg, err := protocol.MakeGetMempool()
	Broadcast(types.MSG_GETMEMPOOL, types.E_NONE, msg, err)
}

func BroadcastMempool(txIDs []types.HashID) {
	msg, err := protocol.MakeMempool(txIDs)
	Broadcast(types.MSG_MEMPOOL, types.E_NONE, msg, err)
}

func BroadcastGetChainTip() {
	msg, err := protocol.MakeGetChainTip()
	Broadcast(types.MSG_GETCHAINTIP, types.E_NONE, msg, err)
}

func BroadcastChainTip(chainTip types.HashID) {
	msg, err := protocol.MakeChainTip(chainTip)
	Broadcast(types.MSG_CHAINTIP, types.E_NONE, msg, err)
}
