package types

import (
	"encoding/json"
	"fmt"
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
	maxBuStringLen = 1024

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

func (n *Nonce) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// if len(s) < 64 {
	// 	s = strings.Repeat("0", 64-len(s)) + s
	// }

	if len(s) > 64 {
		return fmt.Errorf("nonce exceeds maximum length of 64 characters, got %d", len(s))
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return fmt.Errorf("invalid hex character in nonce: %c", c)
		}
	}

	*n = Nonce(s)
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
		c := str[i]

		// testing: allow \t \n \r
		if (c < 32 || c > 126) && c != 10 && c != 13 && c != 9 {
			return fmt.Errorf("non-printable ascii char with hexcode: 0x%X", str[i])
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

	host, portStr, err := net.SplitHostPort(peer)
	if err != nil {
		// log.Printf("Rejected peer %s: invalid host:port format", peer)
		return PEER_INVALID
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		// log.Printf("Rejected peer %s: invalid port", peer)
		return PEER_INVALID
	}

	// Remove IPv6 brackets if present
	ipStr := host
	if strings.HasPrefix(ipStr, "[") && strings.HasSuffix(ipStr, "]") {
		ipStr = ipStr[1 : len(ipStr)-1]
	}

	ip := net.ParseIP(ipStr)

	// IP address validation
	if ip != nil {
		if ip.IsLoopback() ||
			ip.IsPrivate() ||
			ip.IsUnspecified() ||
			ip.IsMulticast() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() {

			// log.Printf("Rejected peer %s: invalid/private IP address", peer)
			return PEER_INVALID
		}

		return Peer(net.JoinHostPort(ip.String(), portStr))
	}

	// Domain validation (no DNS lookup)
	domainRegex := regexp.MustCompile(
		`^(?i)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`,
	)

	if domainRegex.MatchString(host) && !strings.EqualFold(host, "localhost") {
		return Peer(net.JoinHostPort(host, portStr))
	}

	// log.Printf("Rejected peer %s: invalid hostname or address", peer)
	return PEER_INVALID
}
