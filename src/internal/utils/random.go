package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomNonce() big.Int {
	nonceBytes := make([]byte, 32)

	rand.Read(nonceBytes)

	var nonce big.Int
	nonce.SetBytes(nonceBytes)

	return nonce
}
