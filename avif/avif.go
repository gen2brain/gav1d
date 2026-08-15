/*
Package avif decodes AVIF images and image sequences.

	img, err := avif.Decode(r)

[Decode] returns *[image.NRGBA], or *[image.NRGBA64] above 8 bits. The package
registers itself with [image.RegisterFormat], so [image.Decode] works once it
is imported for side effects.

It handles 8, 10 and 12 bits, 4:2:0, 4:2:2, 4:4:4 and monochrome, alpha, film
grain, tiled images stored as a grid, and the clap, irot and imir transforms.
A file it cannot render returns an error rather than a wrong image:
[ErrUnsupported] for one that is well formed but asks for something this
package does not implement, [ErrInvalid] for one that is broken. A caller with
another decoder to fall back on wants the first and not the second.

# Options

[Options] are passed to any of the decode functions and default to zero:

	img, err := avif.Decode(r, avif.Options{
		AutoRotate:     true,    // apply the clap, irot and imir transforms
		FrameSizeLimit: 1 << 24, // bound a frame's area, in pixels
	})

FrameSizeLimit bounds what a header may ask to allocate, so that a corrupt or
hostile file cannot exhaust memory. Zero means [av1.DefaultFrameSizeLimit].

# Color

[Options.ToYCbCr] skips the conversion to RGB and hands back the planes the
bitstream carries: *[image.YCbCr], *[image.NYCbCrA] with alpha, or
*[image.Gray] for monochrome. Above 8 bits there is no such image type, so
*[image.NRGBA64] is returned anyway.

[image.YCbCr] reads its planes as full-range BT.601 whatever the file signals,
which is rarely what the file means. [DecodeColor] reports what they actually
are, so this is for reaching the samples rather than for display:

	img, ci, err := avif.DecodeColor(r, avif.Options{ToYCbCr: true})

[ColorInfo] carries the CICP code points and the range flag, plus the ICC
profile when the file has one. Matrix and FullRange are what the conversion to
RGB uses. Primaries and Transfer are reported but not applied, so RGB output
stays in the file's own color space; converting between color spaces is the
caller's to do, and [ApplyGainMap] is the one place this package does it.

# Sequences

[DecodeAll] returns every frame of an image sequence with its duration, and how
many times the animation repeats. A still image gives one frame.

	anim, err := avif.DecodeAll(r)

LoopCount follows [image/gif]: zero repeats forever, -1 shows each frame once,
and any other value plays the animation LoopCount+1 times.

# Metadata

[DecodeExif] reads the Exif item a file describes its image with, and
[RawExif] and [RawXMP] return the Exif and XMP payloads whole. All three
report [ErrNoExif] or [ErrNoXMP] when the file carries no such item.

	exif, err := avif.DecodeExif(r)

# Gain maps

A gain map is a second image and a set of coefficients that together take an
image to an HDR rendition of itself. [DecodeGainMap] returns the base image and
the [GainMap], and [ApplyGainMap] combines them for a display whose headroom is
a given number of stops above SDR white:

	base, gm, err := avif.DecodeGainMap(r)
	if gm != nil {
		img, err := avif.ApplyGainMap(base, ci, gm, headroom, out)
	}

The headroom is a property of the display rather than of the file, so it is the
caller's to supply. A file without a gain map, or with one this package does
not implement, yields a nil [GainMap] and a base image that is still usable on
its own.

# The bitstream

The [github.com/gen2brain/gav1d/av1] package decodes AV1 on its own, which is
what to reach for to decode video rather than images.
*/
package avif

import (
	"image"
	"io"

	"github.com/gen2brain/gav1d/av1"
)

const alphaURN = "urn:mpeg:mpegB:cicp:systems:auxiliary:alpha"

type file struct {
	src   *source
	meta  *metaBox
	movie *movie

	frameSizeLimit int
}

