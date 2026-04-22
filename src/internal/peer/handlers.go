package peer

import (
	"errors"
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
		p.log(msg.Type, types.E_NONE, string(*msg.Agent)+" ("+p.addr+") says hello, version: "+string(msg.Version))
	} else {
		p.log(msg.Type, types.E_NONE, "Peer "+p.addr+" says hello, version: "+string(msg.Version))
	}
	p.handshakeComplete = true

	p.agent = agent
	discovery.UpdateAgent(p.addr, agent)
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

	obj, err := p.Manager.GetObject(ID)

	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			p.log(mtype, types.E_NONE, "We do not have object "+string(ID)+", bouncing request.")
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
	p.log(msg.Type, types.E_NONE, "Peer: "+p.addr+" has object with ID: "+string(ID))

	exists, e := p.Manager.ExistsObject(ID)
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

	result := p.Manager.ValidateObject(msg.Object, p.addr)

	if result.Error != nil {
		p.err(msg.Type, result.ErrorCode, result.Error.Error())
		p.SendError(result.ErrorCode, result.Error.Error())

		if result.MissingID != types.DUMMY_HASH {
			BroadcastGetObject(result.MissingID)
		}
		return
	}

	p.acceptObject(msg.Type, msg.Object, result)
}

func (p *Peer) handleGetMempool() {
	p.log(types.MSG_GETMEMPOOL, types.E_NONE, "not handled yet")
}

func (p *Peer) handleMempool(msg *protocol.Mempool) {
	p.log(msg.Type, types.E_NONE, "not handled yet")
}

func (p *Peer) handleGetChainTip() {

	p.log(types.MSG_GETCHAINTIP, types.E_NONE, "Peer "+p.addr+" requested chain tip")

	tip, _, err := p.Manager.GetChaintip()
	if err != nil {

		if errors.Is(err, core.ErrNotFound) {
			p.log(types.MSG_GETCHAINTIP, types.E_NONE, "We do not have chain tip, bouncing request.")
			p.SendError(types.E_UNKNOWN_OBJECT, "Chain tip not found")
			return
		}

		p.err(types.MSG_GETCHAINTIP, types.E_NONE, "CRITICAL: Failed to retrieve existing chain tip: "+err.Error())
		p.SendError(types.E_INTERNAL_ERROR, "Internal node error while retrieving chain tip")
		return
	}

	p.log(types.MSG_GETCHAINTIP, types.E_NONE, "Sending chain tip "+string(tip)+" to peer "+p.addr)

	err = p.SendChainTip(tip)
	if err != nil {
		p.err(types.MSG_GETCHAINTIP, types.E_NONE, "Error sending chain tip: "+err.Error())
	}
}

func (p *Peer) handleChainTip(msg *protocol.ChainTip) {

	p.log(msg.Type, types.E_NONE, "Peer "+p.addr+" sent chain tip: "+string(msg.BlockID))

	exists, e := p.Manager.ExistsObject(msg.BlockID)
	if e != nil {
		p.err(msg.Type, types.E_NONE, "Error checking if chain tip exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, types.E_NONE, "We already have chain tip "+string(msg.BlockID))
	} else {
		p.log(msg.Type, types.E_NONE, "We do not have chain tip "+string(msg.BlockID)+", requesting it from peer "+p.addr)
		err := p.SendGetObject(msg.BlockID)
		if err != nil {
			p.err(msg.Type, types.E_NONE, "Error sending getobject for chain tip: "+err.Error())
		}
	}

}
