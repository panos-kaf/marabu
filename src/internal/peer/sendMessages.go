package peer

import (
	"fmt"
	"marabu/internal/messages"
)

// SendMessage sends a message to the peer.
// Not a top level function, intended to be paired with message constructors like messages.MakeHelloMessage().
// If mkErr is not nil, it returns that error instead of sending the message.
func (p *Peer) SendMessage(t MessageType, code ErrorCode, msg string, mkerr error) error {
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
// Intended to be paired with message constructors like messages.MakeHelloMessage().
func Broadcast(t MessageType, code ErrorCode, msg string, mkErr error) {
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
	msg, err := messages.MakeHelloMessage()
	return p.SendMessage(MSG_HELLO, E_NONE, msg, err)
}

func (p *Peer) SendError(name ErrorCode, description string) error {
	msg, err := messages.MakeErrorMessage(name, T_BuString(description))
	return p.SendMessage(MSG_ERROR, name, msg, err)
}

func (p *Peer) SendGetPeers() error {
	msg, err := messages.MakeGetPeersMessage()
	return p.SendMessage(MSG_GETPEERS, E_NONE, msg, err)
}

func (p *Peer) SendPeers(peers T_Peers) error {
	msg, err := messages.MakePeersMessage(peers)
	return p.SendMessage(MSG_PEERS, E_NONE, msg, err)
}

func (p *Peer) SendGetObject(objectID T_HashID) error {
	msg, err := messages.MakeGetObjectMessage(objectID)
	return p.SendMessage(MSG_GETOBJECT, E_NONE, msg, err)
}

func (p *Peer) SendIHaveObject(objectID T_HashID) error {
	msg, err := messages.MakeIHaveObjectMessage(objectID)
	return p.SendMessage(MSG_IHAVEOBJECT, E_NONE, msg, err)
}

func (p *Peer) SendObject(obj messages.Object) error {
	msg, err := messages.MakeObjectMessage(obj)
	return p.SendMessage(MSG_OBJECT, E_NONE, msg, err)
}

func (p *Peer) SendGetMempool() error {
	msg, err := messages.MakeGetMempoolMessage()
	return p.SendMessage(MSG_GETMEMPOOL, E_NONE, msg, err)
}

func (p *Peer) SendMempool(txIDs []T_HashID) error {
	msg, err := messages.MakeMempoolMessage(txIDs)
	return p.SendMessage(MSG_MEMPOOL, E_NONE, msg, err)
}

func (p *Peer) SendGetChainTip() error {
	msg, err := messages.MakeGetChainTipMessage()
	return p.SendMessage(MSG_GETCHAINTIP, E_NONE, msg, err)
}

func (p *Peer) SendChainTip(chainTip T_HashID) error {
	msg, err := messages.MakeChainTipMessage(chainTip)
	return p.SendMessage(MSG_CHAINTIP, E_NONE, msg, err)
}

func (p *Peer) Greet() {
	p.SendHello()
	p.SendGetPeers()
}
