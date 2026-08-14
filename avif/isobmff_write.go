package avif

import "encoding/binary"

type boxWriter struct {
	b []byte
}

func (w *boxWriter) u8(v uint8)   { w.b = append(w.b, v) }
func (w *boxWriter) u16(v uint16) { w.b = binary.BigEndian.AppendUint16(w.b, v) }
func (w *boxWriter) u32(v uint32) { w.b = binary.BigEndian.AppendUint32(w.b, v) }
func (w *boxWriter) raw(p []byte) { w.b = append(w.b, p...) }
func (w *boxWriter) str(s string) { w.b = append(w.b, s...) }

func (w *boxWriter) cstr(s string) {
	w.b = append(w.b, s...)
	w.b = append(w.b, 0)
}

func (w *boxWriter) box(typ string, body func()) {
	start := len(w.b)
	w.u32(0)
	w.str(typ)
	body()
	binary.BigEndian.PutUint32(w.b[start:], uint32(len(w.b)-start))
}

func (w *boxWriter) fullBox(typ string, version uint8, flags uint32, body func()) {
	w.box(typ, func() {
		w.u32(uint32(version)<<24 | flags&0xffffff)
		body()
	})
}

type encProp struct {
	body      []byte
	essential bool
}

type encItem struct {
	id    uint16
	name  string
	data  []byte
	props []int
	off   uint32
}

func propBox(typ string, essential bool, body func(*boxWriter)) encProp {
	var w boxWriter
	w.box(typ, func() { body(&w) })

	return encProp{body: w.b, essential: essential}
}

func ispeProp(width, height int) encProp {
	var w boxWriter
	w.fullBox("ispe", 0, 0, func() {
		w.u32(uint32(width))
		w.u32(uint32(height))
	})

	return encProp{body: w.b}
}

func pixiProp(channels int) encProp {
	var w boxWriter
	w.fullBox("pixi", 0, 0, func() {
		w.u8(uint8(channels))
		for range channels {
			w.u8(8)
		}
	})

	return encProp{body: w.b}
}

func av1CProp(mono bool, seqHdr []byte) encProp {
	return propBox("av1C", true, func(w *boxWriter) {
		w.u8(0x81)
		w.u8(0)

		var v uint8 = 1<<3 | 1<<2
		if mono {
			v |= 1 << 4
		}
		w.u8(v)
		w.u8(0)
		w.raw(seqHdr)
	})
}

func colrProp(ci ColorInfo) encProp {
	return propBox("colr", false, func(w *boxWriter) {
		w.str("nclx")
		w.u16(ci.Primaries)
		w.u16(ci.Transfer)
		w.u16(ci.Matrix)
		var full uint8
		if ci.FullRange {
			full = 0x80
		}
		w.u8(full)
	})
}

func auxCProp(urn string) encProp {
	var w boxWriter
	w.fullBox("auxC", 0, 0, func() { w.cstr(urn) })

	return encProp{body: w.b, essential: true}
}

func writeMeta(items []encItem, props []encProp, alphaOf uint16) []byte {
	var w boxWriter

	w.fullBox("meta", 0, 0, func() {
		w.fullBox("hdlr", 0, 0, func() {
			w.u32(0)
			w.str("pict")
			w.u32(0)
			w.u32(0)
			w.u32(0)
			w.cstr("")
		})

		w.fullBox("pitm", 0, 0, func() { w.u16(items[0].id) })

		w.fullBox("iloc", 0, 0, func() {
			w.u8(4<<4 | 4)
			w.u8(0)
			w.u16(uint16(len(items)))
			for _, it := range items {
				w.u16(it.id)
				w.u16(0)
				w.u16(1)
				w.u32(it.off)
				w.u32(uint32(len(it.data)))
			}
		})

		w.fullBox("iinf", 0, 0, func() {
			w.u16(uint16(len(items)))
			for _, it := range items {
				w.fullBox("infe", 2, 0, func() {
					w.u16(it.id)
					w.u16(0)
					w.str("av01")
					w.cstr(it.name)
				})
			}
		})

		if alphaOf != 0 {
			w.fullBox("iref", 0, 0, func() {
				w.box("auxl", func() {
					w.u16(items[1].id)
					w.u16(1)
					w.u16(alphaOf)
				})
			})
		}

		w.box("iprp", func() {
			w.box("ipco", func() {
				for _, p := range props {
					w.raw(p.body)
				}
			})

			w.fullBox("ipma", 0, 0, func() {
				w.u32(uint32(len(items)))
				for _, it := range items {
					w.u16(it.id)
					w.u8(uint8(len(it.props)))
					for _, idx := range it.props {
						v := uint8(idx + 1)
						if props[idx].essential {
							v |= 0x80
						}
						w.u8(v)
					}
				}
			})
		})
	})

	return w.b
}

func buildAVIF(items []encItem, props []encProp, alphaOf uint16) []byte {
	var ftyp boxWriter
	ftyp.box("ftyp", func() {
		ftyp.str("avif")
		ftyp.u32(0)
		ftyp.str("avif")
		ftyp.str("mif1")
		ftyp.str("miaf")
		ftyp.str("MA1B")
	})

	// Extent offsets are absolute, so meta is measured before it is filled.
	base := len(ftyp.b) + len(writeMeta(items, props, alphaOf)) + 8

	off := uint32(base)
	for i := range items {
		items[i].off = off
		off += uint32(len(items[i].data))
	}

	out := ftyp.b
	out = append(out, writeMeta(items, props, alphaOf)...)

	var mdat boxWriter
	mdat.box("mdat", func() {
		for _, it := range items {
			mdat.raw(it.data)
		}
	})

	return append(out, mdat.b...)
}
