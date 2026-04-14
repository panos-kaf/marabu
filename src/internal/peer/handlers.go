package peer

import (
	"fmt"
	"marabu/internal/discovery"
	"marabu/internal/protocol"
	"marabu/internal/types"
	"strconv"
)

func (p *Peer) handleHello(msg *protocol.Hello) {
	if msg.Agent != nil {
		p.log(msg.Type, types.E_NONE, string(*msg.Agent)+" ("+p.addr+") says hello, version: "+string(msg.Version))
	} else {
		p.log(msg.Type, types.E_NONE, "Peer "+p.addr+" says hello, version: "+string(msg.Version))
	}
	p.handshakeComplete = true
}

func (p *Peer) handleError(msg *protocol.Error) {
	p.log(msg.Type, msg.Name, "peer: "+p.addr+", description: "+string(msg.Description))
}

func (p *Peer) handleGetPeers() {

	knownPeers := discovery.GetKnownPeers()

	e := p.SendPeers(knownPeers)
	if e != nil {
		p.err(types.MSG_PEERS, types.E_NONE, e.Error())
	}
}

func (p *Peer) handlePeers(msg *protocol.Peers) {
	p.log(types.MSG_PEERS, types.E_NONE, "Peer "+p.addr+" sent "+strconv.Itoa(len(msg.Peers))+" peers")
	discovery.AppendPeers(msg.Peers, p.addr)
}

func (p *Peer) handleGetObject(msg *protocol.GetObject) {

	ID := msg.ObjectID

	mtype := msg.Type

	p.log(mtype, types.E_NONE, "Peer: "+p.addr+" requested object: "+string(ID))

	exists, err := p.Store.ExistsObject(ID)
	if err != nil {
		p.err(mtype, types.E_NONE, "Error checking if object exists: "+err.Error())
		return
	}
	if exists {
		p.log(mtype, types.E_NONE, "We have object "+string(ID)+", sending it to peer "+p.addr)
		obj, err := p.Store.GetObject(ID)
		if err != nil {
			p.err(mtype, types.E_NONE, "Error retrieving object: "+err.Error())
			return
		}
		err = p.SendObject(obj)
		if err != nil {
			p.err(mtype, types.E_NONE, "Error sending object: "+err.Error())
		}
	} else {
		p.log(mtype, types.E_NONE, "We do not have object "+string(ID)+", cannot fulfill getobject request from peer "+p.addr)
		p.SendError(types.E_UNKNOWN_OBJECT, "Object not found: "+string(ID))
	}
}

func (p *Peer) handleIHaveObject(msg *protocol.IHaveObject) {

	ID := msg.ObjectID
	p.log(msg.Type, types.E_NONE, "Peer: "+p.addr+" has object with ID: "+string(ID))

	exists, e := p.Store.ExistsObject(ID)
	if e != nil {
		p.err(msg.Type, types.E_NONE, "Error checking if object exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, types.E_NONE, "We already have object "+string(ID))
	} else {
		p.log(msg.Type, types.E_NONE, "We do not have object "+string(ID)+", requesting it from peer "+p.addr)
		err := p.SendGetObject(ID)
		if err != nil {
			p.err(msg.Type, types.E_NONE, "Error sending getobject: "+err.Error())
		}
	}
}

func (p *Peer) handleObject(msg *protocol.Object) {

	objID, fee, errorCode, err := p.ValidateObject(msg.Object)
	if err != nil {
		p.err(msg.Type, types.E_NONE, "Received invalid object from peer "+p.addr+": "+err.Error())
		p.SendError(errorCode, "Invalid object: "+err.Error())
		return
	}

	objIDstr := string(objID)
	p.log(msg.Type, types.E_NONE, "Received Object with ID "+objIDstr+" from peer: "+p.addr)

	exists, err := p.Store.ExistsObject(objID)
	if err != nil {
		p.err(msg.Type, types.E_NONE, "Error checking if object exists: "+err.Error())
		return
	}

	if exists {
		p.log(msg.Type, types.E_NONE, "We already have object "+objIDstr+", ignoring received object.")
	} else {

		_, err := p.Store.PutObject(msg.Object)
		if err != nil {
			p.err(msg.Type, types.E_NONE, "Error storing object: "+err.Error())
			return
		}
		p.log(msg.Type, types.E_NONE, "Object "+objIDstr+" stored successfully")

		if msg.Object.ObjectType() == types.OBJ_TRANSACTION {
			err = p.Store.PutFee(objID, fee)
			if err != nil {
				p.err(msg.Type, types.E_NONE, "Error storing fee for object: "+err.Error())
			}
		}

		// Notify pending blocks that this object is now available
		pendingBlocks := p.Store.PendingBlocks[objID]
		for _, pending := range pendingBlocks {
			blk := pending.Block
			_, _, code, err := p.ValidateObject(blk)
			if code == types.E_NONE && err == nil {
				p.log(msg.Type, types.E_NONE, "Pending block "+objIDstr+" is now valid with new object "+objIDstr)
				_, err := p.Store.PutObject(blk)
				if err != nil {
					p.err(msg.Type, types.E_NONE, "Error storing pending block: "+err.Error())
					continue
				}

				BroadcastIHaveObject(objID)

				connectedPeersMutex.Lock()
				pendingPeer, exists := connectedPeers[pending.Peer]
				connectedPeersMutex.Unlock()
				if exists {
					pendingPeer.log(msg.Type, types.E_NONE, "Successfully validated pending block "+objIDstr+" after receiving missing object "+objIDstr)
				}

			} else if code != types.E_UNKNOWN_OBJECT {

				connectedPeersMutex.Lock()
				pendingPeer, exists := connectedPeers[pending.Peer]
				connectedPeersMutex.Unlock()

				if exists {
					pendingPeer.err(msg.Type, types.E_NONE, fmt.Sprintf("Received object %s, but pending block %s is still invalid: %v", objIDstr, objIDstr, err))
					pendingPeer.SendError(code, "Pending block turned out invalid: "+err.Error())
				}

			}
		}
		delete(p.Store.PendingBlocks, objID)

		// gossip!
		BroadcastIHaveObject(objID)
	}
}

func (p *Peer) handleGetMempool() {
	p.log(types.MSG_GETMEMPOOL, types.E_NONE, "not handled yet")
}

func (p *Peer) handleMempool(msg *protocol.Mempool) {
	p.log(msg.Type, types.E_NONE, "not handled yet")
}

func (p *Peer) handleGetChainTip() {

	exists, err := p.Store.ExistsChaintip()
	if err != nil {
		p.err(types.MSG_GETCHAINTIP, types.E_NONE, "Error checking if chain tip exists: "+err.Error())
		return
	}
	if exists {
		tip, _, err := p.Store.GetChaintip()
		if err != nil {
			p.err(types.MSG_GETCHAINTIP, types.E_NONE, "Error retrieving chain tip: "+err.Error())
			return
		}
		err = p.SendChainTip(tip)
		if err != nil {
			p.err(types.MSG_GETCHAINTIP, types.E_NONE, "Error sending chain tip: "+err.Error())
		}
	} else {
		p.log(types.MSG_GETCHAINTIP, types.E_NONE, "No chain tip found in store")
		p.SendError(types.E_INTERNAL_ERROR, "No chain tip found")
	}
}

func (p *Peer) handleChainTip(msg *protocol.ChainTip) {
	p.log(msg.Type, types.E_NONE, "not handled yet")
}
