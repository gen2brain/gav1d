package av1

import "math/bits"

func quantShift(tx int) int {
	return max(0, int(txfmDimensions[tx].ctx)-2)
}

const quantRound = 6

type quantizer struct {
	recip uint64
	round uint32
}

func newQuantizer(dq uint32) quantizer {
	return quantizer{recip: 1<<48/uint64(dq) + 1, round: dq * quantRound / 16}
}

func quantOne(c int32, q *quantizer, shift int) int32 {
	neg := c < 0
	if neg {
		c = -c
	}
	hi, lo := bits.Mul64(uint64(uint32(c)<<uint(shift)+q.round), q.recip)
	v := uint32(hi<<16 | lo>>48)
	if v > 0xfffff {
		v = 0xfffff
	}
	if neg {
		return -int32(v)
	}

	return int32(v)
}

func quantBlock(levels []int32, coeff []int32, n int, dcDq, acDq uint32, shift int) int {
	dc, ac := newQuantizer(dcDq), newQuantizer(acDq)
	nz := 0
	for i := range n {
		q := &ac
		if i == 0 {
			q = &dc
		}
		v := quantOne(coeff[i], q, shift)
		levels[i] = v
		if v != 0 {
			nz = i + 1
		}
	}

	return nz
}

func dequantBlock(coeff []int32, levels []int32, n int, dcDq, acDq uint32, shift int, cfMax int32) {
	for i := range n {
		dq := acDq
		if i == 0 {
			dq = dcDq
		}
		v := levels[i]
		neg := v < 0
		if neg {
			v = -v
		}
		d := int32(uint32(v) * dq >> uint(shift))
		if d > cfMax {
			d = cfMax
		}
		if neg {
			coeff[i] = -d
		} else {
			coeff[i] = d
		}
	}
}
