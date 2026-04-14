package peer

import (
	"bufio"
	"fmt"
	"marabu/internal/protocol"
	"marabu/internal/storage"
	"marabu/internal/types"
	"marabu/internal/validation"

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
	Store             *storage.Store
	Validator         *validation.Validator
}

// NewPeer creates a new Peer instance for a given network connection.
// It initializes the peer's state and starts a goroutine
// to handle incoming messages from the connection.
func NewPeer(conn net.Conn,
	role string,
	Store *storage.Store) *Peer {

	addr := conn.RemoteAddr().String()
	p := &Peer{
		conn:      conn,
		addr:      addr,
		buffer:    make([]byte, 0),
		role:      role,
		Store:     Store,
		Validator: validation.NewValidator(Store),
		done:      make(chan struct{}),
	}

	connectedPeersMutex.Lock()
	connectedPeersCnt++
	connectedPeers[addr] = p
	p.ID = connectedPeersCnt
	connectedPeersMutex.Unlock()

	go p.initializeSocket()

	// Start a routine to check for unfindable objects every 2 seconds
	// go p.Routine(2*time.Second, func() { p.NotifyUnfindableObject() })

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

func CleanupPendingBlocks(om *storage.Store) {
	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {
		expiredBlocks := om.CheckPendingBlocks()

		for _, expired := range expiredBlocks {

			connectedPeersMutex.Lock()
			expiredPeer, exists := connectedPeers[expired.Peer]
			connectedPeersMutex.Unlock()

			if exists {
				expiredPeer.SendError(types.E_UNFINDABLE_OBJECT, "Failed to retrieve object from the network in time.")
				expiredPeer.log(types.MSG_ERROR, types.E_UNFINDABLE_OBJECT, fmt.Sprintf("Pending block with txid %s is unfindable. Notifying peer %s", expired.Txid, expired.Peer))
			}
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
	var msg types.Message
	msg, err := protocol.UnmarshalMessage(raw)

	if err != nil {
		p.errInfo("Invalid message: " + err.Error())
		p.SendError(types.E_INVALID_FORMAT, "Could not validate JSON message: "+err.Error())
		if !p.handshakeComplete {
			p.disconnect()
		}
		return
	}

	errCode := types.E_NONE
	if msg.MessageType() == types.MSG_ERROR {
		errCode = msg.(*protocol.Error).Name
	}
	p.logMessage(msg.MessageType(), errCode, recv)

	if !p.handshakeComplete && msg.MessageType() != types.MSG_HELLO {
		p.errMessage(msg.MessageType(), types.E_NONE, "Failed handshake.Expected hello message first", false)
		p.SendError(types.E_INVALID_HANDSHAKE, "Handshake not completed, expected hello message but received "+string(msg.MessageType()))
		p.disconnect()
		return
	}

	// Dispatch based on type
	switch m := msg.(type) {
	case *protocol.Hello:
		p.handleHello(m)
	case *protocol.Error:
		p.handleError(m)
	case *protocol.GetPeers:
		p.handleGetPeers()
	case *protocol.Peers:
		p.handlePeers(m)
	case *protocol.GetObject:
		p.handleGetObject(m)
	case *protocol.IHaveObject:
		p.handleIHaveObject(m)
	case *protocol.Object:
		p.handleObject(m)
	case *protocol.GetMempool:
		p.handleGetMempool()
	case *protocol.Mempool:
		p.handleMempool(m)
	case *protocol.GetChainTip:
		p.handleGetChainTip()
	case *protocol.ChainTip:
		p.handleChainTip(m)
	default:
		p.errInfo("Unknown message type")
		p.SendError(types.E_INVALID_FORMAT, "Unknown protocol message")
		p.disconnect()
	}
}

func StartServer(port int, Store *storage.Store) error {

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

		p := NewPeer(conn, "server", Store)

		p.logInfo(fmt.Sprintf("Accepted connection from %s", addr))

		p.Greet()
	}
}

func StartClient(host string, port int, Store *storage.Store) error {

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	p := NewPeer(conn, "client", Store)

	p.logInfo(fmt.Sprintf("Connected to server at %s:%d", host, port))

	p.Greet()

	return nil
}
