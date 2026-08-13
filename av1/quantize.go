package av1

func quantShift(tx int) int {
	return max(0, int(txfmDimensions[tx].ctx)-2)
}

func quantOne(c int32, dq uint32, shift int) int32 {
	neg := c < 0
	if neg {
		c = -c
	}
	v := (uint32(c)<<uint(shift) + dq/2) / dq
	if v > 0xfffff {
		v = 0xfffff
	}
	if neg {
		return -int32(v)
	}

	return int32(v)
}

func quantBlock(levels []int32, coeff []int32, n int, dcDq, acDq uint32, shift int) int {
	nz := 0
	for i := range n {
		dq := acDq
		if i == 0 {
			dq = dcDq
		}
		v := quantOne(coeff[i], dq, shift)
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
