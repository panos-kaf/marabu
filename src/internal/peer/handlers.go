package peer

import (
	"marabu/internal/crypto"
	"marabu/internal/messages"
	"strconv"
)

func (p *Peer) handleHello(msg *HelloMessage) {
	if msg.Agent != nil {
		p.log(msg.Type, string(*msg.Agent)+" ("+p.addr+") says hello, version: "+string(msg.T_Version))
	} else {
		p.log(msg.Type, "Peer "+p.addr+" says hello, version: "+string(msg.T_Version))
	}
	p.handshakeComplete = true
}

func (p *Peer) handleError(msg *ErrorMessage) {
	p.logErrorCode(msg.Name, " peer: "+p.addr+", description: "+string(msg.Description))
}

func (p *Peer) handleGetPeers() {
	peers := make(T_Peers, 0, len(knownPeers))
	for peer := range knownPeers {
		peers = append(peers, peer)
	}
	e := p.SendPeers(peers)
	if e != nil {
		p.err(MSG_PEERS, e.Error())
	}
}

func (p *Peer) handlePeers(msg *PeersMessage) {
	p.log(MSG_PEERS, "Peer "+p.addr+" sent "+strconv.Itoa(len(msg.T_Peers))+" peers")
	AppendPeers(msg.T_Peers, p.addr)
}

func (p *Peer) handleGetObject(msg *GetObjectMessage) {

	Log := func(m string) {
		p.log(MSG_GETOBJECT, m)
	}
	Err := func(m string) {
		p.err(MSG_GETOBJECT, m)
	}
	ID := msg.ObjectID
	Log("Peer: " + p.addr + " requested object: " + string(ID))

	exists, err := p.objectManager.Exists(ID)
	if err != nil {
		Err("Error checking if object exists: " + err.Error())
		return
	}
	if exists {
		Log("We have object " + string(ID) + ", sending it to peer " + p.addr)
		obj, err := p.objectManager.Get(ID)
		if err != nil {
			Err("Error retrieving object: " + err.Error())
			return
		}
		err = p.SendObject(obj)
		if err != nil {
			Err("Error sending object: " + err.Error())
		}
	} else {
		Log("We do not have object " + string(ID) + ", cannot fulfill MSG_GETOBJECT request from peer " + p.addr)
		p.SendError(E_UNKNOWN_OBJECT, "Object not found: "+string(ID))
	}
}

func (p *Peer) handleIHaveObject(msg *IHaveObjectMessage) {

	ID := msg.ObjectID
	p.log(msg.Type, "Peer: "+p.addr+"  has object with ID: "+string(ID))

	exists, e := p.objectManager.Exists(ID)
	if e != nil {
		p.err(msg.Type, "Error checking if object exists: "+e.Error())
		return
	}
	if exists {
		p.log(msg.Type, "We already have object "+string(ID))
	} else {
		p.log(msg.Type, "We do not have object "+string(ID)+", requesting it from peer "+p.addr)
		err := p.SendGetObject(ID)
		if err != nil {
			p.err(msg.Type, "Error sending MSG_GETOBJECT: "+err.Error())
		}
	}
}

func (p *Peer) handleObject(msg *ObjectMessage) {

	errorCode, err := p.ValidateObject(msg.Object)
	if err != nil {
		p.err(msg.Type, "Received invalid object from peer "+p.addr+": "+err.Error())
		p.SendError(errorCode, "Invalid object: "+err.Error())
		return
	}

	ID, err := crypto.HashObject(msg.Object)
	if err != nil {
		p.err(msg.Type, "Error hashing object: "+err.Error())
		return
	}

	hashID := T_HashID(ID)

	p.log(msg.Type, "Received MSG_OBJECT with ID "+ID+" from peer: "+p.addr)

	exists, err := p.objectManager.Exists(hashID)
	if err != nil {
		p.err(msg.Type, "Error checking if object exists: "+err.Error())
		return
	}
	if exists {
		p.log(msg.Type, "We already have object "+ID+", ignoring received object.")
	} else {
		p.log(msg.Type, "Storing new object with ID "+ID)

		_, err := p.objectManager.Put(msg.Object)
		if err != nil {
			p.err(msg.Type, "Error storing object: "+err.Error())
			return
		}
		p.log(msg.Type, "Object stored successfully with ID "+ID)

		// gossip!
		advertisement, err := messages.MakeIHaveObjectMessage(hashID)
		if err != nil {
			p.err(msg.Type, "Error creating MSG_IHAVEOBJECT message: "+err.Error())
			return
		}
		Broadcast(MSG_IHAVEOBJECT, E_NONE, advertisement, err)
	}
}

func (p *Peer) handleGetMempool() {
	p.log(MSG_GETMEMPOOL, "not handled yet")
}

func (p *Peer) handleMempool(msg *MempoolMessage) {
	p.log(msg.Type, "not handled yet")
}

func (p *Peer) handleGetChainTip() {
	p.log(MSG_GETCHAINTIP, "not handled yet")
}

func (p *Peer) handleChainTip(msg *ChainTipMessage) {
	p.log(msg.Type, "not handled yet")
}
