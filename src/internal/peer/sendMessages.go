package peer

import (
	"fmt"
	"marabu/internal/protocol"
	"marabu/internal/types"
)

const (
	sent = true
	recv = false
)

// SendMessage sends a message to the peer.
// Not a top level function, intended to be paired with message constructors like protocol.MakeHello().
// If mkErr is not nil, it returns that error instead of sending the message.
func (p *Peer) SendMessage(t types.MessageType, code types.ErrorCode, msg string, mkerr error) error {
	if mkerr != nil {
		return fmt.Errorf("Failed to create %s message: %w", t, mkerr)
	}
	_, err := p.conn.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("Failed to send %s message: %w", t, err)
	}

	p.logMessage(t, code, sent)

	return nil
}

// Broadcast sends a message to all connected peers.
// Intended to be paired with message constructors like protocol.MakeHello().
func Broadcast(t types.MessageType, code types.ErrorCode, msg string, mkErr error) {
	if mkErr != nil {
		globalError(fmt.Sprintf("Failed to create %s message: %v", t, mkErr))
		return
	}
	connectedPeersMutex.Lock()
	defer connectedPeersMutex.Unlock()
	var hasErrors bool
	for _, peer := range connectedPeers {
		if err := peer.SendMessage(t, code, msg, nil); err != nil {
			peer.errInfo(fmt.Sprintf("Failed to broadcast %s message to %s: %v", t, peer.addr, err))
			hasErrors = true
		}
	}
	if hasErrors {
		globalError(fmt.Sprintf("Failed to broadcast %s message to some peers", t))
	} else {
		globalLog(fmt.Sprintf("Successfully broadcasted %s message to all peers", t))
	}
}

// -- Top Level Send Functions for each message type --

func (p *Peer) SendHello() error {
	msg, err := protocol.MakeHello()
	return p.SendMessage(types.MSG_HELLO, types.E_NONE, msg, err)
}

func (p *Peer) SendError(name types.ErrorCode, description string) error {
	msg, err := protocol.MakeError(name, types.BuString(description))
	return p.SendMessage(types.MSG_ERROR, name, msg, err)
}

func (p *Peer) SendGetPeers() error {
	msg, err := protocol.MakeGetPeers()
	return p.SendMessage(types.MSG_GETPEERS, types.E_NONE, msg, err)
}

func (p *Peer) SendPeers(peers types.Peers) error {
	msg, err := protocol.MakePeers(peers)
	return p.SendMessage(types.MSG_PEERS, types.E_NONE, msg, err)
}

func (p *Peer) SendGetObject(objectID types.HashID) error {
	msg, err := protocol.MakeGetObject(objectID)
	return p.SendMessage(types.MSG_GETOBJECT, types.E_NONE, msg, err)
}

func (p *Peer) SendIHaveObject(objectID types.HashID) error {
	msg, err := protocol.MakeIHaveObject(objectID)
	return p.SendMessage(types.MSG_IHAVEOBJECT, types.E_NONE, msg, err)
}

func (p *Peer) SendObject(obj types.Object) error {
	msg, err := protocol.MakeObject(obj)
	return p.SendMessage(types.MSG_OBJECT, types.E_NONE, msg, err)
}

func (p *Peer) SendGetMempool() error {
	msg, err := protocol.MakeGetMempool()
	return p.SendMessage(types.MSG_GETMEMPOOL, types.E_NONE, msg, err)
}

func (p *Peer) SendMempool(txIDs []types.HashID) error {
	msg, err := protocol.MakeMempool(txIDs)
	return p.SendMessage(types.MSG_MEMPOOL, types.E_NONE, msg, err)
}

func (p *Peer) SendGetChainTip() error {
	msg, err := protocol.MakeGetChainTip()
	return p.SendMessage(types.MSG_GETCHAINTIP, types.E_NONE, msg, err)
}

func (p *Peer) SendChainTip(chainTip types.HashID) error {
	msg, err := protocol.MakeChainTip(chainTip)
	return p.SendMessage(types.MSG_CHAINTIP, types.E_NONE, msg, err)
}

func (p *Peer) Greet() {
	p.SendHello()
	p.SendGetPeers()
	p.SendGetChainTip()
}

// -- Top level broadcast functions for each message type --

func BroadcastHello() {
	Broadcast(types.MSG_HELLO, types.E_NONE, "Broadcasting hello message to all peers", nil)
}

func BroadcastGetPeers() {
	Broadcast(types.MSG_GETPEERS, types.E_NONE, "Broadcasting getpeers message to all peers", nil)
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
