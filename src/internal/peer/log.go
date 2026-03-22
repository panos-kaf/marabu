package peer

import (
	"marabu/internal/logs"
)

func globalLog(msg string) {
	entry := logs.LogEntry{
		MessageType: MSG_NONE,
		ErrorCode:   E_NONE,
		ID:          0,
		Addr:        "",
		IsError:     false,
		Message:     msg,
		Role:        "",
	}
	logs.Log(entry)
}

func globalError(msg string) {
	entry := logs.LogEntry{
		MessageType: MSG_NONE,
		ErrorCode:   E_NONE,
		ID:          0,
		Addr:        "",
		IsError:     true,
		Message:     msg,
		Role:        "",
	}
	logs.Log(entry)
}

func (p *Peer) logInfo(message string) {
	entry := logs.LogEntry{
		MessageType: MSG_NONE,
		ErrorCode:   E_NONE,
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
		MessageType: MSG_NONE,
		ErrorCode:   E_NONE,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) log(mtype MessageType, message string) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   E_NONE,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     false,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) logErrorCode(code ErrorCode, message string) {
	entry := logs.LogEntry{
		MessageType: MSG_ERROR,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     false,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) err(mtype MessageType, message string) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   E_NONE,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) errErrorCode(code ErrorCode, message string) {
	entry := logs.LogEntry{
		MessageType: MSG_ERROR,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Message:     message,
		Role:        p.role,
	}
	logs.Log(entry)
}

func (p *Peer) logMessage(mtype MessageType, sends bool) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   E_NONE,
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

func (p *Peer) errMessage(mtype MessageType, message string, sends bool) {
	entry := logs.LogEntry{
		MessageType: mtype,
		ErrorCode:   E_NONE,
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

func (p *Peer) logMessageErrorCode(code ErrorCode, sends bool) {
	entry := logs.LogEntry{
		MessageType: MSG_ERROR,
		ErrorCode:   code,
		ID:          p.ID,
		Addr:        p.addr,
		IsError:     true,
		Role:        p.role,
	}
	if sends {
		entry.Direction = "sent"
	} else {
		entry.Direction = "received"
	}
	logs.Log(entry)
}

func (p *Peer) errMessageErrorCode(code ErrorCode, message string, sends bool) {
	entry := logs.LogEntry{
		MessageType: MSG_ERROR,
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
