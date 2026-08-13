package av1

const (
	ecProbShift = 6
	ecMinProb   = 4
	ecWinSize   = 64
)

type msacContext struct {
	data           []byte
	pos            int
	dif            uint64
	rng            uint32
	cnt            int
	allowUpdateCdf bool
}

func (s *msacContext) init(data []byte, disableCdfUpdate bool) {
	s.data = data
	s.pos = 0
	s.dif = 0
	s.rng = 0x8000
	s.cnt = -15
	s.allowUpdateCdf = !disableCdfUpdate
	s.refill()
}

func (s *msacContext) refill() {
	c := ecWinSize - s.cnt - 24
	dif := s.dif
	for {
		if s.pos >= len(s.data) {
			dif |= ^(^uint64(0xff) << c)

			break
		}
		dif |= uint64(s.data[s.pos]^0xff) << c
		s.pos++
		c -= 8
		if c < 0 {
			break
		}
	}
	s.dif = dif
	s.cnt = ecWinSize - c - 24
}

func (s *msacContext) norm(dif uint64, rng uint32) {
	d := 15 ^ ulog2(rng)
	cnt := s.cnt
	s.dif = dif << d
	s.rng = rng << d
	s.cnt = cnt - d
	if uint32(cnt) < uint32(d) {
		s.refill()
	}
}

func (s *msacContext) boolEquiGo() uint32 {
	r := s.rng
	dif := s.dif
	v := (r>>8)<<7 + ecMinProb
	vw := uint64(v) << (ecWinSize - 16)
	ret := uint32(0)
	if dif >= vw {
		ret = 1
	}
	dif -= uint64(ret) * vw
	v += ret * (r - 2*v)
	s.norm(dif, v)

	return ret ^ 1
}

func (s *msacContext) boolFGo(f uint32) uint32 {
	r := s.rng
	dif := s.dif
	v := (r>>8)*(f>>ecProbShift)>>(7-ecProbShift) + ecMinProb
	vw := uint64(v) << (ecWinSize - 16)
	ret := uint32(0)
	if dif >= vw {
		ret = 1
	}
	dif -= uint64(ret) * vw
	v += ret * (r - 2*v)
	s.norm(dif, v)

	return ret ^ 1
}

func (s *msacContext) symbolAdaptGo(cdf []uint16, nSymbols int) uint32 {
	c := uint32(s.dif >> (ecWinSize - 16))
	r := s.rng >> 8
	u, v, val := uint32(0), s.rng, ^uint32(0)

	for {
		val++
		u = v
		v = r * uint32(cdf[val]>>ecProbShift)
		v >>= 7 - ecProbShift
		v += ecMinProb * (uint32(nSymbols) - val)
		if c >= v {
			break
		}
	}

	s.norm(s.dif-uint64(v)<<(ecWinSize-16), u-v)

	if s.allowUpdateCdf {
		cdfUpdate(cdf, nSymbols, val)
	}

	return val
}

func (s *msacContext) boolAdaptGo(cdf []uint16) uint32 {
	bit := s.boolFGo(uint32(cdf[0]))

	if s.allowUpdateCdf {
		cdfUpdateBool(cdf, bit)
	}

	return bit
}

func (s *msacContext) hiTok(cdf *[4]uint16) uint32 {
	tokBr := s.symbolAdapt4(cdf, 3)
	tok := 3 + tokBr
	if tokBr == 3 {
		tokBr = s.symbolAdapt4(cdf, 3)
		tok = 6 + tokBr
		if tokBr == 3 {
			tokBr = s.symbolAdapt4(cdf, 3)
			tok = 9 + tokBr
			if tokBr == 3 {
				tok = 12 + s.symbolAdapt4(cdf, 3)
			}
		}
	}

	return tok
}

func (s *msacContext) bools(n int) uint32 {
	v := uint32(0)
	for ; n > 0; n-- {
		v = v<<1 | s.boolEqui()
	}

	return v
}

func (s *msacContext) uniform(n uint32) uint32 {
	l := ulog2(n) + 1
	m := uint32(1)<<l - n
	v := s.bools(l - 1)
	if v < m {
		return v
	}

	return v<<1 - m + s.boolEqui()
}

func (s *msacContext) subexp(ref, n int32, k uint32) int32 {
	a := uint32(0)
	if s.boolEqui() != 0 {
		if s.boolEqui() != 0 {
			k += s.boolEqui() + 1
		}
		a = 1 << k
	}
	v := s.bools(int(k)) + a
	if ref*2 <= n {
		return int32(invRecenter(uint32(ref), v))
	}

	return n - 1 - int32(invRecenter(uint32(n-1-ref), v))
}

func cdfUpdate(cdf []uint16, nSymbols int, val uint32) {
	count := uint32(cdf[nSymbols])
	rate := 4 + count>>4
	if nSymbols > 2 {
		rate++
	}
	i := uint32(0)
	for ; i < val; i++ {
		cdf[i] += (32768 - cdf[i]) >> rate
	}
	for ; i < uint32(nSymbols); i++ {
		cdf[i] -= cdf[i] >> rate
	}
	if count < 32 {
		count++
	}
	cdf[nSymbols] = uint16(count)
}

func cdfUpdateBool(cdf []uint16, bit uint32) {
	count := uint32(cdf[1])
	rate := 4 + count>>4
	if bit != 0 {
		cdf[0] += (32768 - cdf[0]) >> rate
	} else {
		cdf[0] -= cdf[0] >> rate
	}
	if count < 32 {
		count++
	}
	cdf[1] = uint16(count)
}
