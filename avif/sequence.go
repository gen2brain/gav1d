package avif

import (
	"math"

	"github.com/gen2brain/gav1d/av1"
)

type track struct {
	id          uint32
	handler     string
	timescale   uint32
	samples     []extent
	timing      []timeRun
	sizes       []uint32
	offsets     []uint64
	chunks      []chunkRun
	sampleCount int
	sampleSize  uint32
	auxl        []uint32
	auxType     string

	duration   uint64
	hasEdits   bool
	repeating  bool
	segmentDur uint64
}

const indefiniteDuration = ^uint64(0)

// loopCount is zero for forever, -1 to show each frame once, otherwise the
// number of extra plays. ISO/IEC 23008-12 section 9.6.1.
func (t *track) loopCount() int {
	if !t.hasEdits {
		return 0
	}
	if !t.repeating {
		return -1
	}
	if t.duration == indefiniteDuration || t.duration == 0 || t.segmentDur == 0 {
		return 0
	}

	n := t.duration / t.segmentDur
	if t.duration%t.segmentDur != 0 {
		n++
	}
	if n <= 1 {
		return -1
	}
	if n-1 > math.MaxInt32 {
		return 0
	}

	return int(n - 1)
}

type movie struct {
	tracks []track
}

func parseMoov(b []byte) (*movie, error) {
	m := &movie{}

	err := eachBox(b, func(typ string, b []byte) error {
		if typ != "trak" {
			return nil
		}
		t, err := parseTrak(b)
		if err != nil {
			return err
		}
		m.tracks = append(m.tracks, t)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func parseTrak(b []byte) (track, error) {
	var t track

	err := eachBox(b, func(typ string, b []byte) error {
		switch typ {
		case "tkhd":
			r := &reader{b: b}
			v, _ := r.fullBox()
			if v == 1 {
				r.skip(16)
			} else {
				r.skip(8)
			}
			t.id = r.u32()
			r.skip(4)
			if v == 1 {
				t.duration = r.u64()
			} else {
				t.duration = uint64(r.u32())
				if t.duration == 1<<32-1 {
					t.duration = indefiniteDuration
				}
			}
			if r.err {
				return ErrInvalid
			}

		case "edts":
			t.hasEdits = true

			return eachBox(b, func(typ string, b []byte) error {
				if typ != "elst" {
					return nil
				}
				r := &reader{b: b}
				v, flags := r.fullBox()
				if flags&1 == 0 {
					return nil
				}
				t.repeating = true
				if r.u32() != 1 {
					return nil
				}
				if v == 1 {
					t.segmentDur = r.u64()
				} else {
					t.segmentDur = uint64(r.u32())
				}
				if r.err {
					return ErrInvalid
				}

				return nil
			})

		case "tref":
			return eachBox(b, func(typ string, b []byte) error {
				if typ != "auxl" {
					return nil
				}
				r := &reader{b: b}
				for r.remaining() >= 4 {
					t.auxl = append(t.auxl, r.u32())
				}

				return nil
			})

		case "mdia":
			return eachBox(b, func(typ string, b []byte) error {
				switch typ {
				case "mdhd":
					r := &reader{b: b}
					v, _ := r.fullBox()
					if v == 1 {
						r.skip(16)
					} else {
						r.skip(8)
					}
					t.timescale = r.u32()
					if r.err {
						return ErrInvalid
					}

				case "hdlr":
					r := &reader{b: b}
					r.fullBox()
					r.u32()
					t.handler = r.str4()

				case "minf":
					return eachBox(b, func(typ string, b []byte) error {
						if typ != "stbl" {
							return nil
						}

						return t.parseStbl(b)
					})
				}

				return nil
			})
		}

		return nil
	})

	return t, err
}

type chunkRun struct {
	firstChunk, perChunk uint32
}

type timeRun struct {
	count, delta uint32
}

// timeCursor reads compact timing runs once, without expanding sample counts.
type timeCursor struct {
	runs []timeRun
	used uint32
}

func (c *timeCursor) next() uint32 {
	for len(c.runs) != 0 && c.used == c.runs[0].count {
		c.runs = c.runs[1:]
		c.used = 0
	}
	if len(c.runs) == 0 {
		return 0
	}
	c.used++
	return c.runs[0].delta
}

func (t *track) parseStbl(b []byte) error {
	return eachBox(b, func(typ string, b []byte) error {
		r := &reader{b: b}

		switch typ {
		case "stts":
			r.fullBox()
			n := int(r.u32())
			if r.err || n < 0 || n > r.remaining()/8 {
				return ErrInvalid
			}
			for range n {
				count := r.u32()
				delta := r.u32()
				if r.err {
					return ErrInvalid
				}
				t.timing = append(t.timing, timeRun{count, delta})
			}

		case "stsz":
			r.fullBox()
			t.sampleSize = r.u32()
			n := int(r.u32())
			if r.err || n < 0 || n > 1<<24 {
				return ErrInvalid
			}
			t.sampleCount = n
			if t.sampleSize == 0 {
				if n > r.remaining()/4 {
					return ErrInvalid
				}
				t.sizes = make([]uint32, n)
				for i := range n {
					t.sizes[i] = r.u32()
				}
			}

		case "stsd":
			r.fullBox()
			r.u32()

			return eachBox(b[r.i:], func(typ string, b []byte) error {
				if len(b) < 78 {
					return nil
				}

				return eachBox(b[78:], func(typ string, b []byte) error {
					if typ == "auxi" {
						rr := &reader{b: b}
						rr.fullBox()
						t.auxType = rr.cstr()
					}

					return nil
				})
			})

		case "stsc":
			r.fullBox()
			n := int(r.u32())
			if r.err || n < 0 || n > r.remaining()/12 {
				return ErrInvalid
			}
			for range n {
				first := r.u32()
				per := r.u32()
				r.u32()
				if r.err {
					return ErrInvalid
				}
				if first == 0 || per == 0 ||
					(len(t.chunks) != 0 && first <= t.chunks[len(t.chunks)-1].firstChunk) {
					return ErrInvalid
				}
				t.chunks = append(t.chunks, chunkRun{first, per})
			}

		case "stco", "co64":
			r.fullBox()
			n := int(r.u32())
			entryBytes := 4
			if typ == "co64" {
				entryBytes = 8
			}
			if r.err || n < 0 || n > 1<<24 || n > r.remaining()/entryBytes {
				return ErrInvalid
			}
			t.offsets = make([]uint64, n)
			for i := range n {
				if typ == "stco" {
					t.offsets[i] = uint64(r.u32())
				} else {
					t.offsets[i] = r.u64()
				}
			}
		}

		if r.err {
			return ErrInvalid
		}

		return nil
	})
}

// layout resolves sample locations only for a track being decoded. Header
// reads retain compact tables and do not allocate from uniform sample counts.
func (t *track) layout(fileSize uint64) error {
	if t.samples != nil {
		return nil
	}
	if t.sampleCount <= 0 || uint64(t.sampleCount) > fileSize ||
		uint64(t.sampleCount)*uint64(t.sampleSize) > fileSize ||
		len(t.offsets) == 0 || len(t.chunks) == 0 || t.chunks[0].firstChunk != 1 {
		return ErrInvalid
	}
	if t.sampleSize == 0 && len(t.sizes) != t.sampleCount {
		return ErrInvalid
	}

	samples := make([]extent, 0, t.sampleCount)
	run := 0
	for c, off := range t.offsets {
		for run+1 < len(t.chunks) && uint64(t.chunks[run+1].firstChunk) <= uint64(c)+1 {
			run++
		}
		for range t.chunks[run].perChunk {
			if len(samples) == t.sampleCount {
				t.samples = samples
				return nil
			}
			n := t.sampleSize
			if n == 0 {
				n = t.sizes[len(samples)]
			}
			if n == 0 || off > fileSize || uint64(n) > fileSize-off {
				return ErrInvalid
			}
			samples = append(samples, extent{off: off, len: uint64(n)})
			off += uint64(n)
		}
	}
	if len(samples) != t.sampleCount {
		return ErrInvalid
	}
	t.samples = samples
	return nil
}

func (m *movie) pictTrack() *track {
	for i := range m.tracks {
		if m.tracks[i].handler == "pict" && m.tracks[i].sampleCount > 0 {
			return &m.tracks[i]
		}
	}

	return nil
}

func (m *movie) alphaTrack(id uint32) *track {
	for i := range m.tracks {
		t := &m.tracks[i]
		if t.handler != "auxv" || t.sampleCount == 0 || t.auxType != alphaURN {
			continue
		}
		for _, to := range t.auxl {
			if to == id {
				return t
			}
		}
	}

	return nil
}

// decodeSequence decodes an image sequence track.
func (f *file) decodeSequence(o Options) (*AVIF, error) {
	t := f.movie.pictTrack()
	if t == nil || t.timescale == 0 {
		return nil, nil
	}
	alphaTrack := f.movie.alphaTrack(t.id)

	colors, err := f.decodeTrack(t)
	if err != nil {
		return nil, err
	}
	var alphas []*av1.Picture
	if alphaTrack != nil {
		if alphas, err = f.decodeTrack(alphaTrack); err != nil {
			return nil, err
		}
	}

	out := &AVIF{LoopCount: t.loopCount()}
	if len(colors) > 0 {
		out.Color = f.colorInfo(f.primary(), colors[0])
	}
	// A track can show the same picture more than once, so nothing may go back
	// to the pool until every frame referencing it has been converted, and a
	// picture an image still aliases may never go back at all.
	var distinct []*av1.Picture
	seen := map[*av1.Picture]bool{}
	kept := map[*av1.Picture]bool{}
	note := func(p *av1.Picture, alias bool) {
		if p == nil {
			return
		}
		if !seen[p] {
			seen[p] = true
			distinct = append(distinct, p)
		}
		if alias {
			kept[p] = true
		}
	}

	timing := timeCursor{runs: t.timing}
	for i, pic := range colors {
		var alpha *av1.Picture
		if i < len(alphas) {
			alpha = alphas[i]
		}
		img, err := toImage(f, f.primary(), pic, alpha, o.ycbcrFor(f, f.primary()))
		if err != nil {
			return nil, err
		}
		alias := aliasesPicture(img)
		note(pic, alias)
		note(alpha, alias)
		if o.AutoRotate {
			if img, err = f.applyTransforms(img, f.primary()); err != nil {
				return nil, err
			}
		}
		out.Image = append(out.Image, img)

		d := float64(timing.next()) / float64(t.timescale)
		out.Delay = append(out.Delay, d)
	}
	for _, p := range distinct {
		if !kept[p] {
			p.Release()
		}
	}

	if len(out.Image) == 0 {
		return nil, nil
	}

	return out, nil
}

func (f *file) decodeTrack(t *track) ([]*av1.Picture, error) {
	if err := t.layout(f.src.size); err != nil {
		return nil, err
	}
	var d av1.Decoder
	d.FrameSizeLimit = f.frameSizeLimit
	var out []*av1.Picture

	for _, s := range t.samples {
		if s.len == 0 {
			return nil, ErrInvalid
		}
		b, err := f.src.at(s.off, s.len)
		if err != nil {
			return nil, err
		}
		pics, err := d.DecodeOBUs(b)
		if err != nil {
			return nil, err
		}
		out = append(out, pics...)
	}

	return out, nil
}
