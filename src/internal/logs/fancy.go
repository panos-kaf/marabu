package logs

import (
	"fmt"
	"log"
	"marabu/internal/types"
)

const (
	RED     = "\033[31m"
	GREEN   = "\033[32m"
	YELLOW  = "\033[33m"
	BLUE    = "\033[34m"
	MAGENTA = "\033[35m"
	CYAN    = "\033[36m"
	WHITE   = "\033[37m"

	BOLD   = "\033[1m"
	ITALIC = "\033[3m"

	RESET = "\033[0m"
)

type LogEntry struct {
	MessageType types.MessageType
	ErrorCode   types.ErrorCode
	ID          int
	Direction   string
	Addr        string
	IsError     bool
	Message     string
	Role        string
}

func MessageTypeColor(mtype types.MessageType) string {
	switch mtype {
	case types.MSG_HELLO:
		return GREEN
	case types.MSG_ERROR:
		return RED
	case types.MSG_GETPEERS, types.MSG_PEERS:
		return CYAN
	case types.MSG_GETOBJECT, types.MSG_IHAVEOBJECT, types.MSG_OBJECT:
		return YELLOW
	case types.MSG_GETMEMPOOL, types.MSG_MEMPOOL:
		return BLUE
	case types.MSG_GETCHAINTIP, types.MSG_CHAINTIP:
		return MAGENTA
	default:
		return RESET
	}
}

func RoleColor(role string) string {
	switch role {
	case "client":
		return BLUE
	case "server":
		return MAGENTA
	default:
		return RESET
	}
}

func Log(m LogEntry) {

	rolecolor := RoleColor(m.Role)
	msgcolor := MessageTypeColor(m.MessageType)

	id := ""
	if m.ID == 0 {
		id = "[*]"
	} else {
		id = fmt.Sprintf("[%d]", m.ID)
	}

	msg := fmt.Sprintf("%s%s%s", BOLD, rolecolor, id)

	label := string(m.MessageType)
	if m.MessageType == types.MSG_ERROR && m.ErrorCode != types.E_NONE {
		label = string(m.ErrorCode)
	}

	if m.MessageType != types.MSG_NONE {
		msg = fmt.Sprintf("%s%s[%s]", msg, msgcolor, label)
	}

	if m.Direction != "" && m.MessageType != types.MSG_NONE {
		direction := ""
		switch m.Direction {
		case "sent":
			direction = CYAN + "[-->]"
		case "received":
			direction = YELLOW + "[<--]"
		}
		msg = fmt.Sprintf("%s%s%s[%s]", msg, direction, WHITE, m.Addr)
	}

	if m.IsError {
		msg = fmt.Sprintf("%s %s%sError: %s%s\n", msg, RESET, RED, m.Message, RESET)
	} else {
		msg = fmt.Sprintf("%s %s%s\n", msg, RESET, m.Message)
	}

	log.Print(msg)
}

func GlobalLog(msg string) {
	entry := LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          0,
		Addr:        "",
		IsError:     false,
		Message:     msg,
		Role:        "",
	}
	Log(entry)
}

func GlobalError(msg string) {
	entry := LogEntry{
		MessageType: types.MSG_NONE,
		ErrorCode:   types.E_NONE,
		ID:          0,
		Addr:        "",
		IsError:     true,
		Message:     msg,
		Role:        "",
	}
	Log(entry)
}