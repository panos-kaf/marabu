package peer

import (
	"bufio"
	"fmt"
	"marabu/internal/messages"
	"marabu/internal/object"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
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
	done              chan struct{}
	role              string
	objectManager     *object.ObjectManager
}

// NewPeer creates a new Peer instance for a given network connection.
// It initializes the peer's state and starts a goroutine
// to handle incoming messages from the connection.
func NewPeer(conn net.Conn,
	role string,
	objectManager *object.ObjectManager) *Peer {

	addr := conn.RemoteAddr().String()
	p := &Peer{
		conn:          conn,
		addr:          addr,
		buffer:        make([]byte, 0),
		role:          role,
		objectManager: objectManager,
		done:          make(chan struct{}),
	}

	connectedPeersMutex.Lock()
	connectedPeersCnt++
	connectedPeers[addr] = p
	p.ID = connectedPeersCnt
	connectedPeersMutex.Unlock()

	go p.initializeSocket()

	// Start a routine to check for unfindable objects every 2 seconds
	go p.Routine(2*time.Second, func() { p.NotifyUnfindableObject() })

	return p
}

func (p *Peer) Routine(interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fn()
		case <-p.done:
			return
		}
	}
}

// initializeSocket starts a goroutine to read messages from the peer's connection.
// It continuously reads lines from the connection, and for each line, it calls handleMessage.
// On error it disconnects and removes the peer from the connectedPeers map.
func (p *Peer) initializeSocket() {
	reader := bufio.NewReader(p.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			p.disconnect()
			return
		}
		p.handleMessage(line)
	}
}

func (p *Peer) disconnect() {
	select {
	case <-p.done:
		// already closed, do nothing
		return
	default:
		close(p.done)
	}

	connectedPeersMutex.Lock()
	delete(connectedPeers, p.addr)
	connectedPeersMutex.Unlock()

	p.conn.Close()

	switch p.role {
	case "client":
		p.logInfo("Disconnected from server at " + p.addr)
	case "server":
		p.logInfo("Client at " + p.addr + " disconnected")
	default:
		p.logInfo("Peer at " + p.addr + " disconnected")
	}
}

// handleMessage processes incoming messages,
// ensuring they are valid and dispatching them
// to the appropriate handler based on their type.
func (p *Peer) handleMessage(raw string) {

	if len(strings.TrimSpace(raw)) == 0 {
		p.logInfo("Received empty message")
		return
	}

	// Unmarshal and validate message
	var msg Message
	msg, err := messages.UnmarshalMessage(raw)

	if err != nil {
		p.errInfo("Invalid message: " + err.Error())
		p.SendError(E_INVALID_FORMAT, "Could not validate JSON message: "+err.Error())
		if !p.handshakeComplete {
			p.disconnect()
		}
		return
	}

	errCode := E_NONE
	if msg.MessageType() == MSG_ERROR {
		errCode = msg.(*ErrorMessage).Name
	}
	p.logMessage(msg.MessageType(), errCode, recv)

	if !p.handshakeComplete && msg.MessageType() != messages.MSG_HELLO {
		p.errMessage(msg.MessageType(), E_NONE, "Failed handshake.Expected hello message first", false)
		p.SendError(messages.E_INVALID_HANDSHAKE, "Handshake not completed, expected hello message but received "+string(msg.MessageType()))
		p.disconnect()
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
		p.errInfo("Unknown message type")
		p.SendError(messages.E_INVALID_FORMAT, "Unknown protocol message")
		p.disconnect()
	}
}

func StartServer(port int, objectManager *object.ObjectManager) error {

	addr := net.JoinHostPort("", strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	globalLog(fmt.Sprintf("Server listening on port %d...", port))
	for {
		conn, err := ln.Accept()
		if err != nil {
			globalError(fmt.Sprintf("Server failed to accept connection: %s", err))
			continue
		}

		addr := conn.RemoteAddr().String()

		p := NewPeer(conn, "server", objectManager)

		p.logInfo(fmt.Sprintf("Accepted connection from %s", addr))

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

	p := NewPeer(conn, "client", objectManager)

	p.logInfo(fmt.Sprintf("Connected to server at %s:%d", host, port))

	p.Greet()

	return nil
}
