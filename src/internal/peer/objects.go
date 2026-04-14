package peer

import (
	"fmt"
	"marabu/internal/core"
	"marabu/internal/types"
)

func (p *Peer) acceptObject(msgType types.MessageType, obj types.Object, result core.ValidationResult) {

	// Deduplication check
	exists, err := p.Manager.ExistsObject(result.ObjID)
	if err != nil {
		p.err(msgType, types.E_NONE, "DB Error: "+err.Error())
		return
	}
	if exists {
		p.log(msgType, types.E_NONE, "Already have object "+string(result.ObjID)+", ignoring.")
		return
	}

	// Store the object and apply state changes
	if err := p.Manager.CommitObject(obj, result); err != nil {
		p.err(msgType, types.E_NONE, "Failed to apply state: "+err.Error())
		return
	}
	p.logInfo(fmt.Sprintf("Successfully processed and stored %s", result.ObjID))

	// Gossip
	BroadcastIHaveObject(result.ObjID)

	// Resolve pending blocks that depended on this object
	p.resolvePendingBlocks(msgType, result.ObjID)
}

func (p *Peer) resolvePendingBlocks(msgType types.MessageType, resolvedObjID types.HashID) {
	// The processor gives us the pending blocks, keeping DB logic out of the Peer
	pendingBlocks := p.Manager.FetchPendingBlocks(resolvedObjID)

	for _, pending := range pendingBlocks {
		blk := pending.Block

		// Recursively validate and accept!
		result := p.Manager.ValidateObject(blk, p.addr)

		if result.ErrorCode == types.E_NONE && result.Error == nil {
			p.acceptObject(msgType, blk, result)
			p.log(msgType, types.E_NONE, fmt.Sprintf("Successfully validated pending block %s", result.ObjID))
		} else if result.ErrorCode != types.E_UNKNOWN_OBJECT {
			p.err(msgType, types.E_NONE, fmt.Sprintf("Pending block %s remains invalid: %v", result.ObjID, result.Error))
		}
	}
}
