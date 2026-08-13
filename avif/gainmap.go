package avif

import (
	"image"
	"io"

	"github.com/gen2brain/gav1d/av1"
)

// Rational is a value the gain map metadata carries as a fraction.
type Rational struct {
	N int32
	D uint32
}

// Float returns the value, or zero when the denominator is.
func (r Rational) Float() float64 {
	if r.D == 0 {
		return 0
	}

	return float64(r.N) / float64(r.D)
}

// GainMap is the HDR rendition a tone mapped item describes: an image of
// per-pixel gains and the metadata that says how to apply it to the base
// image. The fields are ISO/IEC 21496-1, one entry per channel.
type GainMap struct {
	// Image is the decoded gain map, which may be smaller than the base
	// image and is often monochrome.
	Image image.Image

	// UseBaseColorSpace applies the gain in the base image's color space
	// rather than the alternate rendition's.
	UseBaseColorSpace bool

	// Alt describes the alternate rendition the gain map leads to, from the
	// tone mapped item's own color property. Unspecified primaries mean the
	// base image's.
	Alt ColorInfo

	BaseHDRHeadroom      Rational
	AlternateHDRHeadroom Rational

	Min             [3]Rational
	Max             [3]Rational
	Gamma           [3]Rational
	BaseOffset      [3]Rational
	AlternateOffset [3]Rational
}

// gainMapVersion is the metadata version this reader implements. A file whose
// minimum_version is above it must not be tone mapped.
const gainMapVersion = 0

// parseToneMap reads a tmap item's payload. It reports false when the file asks
// for a version this reader does not implement, which is not an error: the base
// image still decodes.
func parseToneMap(b []byte) (*GainMap, bool) {
	r := &reader{b: b}

	if r.u8() != 0 {
		return nil, false
	}
	minVersion := r.u16()
	writerVersion := r.u16()
	if r.err || minVersion > gainMapVersion || writerVersion < minVersion {
		return nil, false
	}

	flags := r.u8()
	channels := 1
	if flags&0x80 != 0 {
		channels = 3
	}

	g := &GainMap{UseBaseColorSpace: flags&0x40 != 0}
	g.BaseHDRHeadroom = Rational{int32(r.u32()), r.u32()}
	g.AlternateHDRHeadroom = Rational{int32(r.u32()), r.u32()}

	for c := range channels {
		g.Min[c] = Rational{int32(r.u32()), r.u32()}
		g.Max[c] = Rational{int32(r.u32()), r.u32()}
		g.Gamma[c] = Rational{int32(r.u32()), r.u32()}
		g.BaseOffset[c] = Rational{int32(r.u32()), r.u32()}
		g.AlternateOffset[c] = Rational{int32(r.u32()), r.u32()}
	}
	if r.err {
		return nil, false
	}
	// A writer no newer than this reader may not leave anything behind.
	if writerVersion <= gainMapVersion && r.remaining() != 0 {
		return nil, false
	}

	for c := channels; c < 3; c++ {
		g.Min[c] = g.Min[0]
		g.Max[c] = g.Max[0]
		g.Gamma[c] = g.Gamma[0]
		g.BaseOffset[c] = g.BaseOffset[0]
		g.AlternateOffset[c] = g.AlternateOffset[0]
	}

	for c := range 3 {
		if g.Gamma[c].N <= 0 || g.Gamma[c].D == 0 {
			return nil, false
		}
		if g.Min[c].D == 0 || g.Max[c].D == 0 ||
			g.BaseOffset[c].D == 0 || g.AlternateOffset[c].D == 0 {
			return nil, false
		}
		if g.Max[c].Float() < g.Min[c].Float() {
			return nil, false
		}
	}
	if g.BaseHDRHeadroom.D == 0 || g.AlternateHDRHeadroom.D == 0 ||
		g.BaseHDRHeadroom.N < 0 || g.AlternateHDRHeadroom.N < 0 {
		return nil, false
	}

	return g, true
}

// toneMapItem finds the tmap item and the two items it derives from, which are
// the base image and the gain map in that order.
func (f *file) toneMapItem() (tmap, base, gain *item) {
	m := f.meta
	for _, id := range m.order {
		it := m.items[id]
		if it == nil || it.typ != "tmap" || it.unsupported {
			continue
		}
		to := m.refsTo("dimg", it.id)
		if len(to) != 2 {
			continue
		}

		return it, m.items[to[0]], m.items[to[1]]
	}

	return nil, nil, nil
}

// gainMap decodes the gain map image and reads its metadata. It returns nil
// when the file carries none, or one this reader does not implement.
func (f *file) gainMap(o Options) (*GainMap, error) {
	tmap, base, gain := f.toneMapItem()
	if tmap == nil || base == nil || gain == nil {
		return nil, nil
	}

	data, err := f.meta.data(tmap, f.src)
	if err != nil {
		return nil, nil
	}
	g, ok := parseToneMap(data)
	if !ok {
		return nil, nil
	}
	g.Alt = ColorInfo{Primaries: cicpUnspecified, Transfer: cicpUnspecified, Matrix: cicpUnspecified}
	if p := f.meta.prop(tmap, "colr"); p != nil && p.colr != nil {
		if p.colr.hasNCLX {
			g.Alt.Primaries = p.colr.primaries
			g.Alt.Transfer = p.colr.transfer
			g.Alt.Matrix = p.colr.matrix
			g.Alt.FullRange = p.colr.fullRange
		}
		g.Alt.ICCP = p.colr.icc
	}

	var pic *av1.Picture
	if gain.typ == "grid" {
		pic, err = f.decodeGrid(gain)
	} else {
		pic, err = f.decodeItem(gain)
	}
	if err != nil {
		return nil, err
	}

	w, h, err := f.dimensions(gain)
	if err != nil {
		return nil, err
	}
	if w > pic.Width || h > pic.Height {
		return nil, ErrInvalid
	}
	pic.Width, pic.Height = w, h

	img, err := toImage(f, gain, pic, nil, o.ycbcrFor(f, gain))
	if err != nil {
		return nil, err
	}
	if !aliasesPicture(img) {
		pic.Release()
	}
	g.Image = img

	return g, nil
}

// DecodeGainMap decodes the base image and, when the file carries one, the gain
// map that takes it to its HDR rendition. The gain map is nil for a file
// without one, and for one this reader does not implement, in both cases
// leaving the base image usable on its own.
func DecodeGainMap(r io.Reader, opts ...Options) (image.Image, *GainMap, error) {
	src, err := srcFor(r)
	if err != nil {
		return nil, nil, err
	}
	f, err := parse(src)
	if err != nil {
		return nil, nil, err
	}

	o := options(opts)
	f.frameSizeLimit = o.FrameSizeLimit

	img, _, err := f.decodeStill(o)
	if err != nil {
		return nil, nil, err
	}

	g, err := f.gainMap(o)
	if err != nil {
		return nil, nil, err
	}

	return img, g, nil
}
