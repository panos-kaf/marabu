package peer

import (
	"bufio"
	"fmt"
	"io"
	"marabu/internal/logs"
	"marabu/internal/messages"
	"marabu/internal/object"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	connectedPeers      = make(map[string]*Peer)
	connectedPeersMutex sync.Mutex
	connectedPeersCnt   = 0
)

type Peer struct {
	conn              net.Conn
	addr              string
	ID                int
	buffer            []byte
	handshakeComplete bool
	onDisconnect      func()
	onLog             func(MessageType, string)
	onLogErr          func(MessageType, string)
	onLogMessage      func(MessageType, ErrorCode, bool)
	onLogMessageError func(MessageType, ErrorCode, string, bool)
	role              string
	objectManager     *object.ObjectManager
}

// NewPeer creates a new Peer instance for a given network connection.
// It initializes the peer's state and starts a goroutine
// to handle incoming messages from the connection.
func NewPeer(conn net.Conn,
	role string,
	objectManager *object.ObjectManager,
	onDisconnect func(),
	onLog func(MessageType, string),
	onLogErr func(MessageType, string),
	onLogMessage func(MessageType, ErrorCode, bool),
	onLogMessageError func(MessageType, ErrorCode, string, bool)) *Peer {

	addr := conn.RemoteAddr().String()
	p := &Peer{
		conn:              conn,
		addr:              addr,
		buffer:            make([]byte, 0),
		onLog:             onLog,
		onLogErr:          onLogErr,
		onLogMessage:      onLogMessage,
		onLogMessageError: onLogMessageError,
		onDisconnect:      onDisconnect,
		role:              role,
		objectManager:     objectManager,
	}

	connectedPeersMutex.Lock()
	connectedPeers[addr] = p
	p.ID = connectedPeersCnt
	connectedPeersCnt++
	connectedPeersMutex.Unlock()

	go p.initializeSocket()

	return p
}

// initializeSocket starts a goroutine to read messages from the peer's connection.
// It continuously reads lines from the connection, and for each line, it calls handleMessage.
// On error it disconnects and removes the peer from the connectedPeers map.
func (p *Peer) initializeSocket() {
	reader := bufio.NewReader(p.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {

			connectedPeersMutex.Lock()
			delete(connectedPeers, p.addr)
			connectedPeersMutex.Unlock()

			if err != io.EOF {
				p.logErr(MSG_NONE, "Disconnected: "+err.Error())
				return
			}
			if p.onDisconnect != nil {
				p.onDisconnect()
				return
			}
		}
		p.handleMessage(line)
	}
}

// handleMessage processes incoming messages,
// ensuring they are valid and dispatching them
// to the appropriate handler based on their type.
func (p *Peer) handleMessage(raw string) {

	if len(strings.TrimSpace(raw)) == 0 {
		p.log(MSG_NONE, "Received empty message")
		return
	}

	// Unmarshal and validate message
	var msg Message
	msg, err, code := messages.UnmarshalMessage(raw)

	if err != nil {
		p.logErr(MSG_NONE, "Invalid message: "+err.Error())
		p.SendError(code, "Could not validate JSON message: "+err.Error())
		if !p.handshakeComplete {
			p.conn.Close()
		}
		return
	}

	// Must only handle Object validation now.

	// if err, code := msg.Validate(); err != nil {
	// 	p.logMessageError(msg.MessageType(), code, "Message validation failed: "+err.Error(), false)
	// 	p.SendError(code, "Message validation failed: "+err.Error())

	// 	if !p.handshakeComplete {
	// 		p.conn.Close()
	// 	}
	// 	return
	// }

	p.logMessage(msg.MessageType(), code, false)

	if !p.handshakeComplete && msg.MessageType() != messages.MSG_HELLO {
		p.logMessageError(msg.MessageType(), messages.E_INVALID_HANDSHAKE, "Expected MSG_HELLO message first", false)
		p.SendError(messages.E_INVALID_HANDSHAKE, "Handshake not completed, expected hello message but received "+string(msg.MessageType()))
		p.conn.Close()
		return
	}

	// Dispatch based on type
	switch m := msg.(type) {
	case *HelloMessage:
		p.handleHello(m)
	case *ErrorMessage:
		p.handleError(m)
	case *GetPeersMessage:
		p.handleGetPeers()
	case *PeersMessage:
		p.handlePeers(m)
	case *GetObjectMessage:
		p.handleGetObject(m)
	case *IHaveObjectMessage:
		p.handleIHaveObject(m)
	case *ObjectMessage:
		p.handleObject(m)
	case *GetMempoolMessage:
		p.handleGetMempool()
	case *MempoolMessage:
		p.handleMempool(m)
	case *GetChainTipMessage:
		p.handleGetChainTip()
	case *ChainTipMessage:
		p.handleChainTip(m)
	default:
		p.logErr(MSG_NONE, "Unknown message type")
		p.SendError(messages.E_INVALID_FORMAT, "Unknown protocol message")
		p.conn.Close()
	}
}

