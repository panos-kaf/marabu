package utils

const PicabuPerBu = 1_000_000_000_000 // 1 trillion

// BuToPicabu converts a fractional Bu value to whole picabus
func BuToPicabu(bu float64) uint64 {
	return uint64(bu * PicabuPerBu)
}

// PicabuToBu converts whole picabus back to a readable fractional Bu
func PicabuToBu(picabus uint64) float64 {
	return float64(picabus) / float64(PicabuPerBu)
}
