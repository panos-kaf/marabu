package types

import (
	"math/big"
)

type (
	Message interface {
		MessageType() MessageType
	}

	MessageType string
	ErrorCode   string
	Version     string
	Peer        string
	Peers       []Peer
	HashID      string // 32byte (64-character) hex string
	HashIDs     []HashID
	Nonce       string // 32byte 0-padded hex string
	Signature   string // 64byte (128-character) hex string

	Picabu big.Int

	BuInt     int
	BuString  string
	BuInts    []BuInt
	BuStrings []BuString
)

const (

	// Peer sanitization fail value
	PEER_INVALID = ""

	TARGET        = HashID("00000000abc00000000000000000000000000000000000000000000000000000")
	TARGET_STRING = "00000000abc00000000000000000000000000000000000000000000000000000"

	DUMMY_HASH = HashID("CAFEBABECAFEBABECAFEBABECAFEBABECAFEBABECAFEBABECAFEBABECAFEBABE")
)

var TARGET_BIGINT = func() *big.Int {
	n := new(big.Int)
	n.SetString(TARGET_STRING, 16)
	return n
}

// Empty struct evaluates to zero value
var ZERO_PICABU = Picabu{}

func NewPicabu(val uint64) Picabu {
	// create a new bigint, set it to the uint64, dereference it, and cast it!
	return Picabu(*new(big.Int).SetUint64(val))
}

// 50 x 10^12 picabu, must cast it to picabu
const BLOCK_REWARD_UINT64 uint64 = 50000000000000

func BlockReward() Picabu {
	return NewPicabu(BLOCK_REWARD_UINT64)
}

func BlockRewardBigInt() *big.Int {
	return new(big.Int).SetUint64(BLOCK_REWARD_UINT64)
}

func (p Picabu) Add(other Picabu) Picabu {
	sum := new(big.Int).Add((*big.Int)(&p), (*big.Int)(&other))
	return Picabu(*sum)
}