func globalLog(msg string) {
	logs.GlobalLog(msg)
}

func globalError(msg string) {
	logs.GlobalError(msg)
}

func (p *Peer) log(mtype MessageType, msg string) {
	if p.onLog != nil {
		p.onLog(mtype, msg)
	} else {
		fmt.Println("[" + p.role + ":" + p.addr + "] " + msg)
	}
}

func (p *Peer) logErr(mtype MessageType, msg string) {
	if p.onLogErr != nil {
		p.onLogErr(mtype, msg)
	} else {
		fmt.Println("[" + p.role + ":" + p.addr + "] MSG_ERROR: " + msg)
	}
}

func (p *Peer) logMessage(mtype MessageType, code ErrorCode, sends bool) {
	if p.onLogMessage != nil {
		p.onLogMessage(mtype, code, sends)
	} else {
		direction := "received"
		if sends {
			direction = "sent"
		}
		fmt.Printf("[%s:%s] %s message: %s\n", p.role, p.addr, direction, mtype)
	}
}

func (p *Peer) logMessageError(mtype MessageType, code ErrorCode, msg string, sends bool) {
	if p.onLogMessageError != nil {
		p.onLogMessageError(mtype, code, msg, sends)
	} else {
		direction := "receiving"
		if sends {
			direction = "sending"
		}
		fmt.Printf("[%s:%s] Error %s message %s: %s\n", p.role, p.addr, direction, mtype, msg)
	}
}

func StartServer(port int, objectManager *object.ObjectManager) error {

	addr := net.JoinHostPort("", strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logs.GlobalLog(fmt.Sprintf("Server listening on port %d...", port))
	for {
		conn, err := ln.Accept()
		if err != nil {
			logs.GlobalError(fmt.Sprintf("Server failed to accept connection: %s", err))
			continue
		}

		addr := conn.RemoteAddr().String()

		p := NewPeer(conn, "server", objectManager, nil, nil, nil, nil, nil)

		p.onLog = func(mtype MessageType, msg string) { logs.ServerLog(mtype, msg, p.ID) }
		p.onLogErr = func(mtype MessageType, msg string) { logs.ServerError(mtype, msg, p.ID) }
		p.onLogMessage = func(mtype MessageType, code ErrorCode, sends bool) {
			logs.ServerMessage(mtype, code, sends, p.ID, p.addr)
		}
		p.onLogMessageError = func(mtype MessageType, code ErrorCode, msg string, sends bool) {
			logs.ServerMessageError(mtype, code, msg, sends, p.ID, p.addr)
		}
		p.onDisconnect = func() { logs.ServerLog(MSG_NONE, fmt.Sprintf("Client at %s disconnected", p.addr), p.ID) }

		p.onLog(messages.MSG_HELLO, fmt.Sprintf("Accepted connection from %s", addr))

		p.Greet()
	}
}

func StartClient(host string, port int, objectManager *object.ObjectManager, onClose func()) error {

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		// if onClose != nil {
		// 	onClose()
		// }
		return err
	}

	p := NewPeer(conn, "client", objectManager, nil, nil, nil, nil, nil)

	p.onLog = func(mtype MessageType, msg string) { logs.ClientLog(mtype, msg, p.ID) }
	p.onLogErr = func(mtype MessageType, msg string) { logs.ClientError(mtype, msg, p.ID) }
	p.onLogMessage = func(mtype MessageType, code ErrorCode, sends bool) {
		logs.ClientMessage(mtype, code, sends, p.ID, p.addr)
	}
	p.onLogMessageError = func(mtype MessageType, code ErrorCode, msg string, sends bool) {
		logs.ClientMessageError(mtype, code, msg, sends, p.ID, p.addr)
	}
	p.onDisconnect = func() { logs.ClientLog(MSG_NONE, fmt.Sprintf("Disconnected from server at %s", p.addr), p.ID) }

	p.onLog(MSG_NONE, fmt.Sprintf("Connected to server at %s:%d", host, port))

	p.Greet()

	return nil
}
