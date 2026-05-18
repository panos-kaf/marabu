package crypto

import (
	"fmt"
	"marabu/internal/serialization"
	"marabu/internal/types"
	"math/big"

	"golang.org/x/crypto/blake2s"
)

// Hash computes the BLAKE2s hash of the input data and returns it as a HashID.
func Hash(data []byte) (types.HashID, error) {
	hasher, err := blake2s.New256(nil)
	if err != nil {
		return types.DUMMY_HASH, err
	}
	hasher.Write(data)
	// Cast the resulting hex string into a HashID type
	return types.HashID(fmt.Sprintf("%x", hasher.Sum(nil))), nil
}

// HashString is a convenience function that takes a string input.
func HashString(s string) (types.HashID, error) {
	return Hash([]byte(s))
}

// HashBytes is a convenience function that takes a byte slice input.
func HashBytes(b []byte) (types.HashID, error) {
	return Hash(b)
}

// HashObject canonicalizes an object to JSON and returns its BLAKE2s hash as a HashID.
func HashObject(o types.Object) (types.HashID, error) {
	raw, err := serialization.Canonicalize(o)
	if err != nil {
		return types.DUMMY_HASH, err
	}

	return HashString(raw)
}

func HashObjectBigInt(o types.Object) (*big.Int, error) {
	hashID, err := HashObject(o)
	if err != nil {
		return nil, err
	}

	hashInt := new(big.Int)
	// math/big requires a primitive string, so we safely cast the HashID down here
	_, ok := hashInt.SetString(string(hashID), 16)
	if !ok {
		return nil, fmt.Errorf("Error parsing hash as hex: %s", hashID)
	}

	return hashInt, nil
}

// GetObjectID safely retrieves the cached ID.
// If it hasn't been hashed yet, it calculates it on the fly and permanently caches it.
func GetObjectID(obj types.Object) (types.HashID, error) {

	if cached := obj.ObjID(); cached != nil {
		return *cached, nil
	}

	hashID, err := HashObject(obj)
	if err != nil {
		return types.DUMMY_HASH, err
	}

	switch o := obj.(type) {
	case *types.Block:
		o.CachedID = &hashID
	case *types.Transaction:
		o.CachedID = &hashID
	case *types.CoinbaseTransaction:
		o.CachedID = &hashID
	}

	return hashID, nil
}

func VerifyPoW(blockid types.HashID) (bool, error) {

	hashInt := new(big.Int)

	_, ok := hashInt.SetString(string(blockid), 16)
	if !ok {
		return false, fmt.Errorf("Error parsing block ID as hex: %s", blockid)
	}

	return hashInt.Cmp(types.TARGET_BIGINT()) < 1, nil
}
