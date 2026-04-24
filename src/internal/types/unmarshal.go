package types

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(`^0\.10\.[0-9]+$`)

const (
	maxArrLen      = 1000
	maxStrLen      = 1000
	maxBuStringLen = 128

	// Student IDs
	maxBuStringsLen = 10
)

func (mt *MessageType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch MessageType(s) {
	case MSG_HELLO, MSG_ERROR, MSG_GETPEERS, MSG_PEERS, MSG_GETOBJECT, MSG_IHAVEOBJECT, MSG_OBJECT, MSG_GETMEMPOOL, MSG_MEMPOOL, MSG_GETCHAINTIP, MSG_CHAINTIP:
		*mt = MessageType(s)
		return nil
	default:
		return fmt.Errorf("invalid message type: '%s'", s)
	}
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}
	if !versionRegex.MatchString(s) {
		return fmt.Errorf("invalid version format: %s", s)
	}
	*v = Version(s)
	return nil
}

// Custom UnmarshalJSON for ErrorCode to enforce valid error codes
func (ec *ErrorCode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch ErrorCode(s) {
	case E_INTERNAL_ERROR, E_INVALID_FORMAT, E_UNKNOWN_OBJECT, E_UNFINDABLE_OBJECT, E_INVALID_HANDSHAKE, E_INVALID_TX_OUTPOINT, E_INVALID_TX_SIGNATURE, E_INVALID_TX_CONSERVATION, E_INVALID_BLOCK_COINBASE, E_INVALID_BLOCK_TIMESTAMP, E_INVALID_BLOCK_POW, E_INVALID_GENESIS:
		*ec = ErrorCode(s)
		return nil
	default:
		return fmt.Errorf("invalid error code: '%s'", s)
	}
}

// Custom UnmarshalJSON for ObjectType to enforce valid object types
func (ot *ObjectType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch ObjectType(s) {
	case OBJ_BLOCK, OBJ_TRANSACTION:
		*ot = ObjectType(s)
		return nil
	default:
		return fmt.Errorf("invalid object type: '%s'", s)
	}
}

func (p *Peer) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid peer format: %w", err)
	}

	sanitized := sanitizePeer(s)
	*p = sanitized

	return nil
}

// Custom UnmarshalJSON for Hash IDs to enforce length and hex format
func (h *HashID) UnmarshalJSON(data []byte) error {
	var s string

	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if len(s) != 64 {
		return fmt.Errorf("invalid hash ID: must be exactly 64 characters, got %d", len(s))
	}

	// Hexification - Hex strings must be in lower case.
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("invalid hash ID: must be hexadecimal, got invalid character '%c'", c)
		}
	}

	*h = HashID(s)
	return nil
}

// Custom UnmarshalJSON for Signatures to enforce length and hex format
func (sig *Signature) UnmarshalJSON(data []byte) error {
	var s string

	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if len(s) != 128 {
		return fmt.Errorf("invalid signature: must be exactly 128 characters, got %d", len(s))
	}

	// Hexification - Hex strings must be in lower case.
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("invalid signature: must be hexadecimal, got invalid character '%c'", c)
		}
	}

	*sig = Signature(s)
	return nil
}

func (o *TxOutput) UnmarshalJSON(data []byte) error {
	type Alias TxOutput
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("invalid TxOutput format: %w", err)
	}
	if o.Value == nil {
		return fmt.Errorf("missing value field in TxOutput")
	}
	return nil
}

func (c *CoinbaseTransaction) UnmarshalJSON(data []byte) error {
	type Alias CoinbaseTransaction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("invalid CoinbaseTransaction format: %w", err)
	}

	if len(c.Outputs) != 1 {
		return fmt.Errorf("coinbase transaction must have exactly one output, got %d", len(c.Outputs))
	}

	if c.Height == nil {
		return fmt.Errorf("missing height field in CoinbaseTransaction")
	}
	return nil
}

func (o *Outpoint) UnmarshalJSON(data []byte) error {
	type Alias Outpoint
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("invalid Outpoint format: %w", err)
	}
	if o.Index == nil {
		return fmt.Errorf("missing index field in Outpoint")
	}
	return nil
}

// Custom MarshalJSON for Picabu to properly output the number
func (b *Picabu) MarshalJSON() ([]byte, error) {
	numStr := (*big.Int)(b).String()
	return []byte(numStr), nil
}

func (b *Picabu) UnmarshalJSON(data []byte) error {
	var i big.Int
	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("invalid Picabu format: %w", err)
	}

	if i.Sign() < 0 {
		return fmt.Errorf("Picabu value must be non-negative, got %s", i.String())
	}

	*b = Picabu(i)
	return nil
}

func (n *BuInt) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("invalid integer format: %w", err)
	}

	if i < 0 {
		return fmt.Errorf("integer value must be non-negative, got %d", i)
	}

	*n = BuInt(i)
	return nil
}

