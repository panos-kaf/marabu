package crypto

import (
	"fmt"
	"marabu/internal/messages"
	"math/big"

	"golang.org/x/crypto/blake2s"
)

const (
	TARGET_HEX = "00000000abc00000000000000000000000000000000000000000000000000000"
)

var TARGET = func() *big.Int {
	n := new(big.Int)
	n.SetString(TARGET_HEX, 16)
	return n
}

// Hash computes the BLAKE2s hash of the input data and returns it as a hexadecimal string.
func Hash(data []byte) (string, error) {
	hasher, err := blake2s.New256(nil)
	if err != nil {
		return "", err
	}
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// HashString is a convenience function that takes a string input, computes its BLAKE2s hash, and returns the hash as a hexadecimal string.
func HashString(s string) (string, error) {
	return Hash([]byte(s))
}

// HashBytes is a convenience function that takes a byte slice input, computes its BLAKE2s hash, and returns the hash as a hexadecimal string.
func HashBytes(b []byte) (string, error) {
	return Hash(b)
}

// HashObject takes an object, canonicalizes it to JSON, and then computes the BLAKE2s hash of the canonical JSON representation. It returns the hash as a hexadecimal string.
func HashObject(o messages.Object) (string, error) {
	raw, err := messages.Canonicalize(o)
	if err != nil {
		return "", err
	}
	hash, err := HashString(raw)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func VerifyPoW(blockid string) (bool, error) {

	hashInt := new(big.Int)
	_, ok := hashInt.SetString(blockid, 16)
	if !ok {
		return false, fmt.Errorf("Error parsing block ID as hex: %s", blockid)
	}

	return hashInt.Cmp(TARGET()) < 1, nil
}
