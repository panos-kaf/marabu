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

	isMempoolConflict := false
	if tx, ok := obj.(*types.Transaction); ok {
		for _, input := range tx.Inputs {
			outpoint := core.OutpointKey{Txid: input.Outpoint.Txid, Index: int(*input.Outpoint.Index)}
			if p.Manager.IsInputSpent(outpoint) {
				isMempoolConflict = true
				break
			}
		}

		// If it's a conflict and we don't need it for a block, drop it
		if isMempoolConflict && !p.Manager.IsNeededForPendingBlock(result.ObjID) {
			p.err(msgType, types.E_INVALID_TX_OUTPOINT, "Mempool double-spend detected.")
			return
		}
	}

	// Store the object and apply state changes
	if err := p.Manager.CommitObject(obj, result); err != nil {
		p.err(msgType, types.E_NONE, "Failed to apply state: "+err.Error())
		return
	}
	p.logInfo(fmt.Sprintf("Successfully processed and stored %s", result.ObjID))

	if tx, ok := obj.(*types.Transaction); ok {

		// Only add to mempool and gossip if it ISN'T a conflict
		if !isMempoolConflict {
			p.Manager.AddToMempool(tx, result.Fee)
			BroadcastIHaveObject(result.ObjID)
		} else {
			p.logInfo(fmt.Sprintf("Tx %s saved for pending block, but withheld from mempool due to conflict.", result.ObjID))
		}
	} else {
		// gossip
		BroadcastIHaveObject(result.ObjID)
	}
	// Resolve pending blocks that depended on this object
	p.resolvePendingBlocks(msgType, result.ObjID)
}

func (p *Peer) resolvePendingBlocks(msgType types.MessageType, resolvedObjID types.HashID) {

	// Fetch and clear pending blocks waiting on this object
	pendingBlocks := p.Manager.FetchPendingBlocks(resolvedObjID)

	for _, pending := range pendingBlocks {
		blk := pending.Block

		// Revalidate the block now that the missing object has arrived
		result := p.Manager.ValidateObject(blk, p.addr)

		if result.ErrorCode == types.E_NONE && result.Error == nil {
			// The block is valid
			p.log(msgType, types.E_NONE, fmt.Sprintf("Successfully validated pending block %s", result.ObjID))
			p.acceptObject(msgType, blk, result)

		} else if result.ErrorCode == types.E_UNKNOWN_OBJECT {
			// Still missing objects (e.g., the block was missing multiple txs)
			// We DO NOT send an error here. We just ask for the next missing piece.
			p.log(msgType, types.E_NONE, fmt.Sprintf("Pending block still missing objects. Asking for %s", result.MissingID))
			if result.MissingID != types.DUMMY_HASH {
				BroadcastGetObject(result.MissingID)
			}

		} else {
			// Invalid block
			p.err(msgType, types.E_NONE, fmt.Sprintf("Pending block %s remains invalid: %v", result.ObjID, result.Error))
			p.SendError(result.ErrorCode, result.Error.Error())
		}
	}
}
