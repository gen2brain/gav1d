package av1

type putBits struct {
	buf  []byte
	acc  uint64
	bits int
}

func (p *putBits) reset() {
	p.buf = p.buf[:0]
	p.acc = 0
	p.bits = 0
}

func (p *putBits) bits32(v uint32, n int) {
	if n == 0 {
		return
	}
	p.acc = p.acc<<uint(n) | uint64(v)&(1<<uint(n)-1)
	p.bits += n
	for p.bits >= 8 {
		p.bits -= 8
		p.buf = append(p.buf, byte(p.acc>>uint(p.bits)))
	}
}

func (p *putBits) bit(v uint32) {
	p.bits32(v, 1)
}

func (p *putBits) sbits(v int32, n int) {
	p.bits32(uint32(v)&(1<<uint(n)-1), n)
}

func (p *putBits) uleb128(v uint32) {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		p.bits32(uint32(b), 8)
		if v == 0 {
			return
		}
	}
}

func (p *putBits) uniform(v, max uint32) {
	w := ulog2(max) + 1
	m := uint32(1)<<w - max
	if v < m {
		p.bits32(v, int(w)-1)

		return
	}
	x := v + m
	p.bits32(x>>1, int(w)-1)
	p.bit(x & 1)
}

func (p *putBits) vlc(v uint32) {
	n := 0
	for v+1 >= uint32(1)<<uint(n+1) {
		n++
	}
	p.bits32(0, n)
	p.bit(1)
	p.bits32(v+1-uint32(1)<<uint(n), n)
}

func (p *putBits) bytealign() {
	if p.bits != 0 {
		p.bits32(0, 8-p.bits)
	}
}

func (p *putBits) bytes() []byte {
	p.bytealign()

	return p.buf
}
