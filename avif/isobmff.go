package avif

import (
	"errors"
	"io"
)

// ErrInvalid is returned when a file is not an AVIF, or is malformed past
// the point where anything can be decoded from it.
var ErrInvalid = errors.New("avif: invalid file")

// source is the file, addressed by range. b carries it when it is buffered.
type source struct {
	b    []byte
	r    io.ReaderAt
	size uint64
}

func memSource(b []byte) *source {
	return &source{b: b, size: uint64(len(b))}
}

// at returns n bytes at off, aliasing a buffered file, so it must not be written to.
func (s *source) at(off, n uint64) ([]byte, error) {
	if off > s.size || n > s.size-off {
		return nil, ErrInvalid
	}
	if s.r == nil {
		return s.b[off : off+n], nil
	}

	buf := make([]byte, n)
	_, err := io.ReadFull(io.NewSectionReader(s.r, int64(off), int64(n)), buf)
	switch {
	case err == nil:
		return buf, nil
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return nil, ErrInvalid
	}

	return nil, err
}

// eachBox reports where each top level payload lives without reading it.
func (s *source) eachBox(fn func(typ string, off, n uint64) error) error {
	for off := uint64(0); off+8 <= s.size; {
		b, err := s.at(off, 8)
		if err != nil {
			return err
		}
		r := &reader{b: b}
		size := uint64(r.u32())
		typ := r.str4()
		n := uint64(8)

		switch size {
		case 1:
			if off+16 > s.size {
				return ErrInvalid
			}
			b, err = s.at(off+8, 8)
			if err != nil {
				return err
			}
			rr := &reader{b: b}
			size = rr.u64()
			n = 16
			r.err = r.err || rr.err
		case 0:
			size = s.size - off
		}
		if r.err || size < n || size > s.size-off {
			return ErrInvalid
		}
		if typ != "uuid" {
			if err := fn(typ, off+n, size-n); err != nil {
				return err
			}
		}
		off += size
	}

	return nil
}

type reader struct {
	b   []byte
	i   int
	err bool
}

func (r *reader) remaining() int {
	if r.err {
		return 0
	}

	return len(r.b) - r.i
}

func (r *reader) fail() {
	r.err = true
	r.i = len(r.b)
}

func (r *reader) bytes(n int) []byte {
	if n < 0 || r.remaining() < n {
		r.fail()

		return nil
	}
	v := r.b[r.i : r.i+n]
	r.i += n

	return v
}

func (r *reader) skip(n int) {
	r.bytes(n)
}

func (r *reader) u8() uint8 {
	b := r.bytes(1)
	if b == nil {
		return 0
	}

	return b[0]
}

func (r *reader) uint(n int) uint64 {
	b := r.bytes(n)
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}

	return v
}

func (r *reader) u16() uint16 { return uint16(r.uint(2)) }
func (r *reader) u32() uint32 { return uint32(r.uint(4)) }
func (r *reader) u64() uint64 { return r.uint(8) }

func (r *reader) str4() string {
	b := r.bytes(4)
	if b == nil {
		return ""
	}

	return string(b)
}

func (r *reader) cstr() string {
	for j := r.i; j < len(r.b); j++ {
		if r.b[j] == 0 {
			s := string(r.b[r.i:j])
			r.i = j + 1

			return s
		}
	}
	r.fail()

	return ""
}

func (r *reader) fullBox() (uint8, uint32) {
	v := r.u32()

	return uint8(v >> 24), v & 0xffffff
}

// eachBox calls fn for every box in b.
func eachBox(b []byte, fn func(typ string, payload []byte) error) error {
	r := &reader{b: b}

	for r.remaining() >= 8 {
		start := r.i
		size := uint64(r.u32())
		typ := r.str4()
		hdr := 8

		switch size {
		case 1:
			size = r.u64()
			hdr = 16
		case 0:
			size = uint64(len(b) - start)
		}
		if r.err || size < uint64(hdr) || size > uint64(len(b)-start) {
			return ErrInvalid
		}

		payload := b[start+hdr : start+int(size)]
		if typ == "uuid" {
			r.i = start + int(size)

			continue
		}
		if err := fn(typ, payload); err != nil {
			return err
		}
		r.i = start + int(size)
	}

	if r.err {
		return ErrInvalid
	}

	return nil
}

type extent struct {
	off, len uint64
}

type item struct {
	id          uint32
	typ         string
	contentType string
	extents     []extent

	method     uint8
	baseOffset uint64

	props []itemProp

	unsupported bool
}

type itemProp struct {
	idx       int
	essential bool
}

type itemRef struct {
	typ  string
	from uint32
	to   []uint32
}

type property struct {
	typ string

	w, h  uint32
	av1C  av1Config
	pixi  []uint8
	auxC  string
	angle uint8
	axis  uint8
	clap  [8]uint32
	opIdx uint8
	layer uint16
	colr  *colorInfo
}

