package peer

import (
	"fmt"
	"marabu/internal/core"
	"marabu/internal/types"
)

func (p *Peer) acceptObject(obj types.Object, result core.ValidationResult) {

	exists, _ := p.Manager.ExistsObject(result.ObjID)
	if exists {
		return
	}

	gossip, errorCode, err := p.Manager.CommitObject(obj, result)
	if err != nil {
		p.err(types.MSG_OBJECT, errorCode, "Failed to apply state: "+err.Error())
		return
	}

	if errorCode != types.E_NONE {
		p.SendError(errorCode, "Transaction stored, but rejected from mempool.")
	}

	if gossip {
		BroadcastIHaveObject(result.ObjID)
	}

	// We received a valid object, se we reset the timeout for pending blocks
	p.Manager.ResetSyncTimer()

	p.resolvePendingBlocks(types.MSG_OBJECT, result.ObjID)
}

func (p *Peer) rejectObject(obj types.Object, result core.ValidationResult) {
	if result.ErrorCode == types.E_UNKNOWN_OBJECT && result.MissingID != types.DUMMY_HASH {

		p.log(types.MSG_OBJECT, types.E_NONE, fmt.Sprintf("Missing objects. Pausing block and requesting %s", result.MissingID))

		// We received an object with missing dependencies. Reset timer for pending blocks and ask for the missing object.
		p.Manager.ResetSyncTimer()

		if blk, ok := obj.(*types.Block); ok {
			p.Manager.AddPendingBlock(p.addr, result.MissingID, blk)
		}

		p.SendGetObject(result.MissingID)
		return
	}

	p.err(types.MSG_OBJECT, result.ErrorCode, result.Error.Error())
	p.SendError(result.ErrorCode, result.Error.Error())
}

func (p *Peer) resolvePendingBlocks(msgType types.MessageType, resolvedObjID types.HashID) {

	// Fetch and clear pending blocks waiting on this object
	pendingBlocks := p.Manager.FetchPendingBlocks(resolvedObjID)

	for _, pending := range pendingBlocks {
		blk := pending.Block

		// Revalidate the block now that the missing object has arrived
		result := p.Manager.ValidateObject(blk)

		if result.ErrorCode == types.E_NONE && result.Error == nil {
			// The block is valid
			p.log(msgType, types.E_NONE, fmt.Sprintf("Successfully validated pending block %s", result.ObjID))
			p.acceptObject(blk, result)

		} else if result.ErrorCode == types.E_UNKNOWN_OBJECT {
			// Still missing objects (e.g., the block was missing multiple txs)
			// We DO NOT send an error here. We just ask for the next missing piece.
			p.log(msgType, types.E_NONE, fmt.Sprintf("Pending block still missing objects. Asking for %s", result.MissingID))

			if result.MissingID != types.DUMMY_HASH {
				p.Manager.AddPendingBlock(p.addr, result.MissingID, blk)
				p.SendGetObject(result.MissingID)
			}

		} else {
			// Invalid block
			p.err(msgType, types.E_NONE, fmt.Sprintf("Pending block %s remains invalid: %v", result.ObjID, result.Error))
			p.SendError(result.ErrorCode, result.Error.Error())
		}
	}
}
