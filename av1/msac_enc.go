package av1

type msacEncoder struct {
	low            uint64
	rng            uint32
	cnt            int
	precarry       []uint16
	allowUpdateCdf bool
	count          bool
	bits           float64
}

func (e *msacEncoder) init(disableCdfUpdate bool) {
	e.low = 0
	e.rng = 0x8000
	e.cnt = -9
	e.precarry = e.precarry[:0]
	e.allowUpdateCdf = !disableCdfUpdate
}

func (e *msacEncoder) renorm(lowIncr, rng uint32) {
	low := uint64(e.low) + uint64(lowIncr)
	c := e.cnt
	d := int(15 ^ ulog2(rng))
	s := c + d

	if s >= 0 {
		c += 16
		m := uint64(1)<<uint(c) - 1
		if s >= 8 {
			e.precarry = append(e.precarry, uint16(low>>uint(c)))
			low &= m
			c -= 8
			m >>= 8
		}
		e.precarry = append(e.precarry, uint16(low>>uint(c)))
		s = c + d - 24
		low &= m
	}

	e.low = low << uint(d)
	e.rng = rng << uint(d)
	e.cnt = s
}

func (e *msacEncoder) boolEqui(bit uint32) {
	if e.count {
		e.bits++

		return
	}
	r := e.rng
	v := (r>>8)<<7 + ecMinProb
	if bit != 0 {
		e.renorm(r-v, v)

		return
	}
	e.renorm(0, r-v)
}

func (e *msacEncoder) boolF(bit, f uint32) {
	if e.count {
		e.bits += boolFBits(f, bit)

		return
	}
	r := e.rng
	v := (r>>8)*(f>>ecProbShift)>>(7-ecProbShift) + ecMinProb
	if bit != 0 {
		e.renorm(r-v, v)

		return
	}
	e.renorm(0, r-v)
}

func (e *msacEncoder) boolAdapt(cdf []uint16, bit uint32) {
	e.boolF(bit, uint32(cdf[0]))
	if e.count {
		return
	}
	if e.allowUpdateCdf {
		cdfUpdateBool(cdf, bit)
	}
}

func (e *msacEncoder) symbolAdapt(cdf []uint16, nSymbols int, val uint32) {
	if e.count {
		e.bits += cdfBits(cdf, int(val))

		return
	}
	r := e.rng >> 8
	n := uint32(nSymbols)

	u := e.rng
	if val > 0 {
		u = r*uint32(cdf[val-1]>>ecProbShift)>>(7-ecProbShift) + ecMinProb*(n-val+1)
	}
	v := r*uint32(cdf[val]>>ecProbShift)>>(7-ecProbShift) + ecMinProb*(n-val)

	e.renorm(e.rng-u, u-v)

	if e.allowUpdateCdf {
		cdfUpdate(cdf, nSymbols, val)
	}
}

func (e *msacEncoder) bools(v uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		e.boolEqui(v >> uint(i) & 1)
	}
}

func (e *msacEncoder) literal(v uint32, n int) {
	e.bools(v, n)
}

func (e *msacEncoder) uniform(v, n uint32) {
	l := ulog2(n) + 1
	m := uint32(1)<<l - n
	if v < m {
		e.bools(v, int(l-1))

		return
	}
	w := v + m
	e.bools(w>>1, int(l-1))
	e.boolEqui(w & 1)
}

func (e *msacEncoder) done() []byte {
	l := e.low
	c := e.cnt
	s := 10 + c
	m := uint64(0x3fff)
	x := (l+m)&^m | (m + 1)

	if s > 0 {
		n := uint64(1)<<uint(c+16) - 1
		for {
			e.precarry = append(e.precarry, uint16(x>>uint(c+16)))
			x &= n
			s -= 8
			c -= 8
			n >>= 8
			if s <= 0 {
				break
			}
		}
	}

	out := make([]byte, len(e.precarry))
	carry := uint32(0)
	for i := len(e.precarry) - 1; i >= 0; i-- {
		carry += uint32(e.precarry[i])
		out[i] = byte(carry)
		carry >>= 8
	}

	return out
}
