package utils

import (
	"math/big"

	"marabu/internal/types"
)

const PicabuPerBu = 1_000_000_000_000 // 10^12

// BuToPicabu converts a fractional Bu value to a types.Picabu
func BuToPicabu(bu float64) types.Picabu {
	buFloat := big.NewFloat(bu)
	multiplier := big.NewFloat(PicabuPerBu)

	picabuFloat := new(big.Float).Mul(buFloat, multiplier)

	// Convert the big.Float to a big.Int
	picabuInt := new(big.Int)
	picabuFloat.Int(picabuInt)

	return types.Picabu(*picabuInt)
}

// PicabuToBu converts a types.Picabu back to a readable fractional Bu
func PicabuToBu(p types.Picabu) float64 {
	// Cast the wrapper back to the underlying big.Int pointer
	pInt := (*big.Int)(&p)

	// Convert the big.Int to a big.Float for safe division
	pFloat := new(big.Float).SetInt(pInt)
	divider := big.NewFloat(PicabuPerBu)

	// Divide by 10^12 to get the Bu value
	buFloat := new(big.Float).Quo(pFloat, divider)

	// Extract the native float64 value
	bu, _ := buFloat.Float64()
	return bu
}
