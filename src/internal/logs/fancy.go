package logs

import (
	"fmt"
	"log"
	"marabu/internal/messages"
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
	MessageType messages.MessageType
	ErrorCode   messages.ErrorCode
	ID          int
	Direction   string
	Addr        string
	IsError     bool
	Message     string
	Role        string
}

func MessageTypeColor(mtype messages.MessageType) string {
	switch mtype {
	case messages.MSG_HELLO:
		return GREEN
	case messages.MSG_ERROR:
		return RED
	case messages.MSG_GETPEERS, messages.MSG_PEERS:
		return CYAN
	case messages.MSG_GETOBJECT, messages.MSG_IHAVEOBJECT, messages.MSG_OBJECT:
		return YELLOW
	case messages.MSG_GETMEMPOOL, messages.MSG_MEMPOOL:
		return BLUE
	case messages.MSG_GETCHAINTIP, messages.MSG_CHAINTIP:
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
	if m.MessageType == messages.MSG_ERROR && m.ErrorCode != messages.E_NONE {
		label = string(m.ErrorCode)
	}

	if m.MessageType != messages.MSG_NONE {
		msg = fmt.Sprintf("%s%s[%s]", msg, msgcolor, label)
	}

	if m.Direction != "" && m.MessageType != messages.MSG_NONE {
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