func parse(src *source) (*file, error) {
	f := &file{src: src}
	var brandOK bool

	err := src.eachBox(func(typ string, off, n uint64) error {
		switch typ {
		case "ftyp", "meta", "moov":
		default:
			return nil
		}
		b, err := src.at(off, n)
		if err != nil {
			return err
		}

		switch typ {
		case "ftyp":
			r := &reader{b: b}
			major := r.str4()
			r.u32()
			brands := []string{major}
			for r.remaining() >= 4 {
				brands = append(brands, r.str4())
			}
			for _, br := range brands {
				switch br {
				case "avif", "avis", "mif1", "msf1", "miaf":
					brandOK = true
				}
			}

		case "meta":
			m, err := parseMeta(b)
			if err != nil {
				return err
			}
			f.meta = m

		case "moov":
			mv, err := parseMoov(b)
			if err != nil {
				return err
			}
			f.movie = mv
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if !brandOK || f.meta == nil {
		return nil, ErrInvalid
	}

	return f, nil
}

// primary returns the primary item, falling back to the first av01 or grid item.
func (f *file) primary() *item {
	m := f.meta
	if it := m.items[m.primary]; it != nil && (it.typ == "av01" || it.typ == "grid") {
		return it
	}
	for _, id := range m.order {
		if it := m.items[id]; it.typ == "av01" || it.typ == "grid" {
			return it
		}
	}

	return nil
}

// alphaFor finds the auxiliary item carrying alpha for id.
func (f *file) alphaFor(id uint32) *item {
	m := f.meta
	for _, r := range m.refs {
		if r.typ != "auxl" || len(r.to) == 0 || r.to[0] != id {
			continue
		}
		it := m.items[r.from]
		if it == nil {
			continue
		}
		if p := m.prop(it, "auxC"); p != nil && p.auxC == alphaURN {
			return it
		}
	}

	return nil
}

func (f *file) dimensions(it *item) (int, int, error) {
	p := f.meta.prop(it, "ispe")
	if p == nil {
		return 0, 0, ErrInvalid
	}
	if p.w == 0 || p.h == 0 || p.w > 1<<20 || p.h > 1<<20 {
		return 0, 0, ErrInvalid
	}

	return int(p.w), int(p.h), nil
}

// decodeItem decodes a single av01 item to a picture.
func (f *file) decodeItem(it *item) (*av1.Picture, error) {
	if it.unsupported {
		return nil, ErrUnsupported
	}

	data, err := f.meta.data(it, f.src)
	if err != nil {
		return nil, err
	}

	var d av1.Decoder
	d.FrameSizeLimit = f.frameSizeLimit
	if p := f.meta.prop(it, "a1op"); p != nil {
		d.OperatingPoint = int(p.opIdx)
	}
	pics, err := d.DecodeOBUs(data)
	if err != nil {
		return nil, err
	}
	if len(pics) == 0 {
		return nil, ErrInvalid
	}

	// 0xffff selects every layer, which is the last picture decoded.
	if p := f.meta.prop(it, "lsel"); p != nil && p.layer != 0xffff {
		if int(p.layer) >= len(pics) {
			return nil, ErrInvalid
		}

		return pics[p.layer], nil
	}

	return pics[len(pics)-1], nil
}

type gridInfo struct {
	rows, cols int
	w, h       int
}

func parseGrid(b []byte) (gridInfo, error) {
	r := &reader{b: b}
	r.u8()
	flags := r.u8()
	var g gridInfo
	g.rows = int(r.u8()) + 1
	g.cols = int(r.u8()) + 1
	if flags&1 != 0 {
		g.w, g.h = int(r.u32()), int(r.u32())
	} else {
		g.w, g.h = int(r.u16()), int(r.u16())
	}
	if r.err || g.w == 0 || g.h == 0 {
		return g, ErrInvalid
	}

	return g, nil
}

// decodeGrid decodes a grid item's dimg tiles and stitches them row-major.
func (f *file) gridOf(it *item) (gridInfo, []uint32, error) {
	data, err := f.meta.data(it, f.src)
	if err != nil {
		return gridInfo{}, nil, err
	}
	g, err := parseGrid(data)
	if err != nil {
		return gridInfo{}, nil, err
	}
	tiles := f.meta.refsTo("dimg", it.id)
	if len(tiles) != g.rows*g.cols {
		return gridInfo{}, nil, ErrInvalid
	}

	return g, tiles, nil
}

func (f *file) decodeGrid(it *item) (*av1.Picture, error) {
	if it.unsupported {
		return nil, ErrUnsupported
	}

	g, tiles, err := f.gridOf(it)
	if err != nil {
		return nil, err
	}

	return f.decodeTiles(g, tiles)
}

func (f *file) decodeTiles(g gridInfo, tiles []uint32) (*av1.Picture, error) {
	var err error
	pics := make([]*av1.Picture, len(tiles))
	for i, id := range tiles {
		t := f.meta.items[id]
		if t == nil || t.typ != "av01" {
			return nil, ErrInvalid
		}
		if pics[i], err = f.decodeItem(t); err != nil {
			return nil, err
		}
	}

	first := pics[0]
	tw, th := first.Width, first.Height
	for _, p := range pics {
		if p.Width != tw || p.Height != th || p.Layout != first.Layout ||
			p.BitDepth != first.BitDepth {
			return nil, ErrInvalid
		}
	}
	if tw*g.cols < g.w || th*g.rows < g.h {
		return nil, ErrInvalid
	}

	return stitch(pics, g, tw, th), nil
}

func stitch(pics []*av1.Picture, g gridInfo, tw, th int) *av1.Picture {
	first := pics[0]
	ssHor, ssVer := subsampling(first.Layout)
	bpp := 1
	if first.BitDepth > 8 {
		bpp = 2
	}

	out := &av1.Picture{
		Width:          g.w,
		Height:         g.h,
		Layout:         first.Layout,
		BitDepth:       first.BitDepth,
		ColorPrimaries: first.ColorPrimaries,
		Transfer:       first.Transfer,
		MatrixCoeffs:   first.MatrixCoeffs,
		FullRange:      first.FullRange,
	}
	for pl := range 3 {
		if first.Data[pl] == nil {
			continue
		}
		pw, ph := g.w, g.h
		if pl != 0 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		out.Stride[pl] = pw * bpp
		out.Data[pl] = make([]byte, out.Stride[pl]*ph)
	}

	for i, p := range pics {
		row, col := i/g.cols, i%g.cols
		for pl := range 3 {
			if out.Data[pl] == nil {
				continue
			}
			sx, sy := col*tw, row*th
			cw, ch := tw, th
			ow, oh := g.w, g.h
			if pl != 0 {
				sx, sy = sx>>ssHor, sy>>ssVer
				cw, ch = cw>>ssHor, ch>>ssVer
				ow, oh = (ow+ssHor)>>ssHor, (oh+ssVer)>>ssVer
			}
			cw = min(cw, ow-sx)
			ch = min(ch, oh-sy)
			if cw <= 0 || ch <= 0 {
				continue
			}
			for y := range ch {
				dst := (sy+y)*out.Stride[pl] + sx*bpp
				src := y * p.Stride[pl]
				copy(out.Data[pl][dst:dst+cw*bpp], p.Data[pl][src:src+cw*bpp])
			}
		}
	}

	return out
}

func subsampling(layout int) (int, int) {
	switch layout {
	case av1.LayoutI420:
		return 1, 1
	case av1.LayoutI422:
		return 1, 0
	default:
		return 0, 0
	}
}

// gridAlpha assembles alpha from items attached to a grid's tiles.
func (f *file) gridAlpha(it *item) (*av1.Picture, error) {
	g, tiles, err := f.gridOf(it)
	if err != nil {
		return nil, err
	}

	ids := make([]uint32, len(tiles))
	for i, id := range tiles {
		a := f.alphaFor(id)
		if a == nil {
			return nil, nil
		}
		ids[i] = a.id
	}

	return f.decodeTiles(g, ids)
}

func (f *file) decodePrimary() (*av1.Picture, *av1.Picture, error) {
	it := f.primary()
	if it == nil {
		return nil, nil, ErrInvalid
	}

	var pic *av1.Picture
	var err error
	if it.typ == "grid" {
		pic, err = f.decodeGrid(it)
	} else {
		pic, err = f.decodeItem(it)
	}
	if err != nil {
		return nil, nil, err
	}

	w, h, err := f.dimensions(it)
	if err != nil {
		return nil, nil, err
	}
	if w > pic.Width || h > pic.Height {
		return nil, nil, ErrInvalid
	}
	pic.Width, pic.Height = w, h

	var alpha *av1.Picture
	if a := f.alphaFor(it.id); a != nil {
		if a.typ == "grid" {
			alpha, err = f.decodeGrid(a)
		} else {
			alpha, err = f.decodeItem(a)
		}
		if err != nil {
			return nil, nil, err
		}
	} else if it.typ == "grid" {
		alpha, err = f.gridAlpha(it)
		if err != nil {
			return nil, nil, err
		}
	}
	if alpha != nil {
		if alpha.Width < w || alpha.Height < h {
			return nil, nil, ErrInvalid
		}
		alpha.Width, alpha.Height = w, h
	}

	return pic, alpha, nil
}

// ColorInfo is a file's color description, as the code points of
// ISO/IEC 23091-2. An nclx box replaces the sequence header's description
// whole, which is how it is resolved here.
//
// Only Matrix and FullRange take part in the conversion to RGB. Primaries and
// Transfer are reported but not applied, so RGB output stays in the file's own
// color space, and ICCP is carried rather than honored.
type ColorInfo struct {
	Primaries uint16
	Transfer  uint16
	Matrix    uint16
	FullRange bool
	// ICCP is the embedded ICC profile, for files that carry one in place of
	// an nclx description. It aliases the input, so it is not a copy.
	ICCP []byte
}

// AVIF holds the images of an AVIF file, which may be an image sequence.
type AVIF struct {
	// Image holds the decoded frames, *image.NRGBA or *image.NRGBA64.
	Image []image.Image
	// Delay holds each frame's duration in seconds.
	Delay []float64
	// LoopCount controls how many times the animation restarts, following
	// image/gif: zero loops forever, -1 shows each frame once, and any other
	// value plays the animation LoopCount+1 times. A file whose track carries
	// no edit list loops forever.
	LoopCount int
	// Color describes the color space the frames were decoded from.
	Color ColorInfo
}

// Options are the decoding parameters.
type Options struct {
	// AutoRotate applies the clap/irot/imir transforms, forcing NRGBA output
	// when it transforms.
	AutoRotate bool
	// FrameSizeLimit bounds a frame's area in pixels. Zero means
	// av1.DefaultFrameSizeLimit; a negative value removes the limit.
	FrameSizeLimit int
	// ToYCbCr forces the image's native color space instead of NRGBA:
	// *image.YCbCr, *image.NYCbCrA when there is alpha, or *image.Gray when
	// the image is monochrome. Above 8 bits NRGBA64 is returned anyway.
	// image.YCbCr reads the planes as full-range BT.601 whatever the file
	// signals, so this is for reaching the samples, not for display.
	// DecodeColor reports what the samples actually are.
	ToYCbCr bool
}

// Decode reads an AVIF image as *image.NRGBA, or *image.NRGBA64 above 8 bits.
func Decode(r io.Reader, opts ...Options) (image.Image, error) {
	src, err := srcFor(r)
	if err != nil {
		return nil, err
	}
	f, err := parse(src)
	if err != nil {
		return nil, err
	}

	o := options(opts)
	f.frameSizeLimit = o.FrameSizeLimit

	img, _, err := f.decodeStill(o)

	return img, err
}

// DecodeColor reads an AVIF image and reports the color space it came from,
// which is what the native planes of Options.ToYCbCr need to be read against.
func DecodeColor(r io.Reader, opts ...Options) (image.Image, ColorInfo, error) {
	src, err := srcFor(r)
	if err != nil {
		return nil, ColorInfo{}, err
	}
	f, err := parse(src)
	if err != nil {
		return nil, ColorInfo{}, err
	}

	o := options(opts)
	f.frameSizeLimit = o.FrameSizeLimit

	return f.decodeStill(o)
}

// DecodeAll reads every frame of an AVIF file. A still image gives one frame.
func DecodeAll(r io.Reader, opts ...Options) (*AVIF, error) {
	src, err := srcFor(r)
	if err != nil {
		return nil, err
	}
	f, err := parse(src)
	if err != nil {
		return nil, err
	}
	o := options(opts)
	f.frameSizeLimit = o.FrameSizeLimit

	if f.movie != nil {
		if a, err := f.decodeSequence(o); err == nil && a != nil {
			return a, nil
		}
	}

	img, ci, err := f.decodeStill(o)
	if err != nil {
		return nil, err
	}

	return &AVIF{Image: []image.Image{img}, Delay: []float64{0}, Color: ci}, nil
}

// DecodeConfig reads the dimensions and color model without decoding pixels.
func DecodeConfig(r io.Reader) (image.Config, error) {
	src, err := srcFor(r)
	if err != nil {
		return image.Config{}, err
	}
	f, err := parse(src)
	if err != nil {
		return image.Config{}, err
	}
	it := f.primary()
	if it == nil {
		return image.Config{}, ErrInvalid
	}
	w, h, err := f.dimensions(it)
	if err != nil {
		return image.Config{}, err
	}

	return image.Config{Width: w, Height: h, ColorModel: colorModelFor(f, it)}, nil
}

// ycbcrFor drops the native form when a transform has to run over the pixels.
func (o Options) ycbcrFor(f *file, it *item) bool {
	return o.ToYCbCr && !(o.AutoRotate && f.hasTransform(it))
}

func options(opts []Options) Options {
	if len(opts) > 0 {
		return opts[0]
	}

	return Options{}
}

func (f *file) decodeStill(o Options) (image.Image, ColorInfo, error) {
	pic, alpha, err := f.decodePrimary()
	if err != nil {
		return nil, ColorInfo{}, err
	}
	it := f.primary()
	ci := f.colorInfo(it, pic)
	img, err := toImage(f, it, pic, alpha, o.ycbcrFor(f, it))
	if err != nil {
		return nil, ColorInfo{}, err
	}
	if !aliasesPicture(img) {
		pic.Release()
		if alpha != nil {
			alpha.Release()
		}
	}
	if !o.AutoRotate {
		return img, ci, nil
	}
	img, err = f.applyTransforms(img, it)
	if err != nil {
		return nil, ColorInfo{}, err
	}

	return img, ci, nil
}

// srcFor reads by range when the reader can seek, bounded to what it has left.
func srcFor(r io.Reader) (*source, error) {
	ra, raOK := r.(io.ReaderAt)
	sk, skOK := r.(io.Seeker)
	if raOK && skOK {
		cur, err1 := sk.Seek(0, io.SeekCurrent)
		end, err2 := sk.Seek(0, io.SeekEnd)
		if err1 == nil && err2 == nil && end > cur {
			n := end - cur

			return &source{r: io.NewSectionReader(ra, cur, n), size: uint64(n)}, nil
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return memSource(data), nil
}

func decodeWrapper(r io.Reader) (image.Image, error) {
	return Decode(r)
}

func init() {
	image.RegisterFormat("avif", "????ftypavif", decodeWrapper, DecodeConfig)
	image.RegisterFormat("avif", "????ftypavis", decodeWrapper, DecodeConfig)
}
