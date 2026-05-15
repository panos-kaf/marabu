package peer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"marabu/internal/core"
	"marabu/internal/protocol"
	"marabu/internal/types"

	"net"
	"strconv"
	"strings"
	"time"
)

const (
	inbound  = false
	outbound = true
)

type Peer struct {
	conn              net.Conn
	agent             string
	addr              string
	host              string
	ID                int
	buffer            []byte
	handshakeComplete bool
	done              chan struct{}
	origin            types.Origin
	isPersistent      bool
	Manager           *core.Manager
	sentChainTip      bool
}

// NewPeer creates a new Peer instance for a given network connection.
// It initializes the peer's state and starts a goroutine
// to handle incoming messages from the connection.
func NewPeer(conn net.Conn,
	origin types.Origin,
	isPersistent bool,
	Manager *core.Manager) (*Peer, error) {

	addr := conn.RemoteAddr().String()
	p := &Peer{
		conn:         conn,
		addr:         addr,
		buffer:       make([]byte, 0),
		origin:       origin,
		Manager:      Manager,
		isPersistent: isPersistent,
		done:         make(chan struct{}),
		sentChainTip: false,
	}

	p.host, _, _ = net.SplitHostPort(addr)

	err := ConnManager.Add(p)
	if err != nil {
		p.conn.Close()
		return nil, err
	}

	go p.initializeSocket()

	return p, nil
}

// Name returns the peer's agent name if available, otherwise it falls back to the address.
func (p *Peer) Name() string {
	if p.agent != "" && p.agent != "unknown" {
		return p.agent
	}
	return p.addr
}

// Getters
func (p *Peer) Addr() string         { return p.addr }
func (p *Peer) Agent() string        { return p.agent }
func (p *Peer) Origin() types.Origin { return p.origin }
func (p *Peer) IsPersistent() bool   { return p.isPersistent }

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

// NotifyPeerUnfindable is the callback we will give to the Manager.
// It looks up the peer and sends the specific network error.
func NotifyPeerUnfindable(peerAddr string, txid types.HashID) {

	expiredPeer, exists := ConnManager.Exists(peerAddr)

	if exists {
		expiredPeer.SendError(types.E_UNFINDABLE_OBJECT, "Failed to retrieve object from the network in time.")
		expiredPeer.log(types.MSG_ERROR, types.E_UNFINDABLE_OBJECT, fmt.Sprintf("Pending block with txid %s is unfindable. Notifying peer %s", txid, peerAddr))
	}
}

// initializeSocket starts a goroutine to read messages from the peer's connection.
// It continuously reads lines from the connection, and for each line, it calls handleMessage.
// On error it disconnects and removes the peer from the connManager peers map.
// func (p *Peer) initializeSocket() {
// 	reader := bufio.NewReader(p.conn)
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			p.Disconnect()
// 			return
// 		}
// 		p.handleMessage(line)
// 	}
// }

func (p *Peer) initializeSocket() {
	decoder := json.NewDecoder(p.conn)
	for {
		var rawMsg json.RawMessage

		if err := decoder.Decode(&rawMsg); err != nil {

			if !errors.Is(err, io.EOF) {
				var netErr net.Error
				if errors.As(err, &netErr) {
					p.errInfo(fmt.Sprintf("Network error with peer %s: %v", p.Name(), err.Error()))

				} else {
					p.errInfo(fmt.Sprintf("%s sent invalid JSON or corrupted stream: %v", p.Name(), err.Error()))
				}
			}

			p.Disconnect()
			break
		}

		p.handleMessage(string(rawMsg))
	}
}

func (p *Peer) Disconnect() {
	select {
	case <-p.done:
		// already closed, do nothing
		return
	default:
		close(p.done)
	}

	ConnManager.Remove(p)

	p.conn.Close()

	p.logInfo(p.Name() + " disconnected")

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
			p.Disconnect()
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
		p.Disconnect()
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
		p.Disconnect()
	}
}

func StartServer(Manager *core.Manager) error {

	port := Manager.Port()

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
		host, _, _ := net.SplitHostPort(addr)

		if ConnManager.IsBanned(host) {
			globalLog(fmt.Sprintf("Rejected connection from banned IP: %s", host))
			conn.Close()
			continue
		}

		p, err := NewPeer(conn, types.Inbound, false, Manager)

		if err != nil {
			return fmt.Errorf("Error accepting new connection: %s", err.Error())
		}

		p.logInfo(fmt.Sprintf("Accepted connection from %s", addr))

		p.Greet()
	}
}

func StartClient(host string, port int, isPersistent bool, Manager *core.Manager) error {

	if ConnManager.IsBanned(host) {
		return fmt.Errorf("peer IP %s is banned, aborting dial", host)
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}

	p, err := NewPeer(conn, types.Outbound, isPersistent, Manager)

	if err != nil {
		return fmt.Errorf("Error starting client: %w", err)
	}

	p.logInfo(fmt.Sprintf("Connected to peer at %s:%d", host, port))

	p.Greet()

	return nil
}
