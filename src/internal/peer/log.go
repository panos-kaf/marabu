package peer

import (
	"cmp"
	"marabu/internal/logs"
	"marabu/internal/types"
)

func globalLog(msg string) {
	logs.GlobalLog(msg)
}

func globalError(msg string) {
	logs.GlobalError(msg)
}

func (p *Peer) isMuted() bool {
	return ConnManager.IsMuted(p.host) || ConnManager.IsMuted(p.agent) || ConnManager.IsMuted(p.addr)
}

func (p *Peer) logInfo(message string) {

	if p.isMuted() {
		return
	}
	entry := logs.LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     false,
		Message:     message,
		Origin:      p.origin,
	}
	logs.Log(entry)
}

func (p *Peer) errInfo(message string) {

	if p.isMuted() {
		return
	}

	entry := logs.LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     true,
		Message:     message,
		Origin:      p.origin,
	}
	logs.Log(entry)
}

func (p *Peer) log(mtype types.MessageType, code types.ErrorCode, message string) {

	if p.isMuted() {
		return
	}

	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     false,
		Message:     message,
		Origin:      p.origin,
	}
	logs.Log(entry)
}

func (p *Peer) err(mtype types.MessageType, code types.ErrorCode, message string) {

	if p.isMuted() {
		return
	}

	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     true,
		Message:     message,
		Origin:      p.origin,
	}
	logs.Log(entry)
}

func (p *Peer) logMessage(mtype types.MessageType, code types.ErrorCode, sends bool, message string) {

	if p.isMuted() {
		return
	}

	dir := types.Recv
	if sends {
		dir = types.Sent
	}

	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     false,
		Origin:      p.origin,
		Message:     message,
		Direction:   dir,
	}

	logs.Log(entry)
}

func (p *Peer) errMessage(mtype types.MessageType, code types.ErrorCode, message string, sends bool) {

	if p.isMuted() {
		return
	}

	dir := types.Recv
	if sends {
		dir = types.Sent
	}

	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        cmp.Or(p.agent, p.addr),
		IsError:     true,
		Message:     message,
		Origin:      p.origin,
		Direction:   dir,
	}

	logs.Log(entry)
}