func (s *BuString) UnmarshalJSON(data []byte) error {
	var str string

	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("invalid string format: %w", err)
	}

	for i := 0; i < len(str); i++ {
		if str[i] < 32 || str[i] > 126 {
			return fmt.Errorf("non-printable ascii char: %X", str[i])
		}
	}

	if len(str) > maxBuStringLen {
		return fmt.Errorf("string exceeds maximum length of %d characters, got %d", maxBuStringLen, len(str))
	}
	*s = BuString(str)
	return nil
}

// Generic helper for unmarshaling arrays with length checks
func UnmarshalArray[T any](data []byte, target *[]T, minLen int, maxLen int, fieldName string) error {
	var arr []T
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("invalid %s array format: %w", fieldName, err)
	}
	if len(arr) < minLen {
		return fmt.Errorf("%s array must have at least %d elements, got %d", fieldName, minLen, len(arr))
	}
	if len(arr) > maxLen {
		return fmt.Errorf("%s array exceeds maximum length of %d elements, got %d", fieldName, maxLen, len(arr))
	}
	*target = arr
	return nil
}

func (arr *BuInts) UnmarshalJSON(data []byte) error {
	return UnmarshalArray(data, (*[]BuInt)(arr), 0, maxArrLen, "BuInts")
}
func (arr *BuStrings) UnmarshalJSON(data []byte) error {
	return UnmarshalArray(data, (*[]BuString)(arr), 0, maxBuStringsLen, "BuStrings")
}
func (arr *HashIDs) UnmarshalJSON(data []byte) error {
	return UnmarshalArray(data, (*[]HashID)(arr), 0, maxArrLen, "HashIDs")
}
func (arr *TxInputs) UnmarshalJSON(data []byte) error {
	return UnmarshalArray(data, (*[]TxInput)(arr), 1, maxArrLen, "TxInputs")
}
func (arr *TxOutputs) UnmarshalJSON(data []byte) error {
	return UnmarshalArray(data, (*[]TxOutput)(arr), 0, maxArrLen, "TxOutputs")
}

func (arr *Peers) UnmarshalJSON(data []byte) error {
	var rawPeers []Peer
	if err := json.Unmarshal(data, &rawPeers); err != nil {
		return fmt.Errorf("invalid Peers array format: %w", err)
	}
	validPeers := make(Peers, 0, len(rawPeers))
	for _, peer := range rawPeers {
		if peer != PEER_INVALID {
			validPeers = append(validPeers, peer)
		}
	}
	*arr = validPeers
	return nil
}

// Validate and sanitize peer address, returning PEER_INVALID if invalid
func sanitizePeer(peer string) Peer {
	peer = strings.TrimSpace(peer)
	lastColon := strings.LastIndex(peer, ":")
	if lastColon == -1 {
		return PEER_INVALID
	}
	portStr := peer[lastColon+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return PEER_INVALID
	}
	host := peer[:lastColon]

	isv6 := false
	// Remove brackets for IPv6
	ipStr := host
	if strings.HasPrefix(ipStr, "[") && strings.HasSuffix(ipStr, "]") {
		ipStr = ipStr[1 : len(ipStr)-1]
		isv6 = true
	}
	ip := net.ParseIP(ipStr)

	if ip != nil {
		// IPv4
		if ip.To4() != nil && !isv6 {
			if strings.HasPrefix(ipStr, "127.") || strings.HasPrefix(ipStr, "0.") ||
				strings.HasPrefix(ipStr, "192.168.") || strings.HasPrefix(ipStr, "10.") {
				log.Printf("Rejected peer %s: IPv4 address is loopback or private", peer)
				return PEER_INVALID
			}
			octets := strings.Split(ipStr, ".")
			if len(octets) != 4 {
				log.Printf("Rejected peer %s: invalid IPv4 format", peer)
				return PEER_INVALID
			}
			if strings.HasPrefix(ipStr, "172.") {
				second, err := strconv.Atoi(octets[1])
				if err != nil || second < 16 || second > 31 {
					log.Printf("Rejected peer %s: invalid IPv4 octet", peer)
					return PEER_INVALID
				}
			}
			return Peer(peer)
		}
		// IPv6
		if ip.To16() != nil && ip.To4() == nil {
			if ipStr == "::1" || strings.HasPrefix(ipStr, "fe80:") || strings.HasPrefix(ipStr, "fc00:") {
				log.Printf("Rejected peer %s: IPv6 address is loopback or link-local", peer)
				return PEER_INVALID
			}
			return Peer(peer)
		}
	}

	// Domain check (strict format, no DNS lookup)
	if !isv6 {
		domain := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
		if domain.MatchString(host) && host != "localhost" {
			return Peer(peer)
		}
	}

	return PEER_INVALID
}
