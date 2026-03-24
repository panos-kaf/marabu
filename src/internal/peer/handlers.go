package peer

import (
	"fmt"
	"strconv"
)

func (p *Peer) handleHello(msg *HelloMessage) {
	if msg.Agent != nil {
		p.log(msg.Type, E_NONE, string(*msg.Agent)+" ("+p.addr+") says hello, version: "+string(msg.T_Version))
	} else {
		p.log(msg.Type, E_NONE, "Peer "+p.addr+" says hello, version: "+string(msg.T_Version))
	}
	p.handshakeComplete = true
}

func (p *Peer) handleError(msg *ErrorMessage) {
	p.log(msg.Type, msg.Name, "peer: "+p.addr+", description: "+string(msg.Description))
}

func (p *Peer) handleGetPeers() {
	peers := make(T_Peers, 0, len(knownPeers))
	for peer := range knownPeers {
		peers = append(peers, peer)
	}
	e := p.SendPeers(peers)
	if e != nil {
		p.err(MSG_PEERS, E_NONE, e.Error())
	}
}

func (p *Peer) handlePeers(msg *PeersMessage) {
	p.log(MSG_PEERS, E_NONE, "Peer "+p.addr+" sent "+strconv.Itoa(len(msg.T_Peers))+" peers")
	AppendPeers(msg.T_Peers, p.addr)
}

func (p *Peer) handleGetObject(msg *GetObjectMessage) {

	ID := msg.ObjectID

	mtype := msg.Type

	p.log(mtype, E_NONE, "Peer: "+p.addr+" requested object: "+string(ID))

	exists, err := p.objectManager.Exists(ID)
	if err != nil {
		p.err(mtype, E_NONE, "Error checking if object exists: "+err.Error())
		return
	}
	if exists {
		p.log(mtype, E_NONE, "We have object "+string(ID)+", sending it to peer "+p.addr)
		obj, err := p.objectManager.Get(ID)
		if err != nil {
			p.err(mtype, E_NONE, "Error retrieving object: "+err.Error())
			return
		}
		err = p.SendObject(obj)
		if err != nil {
			p.err(mtype, E_NONE, "Error sending object: "+err.Error())
		}
	} else {
		p.log(mtype, E_NONE, "We do not have object "+string(ID)+", cannot fulfill getobject request from peer "+p.addr)
		p.SendError(E_UNKNOWN_OBJECT, "Object not found: "+string(ID))
	}
}

func (p *Peer) handleIHaveObject(msg *IHaveObjectMessage) {

	ID := msg.ObjectID
	p.log(msg.Type, E_NONE, "Peer: "+p.addr+"  has object with ID: "+string(ID))

	exists, e := p.objectManager.Exists(ID)
	if e != nil {
		p.err(msg.Type, E_NONE, "Error checking if object exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, E_NONE, "We already have object "+string(ID))
	} else {
		p.log(msg.Type, E_NONE, "We do not have object "+string(ID)+", requesting it from peer "+p.addr)
		err := p.SendGetObject(ID)
		if err != nil {
			p.err(msg.Type, E_NONE, "Error sending getobject: "+err.Error())
		}
	}
}

func (p *Peer) handleObject(msg *ObjectMessage) {

	objID, errorCode, err := p.ValidateObject(msg.Object)
	if err != nil {
		p.err(msg.Type, E_NONE, "Received invalid object from peer "+p.addr+": "+err.Error())
		p.SendError(errorCode, "Invalid object: "+err.Error())
		return
	}

	objIDstr := string(objID)
	p.log(msg.Type, E_NONE, "Received Object with ID "+objIDstr+" from peer: "+p.addr)

	exists, err := p.objectManager.Exists(objID)
	if err != nil {
		p.err(msg.Type, E_NONE, "Error checking if object exists: "+err.Error())
		return
	}

	if exists {
		p.log(msg.Type, E_NONE, "We already have object "+objIDstr+", ignoring received object.")
	} else {
		_, err := p.objectManager.Put(msg.Object)
		if err != nil {
			p.err(msg.Type, E_NONE, "Error storing object: "+err.Error())
			return
		}
		p.log(msg.Type, E_NONE, "Object "+objIDstr+" stored successfully")

		// Notify pending blocks that this object is now available
		pendingBlocks := p.objectManager.PendingBlocks[objID]
		for _, block := range pendingBlocks {
			blk := block.Block
			_, code, err := p.ValidateObject(blk)
			if code == E_NONE && err == nil {
				p.log(MSG_OBJECT, E_NONE, "Pending block "+objIDstr+" is now valid with new object "+objIDstr)
				_, err := p.objectManager.Put(blk)
				if err != nil {
					p.err(MSG_OBJECT, E_NONE, "Error storing pending block: "+err.Error())
					continue
				}

				BroadcastIHaveObject(objID)

				connectedPeersMutex.Lock()
				pendingPeer, exists := connectedPeers[block.Peer]
				connectedPeersMutex.Unlock()
				if exists {
					pendingPeer.log(MSG_OBJECT, E_NONE, "Successfully validated pending block "+objIDstr+" after receiving missing object "+objIDstr)
				}

			} else if code != E_UNKNOWN_OBJECT {

				connectedPeersMutex.Lock()
				pendingPeer, exists := connectedPeers[block.Peer]
				connectedPeersMutex.Unlock()

				if exists {
					pendingPeer.err(MSG_OBJECT, E_NONE, fmt.Sprintf("Received object %s, but pending block %s is still invalid: %v", objIDstr, objIDstr, err))
					pendingPeer.SendError(code, "Pending block turned out invalid: "+err.Error())
				}

			}
		}
		delete(p.objectManager.PendingBlocks, objID)

		// gossip!
		BroadcastIHaveObject(objID)
	}
}

func (p *Peer) handleGetMempool() {
	p.log(MSG_GETMEMPOOL, E_NONE, "not handled yet")
}

func (p *Peer) handleMempool(msg *MempoolMessage) {
	p.log(msg.Type, E_NONE, "not handled yet")
}

func (p *Peer) handleGetChainTip() {
	p.log(MSG_GETCHAINTIP, E_NONE, "not handled yet")
}

func (p *Peer) handleChainTip(msg *ChainTipMessage) {
	p.log(msg.Type, E_NONE, "not handled yet")
}
