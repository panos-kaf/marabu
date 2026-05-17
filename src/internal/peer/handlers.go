package peer

import (
	"errors"
	"fmt"
	"marabu/internal/core"
	"marabu/internal/discovery"
	"marabu/internal/protocol"
	"marabu/internal/types"
	"strconv"
)

func (p *Peer) handleHello(msg *protocol.Hello) {

	agent := "unknown"
	if msg.Agent != nil {
		agent = string(*msg.Agent)
	}

	if ConnManager.IsBanned(p.host) || ConnManager.IsBanned(agent) || ConnManager.IsBanned(p.addr) {
		p.err(msg.Type, types.E_NONE, fmt.Sprintf("Peer \"%s\" is banned.", agent))
		p.SendError(types.E_INVALID_HANDSHAKE, "Your agent is banned from this node.")
		p.Disconnect()
		return
	}

	p.log(msg.Type, types.E_NONE, string(*msg.Agent)+" ("+p.addr+") says hello, version: "+string(msg.Version))

	p.handshakeComplete = true

	p.agent = agent
	discovery.UpdatePeer(p.addr, agent)
}

func (p *Peer) handleError(msg *protocol.Error) {
	p.log(msg.Type, msg.Name, "peer: "+p.Name()+", description: "+string(msg.Description))
}

func (p *Peer) handleGetPeers() {

	knownPeers := discovery.GetKnownPeers()

	e := p.SendPeers(knownPeers)
	if e != nil {
		p.err(types.MSG_PEERS, types.E_NONE, e.Error())
	}
}

func (p *Peer) handlePeers(msg *protocol.Peers) {
	p.log(types.MSG_PEERS, types.E_NONE, p.Name()+" sent "+strconv.Itoa(len(msg.Peers))+" peers")
	discovery.AppendPeers(msg.Peers, p.addr)
}

func (p *Peer) handleGetObject(msg *protocol.GetObject) {
	ID := msg.ObjectID
	mtype := msg.Type

	p.log(mtype, types.E_NONE, p.Name()+" requested object: "+string(ID))

	obj, err := p.Manager.GetObject(ID)

	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			p.log(mtype, types.E_NONE, "We do not have object "+string(ID))
			p.SendError(types.E_UNKNOWN_OBJECT, "Object not found: "+string(ID))
			return
		}

		// We have it, but it's corrupted or the DB is failing
		p.err(mtype, types.E_NONE, "CRITICAL: Failed to read existing object "+string(ID)+": "+err.Error())
		p.SendError(types.E_INTERNAL_ERROR, "Internal node error while reading object.")
		return
	}

	p.log(mtype, types.E_NONE, "We have object "+string(ID)+", sending it to peer "+p.addr)

	if err := p.SendObject(obj); err != nil {
		p.err(mtype, types.E_NONE, "Error sending object: "+err.Error())
	}
}

func (p *Peer) handleIHaveObject(msg *protocol.IHaveObject) {

	ID := msg.ObjectID
	p.log(msg.Type, types.E_NONE, p.Name()+" has object with ID: "+string(ID))

	exists, e := p.Manager.ExistsObject(ID)
	if e != nil {
		p.err(msg.Type, types.E_NONE, "Error checking if object exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, types.E_NONE, "We already have object "+string(ID))
	} else {
		p.log(msg.Type, types.E_NONE, "We do not have object "+string(ID)+", requesting it from "+p.Name())
		err := p.SendGetObject(ID)
		if err != nil {
			p.err(msg.Type, types.E_NONE, "Error sending getobject: "+err.Error())
		}
	}
}

func (p *Peer) handleObject(msg *protocol.Object) {

	result := p.Manager.ValidateObject(msg.Object)

	if result.ErrorCode == types.E_INVALID_BLOCK_POW {
		p.SendError(types.E_INVALID_BLOCK_POW, "Invalid proof of work. You have been banned.")
		ConnManager.BanPeer(p.host)
		p.Disconnect()
		return
	}

	if result.Error != nil {
		p.rejectObject(msg.Object, result)
		return
	}

	p.acceptObject(msg.Object, result)
}

func (p *Peer) handleGetMempool() {

	mempool := p.Manager.GetMempoolTxids()

	// If mempool is nil, send an empty array instead of null to avoid JSON parsing issues on the peer side
	if mempool == nil {
		mempool = make([]types.HashID, 0)
	}

	p.SendMempool(mempool)
}

func (p *Peer) handleMempool(msg *protocol.Mempool) {

	for _, txid := range msg.Txids {
		if exists, err := p.Manager.ExistsObject(txid); err != nil {
			p.err(types.MSG_MEMPOOL, types.E_NONE, "Error checking if mempool transaction exists: "+err.Error())
			continue
		} else if exists {
			p.log(types.MSG_MEMPOOL, types.E_NONE, "We already have mempool transaction "+string(txid))
			continue
		}

		// only ask the peer that has the mempool
		p.SendGetObject(txid)
	}
}

func (p *Peer) handleGetChainTip() {

	p.log(types.MSG_GETCHAINTIP, types.E_NONE, p.Name()+" requested chain tip")

	tip, _, err := p.Manager.GetChaintip()
	if err != nil {

		if errors.Is(err, core.ErrNotFound) {
			p.log(types.MSG_GETCHAINTIP, types.E_NONE, "We do not have a chain tip.")
			p.SendError(types.E_UNKNOWN_OBJECT, "Chain tip not found")
			return
		}

		p.err(types.MSG_GETCHAINTIP, types.E_NONE, "CRITICAL: Failed to retrieve existing chain tip: "+err.Error())
		p.SendError(types.E_INTERNAL_ERROR, "Internal node error while retrieving chain tip")
		return
	}

	p.log(types.MSG_GETCHAINTIP, types.E_NONE, "Sending chain tip "+string(tip)+" to "+p.Name())

	err = p.SendChainTip(tip)
	if err != nil {
		p.err(types.MSG_GETCHAINTIP, types.E_NONE, "Error sending chain tip: "+err.Error())
	}
}

func (p *Peer) handleChainTip(msg *protocol.ChainTip) {

	p.log(msg.Type, types.E_NONE, p.Name()+" sent chain tip: "+string(msg.BlockID))

	if !p.sentChainTip {
		p.sentChainTip = true
		p.Manager.IncrementChaintipsReceived()
	}

	exists, e := p.Manager.ExistsObject(msg.BlockID)
	if e != nil {
		p.err(msg.Type, types.E_NONE, "Error checking if chain tip exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, types.E_NONE, "We already have chain tip "+string(msg.BlockID))
	} else {
		p.log(msg.Type, types.E_NONE, "We do not have chain tip "+string(msg.BlockID)+", requesting it from "+p.Name())
		err := p.SendGetObject(msg.BlockID)
		if err != nil {
			p.err(msg.Type, types.E_NONE, "Error sending getobject for chain tip: "+err.Error())
		}
	}

}
