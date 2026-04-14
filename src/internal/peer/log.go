package peer

import (
	"marabu/internal/logs"
	"marabu/internal/types"
)

func globalLog(msg string) {
	logs.GlobalLog(msg)
}

func globalError(msg string) {
	logs.GlobalError(msg)
}

func (p *Peer) logInfo(message string) {
	entry := logs.LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     false,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) errInfo(message string) {
	entry := logs.LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) log(mtype types.MessageType, code types.ErrorCode, message string) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     false,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) err(mtype types.MessageType, code types.ErrorCode, message string) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) logMessage(mtype types.MessageType, code types.ErrorCode, sends bool) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     false,
		Role:        p.role,
	}
	entry.Role = p.role
	if sends {
		entry.Direction = "sent"
	} else {
		entry.Direction = "received"
	}
	logs.Log(entry)
}

func (p *Peer) errMessage(mtype types.MessageType, code types.ErrorCode, message string, sends bool) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	if sends {
		entry.Direction = "sent"
	} else {
		entry.Direction = "received"
	}
	logs.Log(entry)
}
