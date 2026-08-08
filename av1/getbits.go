package av1

type getBits struct {
	state    uint64
	bitsLeft int
	err      bool
	index    int
	data     []byte
}

func (c *getBits) init(data []byte) {
	*c = getBits{data: data}
}

func (c *getBits) bit() uint32 {
	if c.bitsLeft == 0 {
		if c.index >= len(c.data) {
			c.err = true
		} else {
			state := uint32(c.data[c.index])
			c.index++
			c.bitsLeft = 7
			c.state = uint64(state) << 57
			return state >> 7
		}
	}

	state := c.state
	c.bitsLeft--
	c.state = state << 1

	return uint32(state >> 63)
}

func (c *getBits) refill(n int) {
	var state uint32
	for {
		if c.index >= len(c.data) {
			c.err = true
			if state != 0 {
				break
			}
			return
		}
		state = state<<8 | uint32(c.data[c.index])
		c.index++
		c.bitsLeft += 8
		if n <= c.bitsLeft {
			break
		}
	}
	c.state |= uint64(state) << (64 - c.bitsLeft)
}

func (c *getBits) bits(n int) uint32 {
	if uint32(n) > uint32(c.bitsLeft) {
		c.refill(n)
	}

	state := c.state
	c.bitsLeft -= n
	c.state = state << n

	return uint32(state >> (64 - n))
}

func (c *getBits) sbits(n int) int32 {
	if uint32(n) > uint32(c.bitsLeft) {
		c.refill(n)
	}

	state := c.state
	c.bitsLeft -= n
	c.state = state << n

	return int32(int64(state) >> (64 - n))
}

func (c *getBits) uleb128() uint32 {
	var val uint64
	var i, more uint32

	for {
		v := c.bits(8)
		more = v & 0x80
		val |= uint64(v&0x7f) << i
		i += 7
		if more == 0 || i >= 56 {
			break
		}
	}

	if val > 0xffffffff || more != 0 {
		c.err = true

		return 0
	}

	return uint32(val)
}

func (c *getBits) uniform(max uint32) uint32 {
	l := ulog2(max) + 1
	m := uint32(1)<<l - max
	v := c.bits(l - 1)
	if v < m {
		return v
	}

	return v<<1 - m + c.bit()
}

func (c *getBits) vlc() uint32 {
	if c.bit() != 0 {
		return 0
	}

	nBits := 0
	for {
		nBits++
		if nBits == 32 {
			return 0xffffffff
		}
		if c.bit() != 0 {
			break
		}
	}

	return uint32(1)<<nBits - 1 + c.bits(nBits)
}

func (c *getBits) subexpU(ref, n uint32) uint32 {
	var v uint32

	for i := 0; ; i++ {
		b := 3
		if i != 0 {
			b = 3 + i - 1
		}

		if n < v+3<<b {
			v += c.uniform(n - v + 1)

			break
		}

		if c.bit() == 0 {
			v += c.bits(b)

			break
		}

		v += 1 << b
	}

	if ref*2 <= n {
		return invRecenter(ref, v)
	}

	return n - invRecenter(n-ref, v)
}

func (c *getBits) subexp(ref int32, n uint32) int32 {
	return int32(c.subexpU(uint32(ref)+1<<n, 2<<n)) - 1<<n
}

func (c *getBits) bytealign() {
	c.bitsLeft = 0
	c.state = 0
}

func (c *getBits) pos() int {
	return c.index*8 - c.bitsLeft
}

func (c *getBits) bytes(n int) []byte {
	i := c.index
	c.index += n

	return c.data[i : i+n]
}

func (c *getBits) setRemaining(n int) bool {
	if c.index+n > len(c.data) {
		return false
	}
	c.data = c.data[:c.index+n]

	return true
}

func (c *getBits) remaining() int {
	return len(c.data) - c.index
}