type av1Config struct {
	profile      uint8
	level        uint8
	tier         uint8
	highBitdepth bool
	twelveBit    bool
	monochrome   bool
	subX, subY   bool
	chromaPos    uint8
}

type colorInfo struct {
	icc       []byte
	primaries uint16
	transfer  uint16
	matrix    uint16
	fullRange bool
	hasNCLX   bool
}

type metaBox struct {
	primary uint32
	items   map[uint32]*item
	order   []uint32
	props   []property
	refs    []itemRef
	idat    []byte
}

func parseMeta(payload []byte) (*metaBox, error) {
	r := &reader{b: payload}
	r.fullBox()
	if r.err {
		return nil, ErrInvalid
	}

	m := &metaBox{items: map[uint32]*item{}}

	err := eachBox(payload[r.i:], func(typ string, b []byte) error {
		switch typ {
		case "pitm":
			rr := &reader{b: b}
			v, _ := rr.fullBox()
			if v == 0 {
				m.primary = uint32(rr.u16())
			} else {
				m.primary = rr.u32()
			}
			if rr.err {
				return ErrInvalid
			}

		case "iloc":
			return m.parseIloc(b)

		case "iinf":
			return m.parseIinf(b)

		case "iref":
			return m.parseIref(b)

		case "idat":
			m.idat = b

		case "iprp":
			return eachBox(b, func(typ string, b []byte) error {
				switch typ {
				case "ipco":
					return m.parseIpco(b)
				case "ipma":
					return m.parseIpma(b)
				}

				return nil
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	m.markUnsupported()

	return m, nil
}

// knownProps are the property types this reader implements. An item with an
// essential association to anything else must not be processed.
var knownProps = map[string]bool{
	"ispe": true, "av1C": true, "pixi": true, "auxC": true, "a1op": true,
	"irot": true, "imir": true, "clap": true, "colr": true, "lsel": true,
	"a1lx": true,
}

func (m *metaBox) markUnsupported() {
	for _, it := range m.items {
		for _, ip := range it.props {
			if !ip.essential {
				continue
			}
			if ip.idx >= len(m.props) || !knownProps[m.props[ip.idx].typ] {
				it.unsupported = true

				break
			}
		}
	}
}

func (m *metaBox) item(id uint32) *item {
	it := m.items[id]
	if it == nil {
		it = &item{id: id}
		m.items[id] = it
		m.order = append(m.order, id)
	}

	return it
}

func (m *metaBox) parseIloc(b []byte) error {
	r := &reader{b: b}
	v, _ := r.fullBox()

	sizes := r.u8()
	offSize, lenSize := int(sizes>>4), int(sizes&0xf)
	sizes = r.u8()
	baseSize, idxSize := int(sizes>>4), int(sizes&0xf)
	if v < 1 {
		idxSize = 0
	}
	for _, n := range []int{offSize, lenSize, baseSize, idxSize} {
		if n != 0 && n != 4 && n != 8 {
			return ErrInvalid
		}
	}

	count := int(r.u16())
	if v == 2 {
		r.i -= 2
		count = int(r.u32())
	}

	for range count {
		var id uint32
		if v < 2 {
			id = uint32(r.u16())
		} else {
			id = r.u32()
		}
		it := m.item(id)
		if v >= 1 {
			it.method = uint8(r.u16() & 0xf)
		}
		r.skip(2)
		it.baseOffset = r.uint(baseSize)

		n := int(r.u16())
		for range n {
			r.skip(idxSize)
			off := r.uint(offSize)
			l := r.uint(lenSize)
			if r.err {
				return ErrInvalid
			}
			it.extents = append(it.extents, extent{off: off, len: l})
		}
		if r.err {
			return ErrInvalid
		}
	}

	return nil
}

func (m *metaBox) parseIinf(b []byte) error {
	r := &reader{b: b}
	v, _ := r.fullBox()
	if v == 0 {
		r.u16()
	} else {
		r.u32()
	}
	if r.err {
		return ErrInvalid
	}

	return eachBox(b[r.i:], func(typ string, b []byte) error {
		if typ != "infe" {
			return nil
		}
		rr := &reader{b: b}
		v, _ := rr.fullBox()
		if v < 2 {
			return nil
		}

		var id uint32
		if v == 2 {
			id = uint32(rr.u16())
		} else {
			id = rr.u32()
		}
		rr.skip(2)
		itemType := rr.str4()
		if rr.err {
			return ErrInvalid
		}

		it := m.item(id)
		it.typ = itemType

		if itemType == "mime" {
			rr.cstr()
			if ct := rr.cstr(); !rr.err {
				it.contentType = ct
			}
		}

		return nil
	})
}

func (m *metaBox) parseIref(b []byte) error {
	r := &reader{b: b}
	v, _ := r.fullBox()
	if r.err {
		return ErrInvalid
	}

	return eachBox(b[r.i:], func(typ string, b []byte) error {
		rr := &reader{b: b}
		ref := itemRef{typ: typ}
		if v == 0 {
			ref.from = uint32(rr.u16())
		} else {
			ref.from = rr.u32()
		}
		n := int(rr.u16())
		for range n {
			var to uint32
			if v == 0 {
				to = uint32(rr.u16())
			} else {
				to = rr.u32()
			}
			ref.to = append(ref.to, to)
		}
		if rr.err {
			return ErrInvalid
		}
		m.refs = append(m.refs, ref)

		return nil
	})
}

func (m *metaBox) parseIpco(b []byte) error {
	return eachBox(b, func(typ string, b []byte) error {
		p := property{typ: typ}
		r := &reader{b: b}

		switch typ {
		case "ispe":
			r.fullBox()
			p.w = r.u32()
			p.h = r.u32()

		case "av1C":
			marker := r.u8()
			if marker>>7 != 1 || marker&0x7f != 1 {
				return ErrInvalid
			}
			v := r.u8()
			p.av1C.profile = v >> 5
			p.av1C.level = v & 0x1f
			v = r.u8()
			p.av1C.tier = v >> 7
			p.av1C.highBitdepth = v>>6&1 != 0
			p.av1C.twelveBit = v>>5&1 != 0
			p.av1C.monochrome = v>>4&1 != 0
			p.av1C.subX = v>>3&1 != 0
			p.av1C.subY = v>>2&1 != 0
			p.av1C.chromaPos = v & 3

		case "pixi":
			r.fullBox()
			n := int(r.u8())
			for range n {
				p.pixi = append(p.pixi, r.u8())
			}

		case "auxC":
			r.fullBox()
			p.auxC = r.cstr()

		case "a1op":
			p.opIdx = r.u8()

		case "lsel":
			p.layer = r.u16()

		case "irot":
			p.angle = r.u8() & 3

		case "imir":
			p.axis = r.u8() & 1

		case "clap":
			for i := range 8 {
				p.clap[i] = r.u32()
			}

		case "colr":
			switch r.str4() {
			case "nclx":
				p.colr = &colorInfo{hasNCLX: true}
				p.colr.primaries = r.u16()
				p.colr.transfer = r.u16()
				p.colr.matrix = r.u16()
				p.colr.fullRange = r.u8()>>7 != 0
			case "rICC", "prof":
				p.colr = &colorInfo{icc: b[4:]}
			}
		}

		if r.err {
			return ErrInvalid
		}
		m.props = append(m.props, p)

		return nil
	})
}

func (m *metaBox) parseIpma(b []byte) error {
	r := &reader{b: b}
	v, flags := r.fullBox()
	count := int(r.u32())
	if r.err {
		return ErrInvalid
	}

	for range count {
		var id uint32
		if v < 1 {
			id = uint32(r.u16())
		} else {
			id = r.u32()
		}
		it := m.item(id)

		n := int(r.u8())
		for range n {
			var idx int
			var essential bool
			if flags&1 != 0 {
				v := r.u16()
				idx, essential = int(v&0x7fff), v&0x8000 != 0
			} else {
				v := r.u8()
				idx, essential = int(v&0x7f), v&0x80 != 0
			}
			if r.err {
				return ErrInvalid
			}
			if idx == 0 {
				continue
			}
			it.props = append(it.props, itemProp{idx: idx - 1, essential: essential})
		}
		if r.err {
			return ErrInvalid
		}
	}

	return nil
}

// data gathers an item's extents out of the file or, for construction method 1, out of idat.
func (m *metaBox) data(it *item, file *source) ([]byte, error) {
	src := file
	if it.method == 1 {
		src = memSource(m.idat)
	} else if it.method != 0 {
		return nil, ErrInvalid
	}

	span := func(e extent) (uint64, uint64, error) {
		off, n := it.baseOffset+e.off, e.len
		if off < it.baseOffset || off > src.size {
			return 0, 0, ErrInvalid
		}
		if n == 0 {
			n = src.size - off
		}

		return off, n, nil
	}

	if len(it.extents) == 1 {
		off, n, err := span(it.extents[0])
		if err != nil {
			return nil, err
		}

		return src.at(off, n)
	}

	var out []byte
	for _, e := range it.extents {
		off, n, err := span(e)
		if err != nil {
			return nil, err
		}
		if uint64(len(out))+n > src.size {
			return nil, ErrInvalid
		}
		b, err := src.at(off, n)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	if out == nil {
		return nil, ErrInvalid
	}

	return out, nil
}

func (m *metaBox) prop(it *item, typ string) *property {
	if it == nil {
		return nil
	}
	for _, ip := range it.props {
		if ip.idx < len(m.props) && m.props[ip.idx].typ == typ {
			return &m.props[ip.idx]
		}
	}

	return nil
}

func (m *metaBox) refsTo(typ string, from uint32) []uint32 {
	for _, r := range m.refs {
		if r.typ == typ && r.from == from {
			return r.to
		}
	}

	return nil
}
