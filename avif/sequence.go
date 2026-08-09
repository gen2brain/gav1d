package avif

import "github.com/gen2brain/gav1d/av1"

type track struct {
	id        uint32
	handler   string
	timescale uint32
	samples   []extent
	deltas    []uint32
	auxl      []uint32
	auxType   string
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
			if r.err {
				return errInvalid
			}

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
						return errInvalid
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

func (t *track) parseStbl(b []byte) error {
	var sizes []uint32
	var offsets []uint64
	var runs []chunkRun

	err := eachBox(b, func(typ string, b []byte) error {
		r := &reader{b: b}

		switch typ {
		case "stts":
			r.fullBox()
			n := int(r.u32())
			for range n {
				count := r.u32()
				delta := r.u32()
				if r.err || count > 1<<24 {
					return errInvalid
				}
				for range int(count) {
					t.deltas = append(t.deltas, delta)
				}
			}

		case "stsz":
			r.fullBox()
			uniform := r.u32()
			n := int(r.u32())
			if r.err || n < 0 || n > 1<<24 {
				return errInvalid
			}
			sizes = make([]uint32, n)
			for i := range n {
				if uniform != 0 {
					sizes[i] = uniform
				} else {
					sizes[i] = r.u32()
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
			for range n {
				first := r.u32()
				per := r.u32()
				r.u32()
				if r.err {
					return errInvalid
				}
				runs = append(runs, chunkRun{first, per})
			}

		case "stco", "co64":
			r.fullBox()
			n := int(r.u32())
			if r.err || n < 0 || n > 1<<24 {
				return errInvalid
			}
			offsets = make([]uint64, n)
			for i := range n {
				if typ == "stco" {
					offsets[i] = uint64(r.u32())
				} else {
					offsets[i] = r.u64()
				}
			}
		}

		if r.err {
			return errInvalid
		}

		return nil
	})
	if err != nil {
		return err
	}

	t.samples = layoutSamples(sizes, offsets, runs)

	return nil
}

// layoutSamples walks the chunk table to give every sample a file offset.
func layoutSamples(sizes []uint32, offsets []uint64, runs []chunkRun) []extent {
	if len(sizes) == 0 || len(offsets) == 0 || len(runs) == 0 {
		return nil
	}

	out := make([]extent, 0, len(sizes))
	s := 0

	for c := range offsets {
		var per uint32
		for _, run := range runs {
			if uint32(c+1) < run.firstChunk {
				break
			}
			per = run.perChunk
		}

		off := offsets[c]
		for range int(per) {
			if s >= len(sizes) {
				return out
			}
			out = append(out, extent{off: off, len: uint64(sizes[s])})
			off += uint64(sizes[s])
			s++
		}
	}

	return out
}

func (m *movie) pictTrack() *track {
	for i := range m.tracks {
		if m.tracks[i].handler == "pict" && len(m.tracks[i].samples) > 0 {
			return &m.tracks[i]
		}
	}

	return nil
}

func (m *movie) alphaTrack(id uint32) *track {
	for i := range m.tracks {
		t := &m.tracks[i]
		if t.handler != "auxv" || len(t.samples) == 0 || t.auxType != alphaURN {
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

	out := &AVIF{}
	if len(colors) > 0 {
		out.Color = f.colorInfo(f.primary(), colors[0])
	}
	for i, pic := range colors {
		var alpha *av1.Picture
		if i < len(alphas) {
			alpha = alphas[i]
		}
		img, err := toImage(f, f.primary(), pic, alpha, o.ycbcrFor(f, f.primary()))
		if err != nil {
			return nil, err
		}
		if o.AutoRotate {
			if img, err = f.applyTransforms(img, f.primary()); err != nil {
				return nil, err
			}
		}
		out.Image = append(out.Image, img)

		d := 0.0
		if i < len(t.deltas) {
			d = float64(t.deltas[i]) / float64(t.timescale)
		}
		out.Delay = append(out.Delay, d)
	}
	if len(out.Image) == 0 {
		return nil, nil
	}

	return out, nil
}

func (f *file) decodeTrack(t *track) ([]*av1.Picture, error) {
	var d av1.Decoder
	d.FrameSizeLimit = f.frameSizeLimit
	var out []*av1.Picture

	// A sample needs at least one byte, so a table claiming more samples than
	// the file has bytes is describing data that cannot exist.
	if len(t.samples) > len(f.data) {
		return nil, errInvalid
	}

	for _, s := range t.samples {
		if s.len == 0 || s.off > uint64(len(f.data)) || s.len > uint64(len(f.data))-s.off {
			return nil, errInvalid
		}
		pics, err := d.DecodeOBUs(f.data[s.off : s.off+s.len])
		if err != nil {
			return nil, err
		}
		out = append(out, pics...)
	}

	return out, nil
}
