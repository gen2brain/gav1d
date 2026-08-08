package av1

import "math/bits"

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func clip[T integer](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clipU8[T ~int | ~int32 | ~int64](v T) T {
	return clip(v, 0, 255)
}

func applySign(v, s int32) int32 {
	if s < 0 {
		return -v
	}
	return v
}

func applySign64(v int32, s int64) int32 {
	if s < 0 {
		return -v
	}
	return v
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}

	return v
}

func ulog2(v uint32) int {
	return bits.Len32(v) - 1
}

func u64log2(v uint64) int {
	return bits.Len64(v) - 1
}

func invRecenter(r, v uint32) uint32 {
	switch {
	case v > r<<1:
		return v
	case v&1 == 0:
		return v>>1 + r
	default:
		return r - (v+1)>>1
	}
}
