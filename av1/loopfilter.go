package av1

type filterLUT struct {
	e     [64]uint8
	i     [64]uint8
	sharp [2]uint64
}

func loopFilter[P pixel](dst []P, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
	fThr := int32(1) << bitdepthMin8
	e <<= bitdepthMin8
	i <<= bitdepthMin8
	h <<= bitdepthMin8

	for n := 0; n < lines; n, dstOff = n+1, dstOff+stridea {
		var p6, p5, p4, p3, p2 int32
		var q2, q3, q4, q5, q6 int32
		p1 := int32(dst[dstOff+strideb*-2])
		p0 := int32(dst[dstOff+strideb*-1])
		q0 := int32(dst[dstOff+strideb*0])
		q1 := int32(dst[dstOff+strideb*1])

		fm := abs32(p1-p0) <= i && abs32(q1-q0) <= i &&
			abs32(p0-q0)*2+(abs32(p1-q1)>>1) <= e

		if wd > 4 {
			p2 = int32(dst[dstOff+strideb*-3])
			q2 = int32(dst[dstOff+strideb*2])

			fm = fm && abs32(p2-p1) <= i && abs32(q2-q1) <= i

			if wd > 6 {
				p3 = int32(dst[dstOff+strideb*-4])
				q3 = int32(dst[dstOff+strideb*3])

				fm = fm && abs32(p3-p2) <= i && abs32(q3-q2) <= i
			}
		}
		if !fm {
			continue
		}

		var flat8out, flat8in bool

		if wd >= 16 {
			p6 = int32(dst[dstOff+strideb*-7])
			p5 = int32(dst[dstOff+strideb*-6])
			p4 = int32(dst[dstOff+strideb*-5])
			q4 = int32(dst[dstOff+strideb*4])
			q5 = int32(dst[dstOff+strideb*5])
			q6 = int32(dst[dstOff+strideb*6])

			flat8out = abs32(p6-p0) <= fThr && abs32(p5-p0) <= fThr &&
				abs32(p4-p0) <= fThr && abs32(q4-q0) <= fThr &&
				abs32(q5-q0) <= fThr && abs32(q6-q0) <= fThr
		}

		if wd >= 6 {
			flat8in = abs32(p2-p0) <= fThr && abs32(p1-p0) <= fThr &&
				abs32(q1-q0) <= fThr && abs32(q2-q0) <= fThr
		}

		if wd >= 8 {
			flat8in = flat8in && abs32(p3-p0) <= fThr && abs32(q3-q0) <= fThr
		}

		set := func(k int, v int32) { dst[dstOff+strideb*k] = P(v) }

		switch {
		case wd >= 16 && flat8out && flat8in:
			set(-6, (p6+p6+p6+p6+p6+p6*2+p5*2+p4*2+p3+p2+p1+p0+q0+8)>>4)
			set(-5, (p6+p6+p6+p6+p6+p5*2+p4*2+p3*2+p2+p1+p0+q0+q1+8)>>4)
			set(-4, (p6+p6+p6+p6+p5+p4*2+p3*2+p2*2+p1+p0+q0+q1+q2+8)>>4)
			set(-3, (p6+p6+p6+p5+p4+p3*2+p2*2+p1*2+p0+q0+q1+q2+q3+8)>>4)
			set(-2, (p6+p6+p5+p4+p3+p2*2+p1*2+p0*2+q0+q1+q2+q3+q4+8)>>4)
			set(-1, (p6+p5+p4+p3+p2+p1*2+p0*2+q0*2+q1+q2+q3+q4+q5+8)>>4)
			set(0, (p5+p4+p3+p2+p1+p0*2+q0*2+q1*2+q2+q3+q4+q5+q6+8)>>4)
			set(1, (p4+p3+p2+p1+p0+q0*2+q1*2+q2*2+q3+q4+q5+q6+q6+8)>>4)
			set(2, (p3+p2+p1+p0+q0+q1*2+q2*2+q3*2+q4+q5+q6+q6+q6+8)>>4)
			set(3, (p2+p1+p0+q0+q1+q2*2+q3*2+q4*2+q5+q6+q6+q6+q6+8)>>4)
			set(4, (p1+p0+q0+q1+q2+q3*2+q4*2+q5*2+q6+q6+q6+q6+q6+8)>>4)
			set(5, (p0+q0+q1+q2+q3+q4*2+q5*2+q6*2+q6+q6+q6+q6+q6+8)>>4)

		case wd >= 8 && flat8in:
			set(-3, (p3+p3+p3+2*p2+p1+p0+q0+4)>>3)
			set(-2, (p3+p3+p2+2*p1+p0+q0+q1+4)>>3)
			set(-1, (p3+p2+p1+2*p0+q0+q1+q2+4)>>3)
			set(0, (p2+p1+p0+2*q0+q1+q2+q3+4)>>3)
			set(1, (p1+p0+q0+2*q1+q2+q3+q3+4)>>3)
			set(2, (p0+q0+q1+2*q2+q3+q3+q3+4)>>3)

		case wd == 6 && flat8in:
			set(-2, (p2+2*p2+2*p1+2*p0+q0+4)>>3)
			set(-1, (p2+2*p1+2*p0+2*q0+q1+4)>>3)
			set(0, (p1+2*p0+2*q0+2*q1+q2+4)>>3)
			set(1, (p0+2*q0+2*q1+2*q2+q2+4)>>3)

		default:
			hev := abs32(p1-p0) > h || abs32(q1-q0) > h
			lim := int32(128) << bitdepthMin8
			clipDiff := func(v int32) int32 { return clip(v, -lim, lim-1) }

			if hev {
				f := clipDiff(p1 - q1)
				f = clipDiff(3*(q0-p0) + f)

				f1 := min(f+4, lim-1) >> 3
				f2 := min(f+3, lim-1) >> 3

				set(-1, clip(p0+f2, 0, bitdepthMax))
				set(0, clip(q0-f1, 0, bitdepthMax))
			} else {
				f := clipDiff(3 * (q0 - p0))

				f1 := min(f+4, lim-1) >> 3
				f2 := min(f+3, lim-1) >> 3

				set(-1, clip(p0+f2, 0, bitdepthMax))
				set(0, clip(q0-f1, 0, bitdepthMax))

				f = (f1 + 1) >> 1
				set(-2, clip(p1+f, 0, bitdepthMax))
				set(1, clip(q1-f, 0, bitdepthMax))
			}
		}
	}
}

func loopFilterHSb128y[P pixel](dsp *dspContext[P], dst []P, dstOff, stride int,
	vmask *[3]uint32, l []uint8, lOff, b4Stride int, lut *filterLUT,
	bitdepthMax int32,
) {
	vm := vmask[0] | vmask[1] | vmask[2]
	for b := uint32(1); vm&^(b-1) != 0; {
		lv, wd := lfSegY(vmask, b, l, lOff, -4)
		if lv == 0 {
			b, dstOff, lOff = b<<1, dstOff+4*stride, lOff+b4Stride*4

			continue
		}

		start, lines := dstOff, 0
		for {
			lines += 4
			b, dstOff, lOff = b<<1, dstOff+4*stride, lOff+b4Stride*4
			if vm&^(b-1) == 0 {
				break
			}
			nlv, nwd := lfSegY(vmask, b, l, lOff, -4)
			if nlv != lv || nwd != wd {
				break
			}
		}

		h := int32(lv >> 4)
		e, i := int32(lut.e[lv]), int32(lut.i[lv])
		dsp.loopFilter(dst, start, e, i, h, stride, 1, wd, lines, bitdepthMax)
	}
}

func loopFilterVSb128y[P pixel](dsp *dspContext[P], dst []P, dstOff, stride int,
	vmask *[3]uint32, l []uint8, lOff, b4Stride int, lut *filterLUT,
	bitdepthMax int32,
) {
	vm := vmask[0] | vmask[1] | vmask[2]
	for b := uint32(1); vm&^(b-1) != 0; {
		lv, wd := lfSegY(vmask, b, l, lOff, -b4Stride*4)
		if lv == 0 {
			b, dstOff, lOff = b<<1, dstOff+4, lOff+4

			continue
		}

		start, lines := dstOff, 0
		for {
			lines += 4
			b, dstOff, lOff = b<<1, dstOff+4, lOff+4
			if vm&^(b-1) == 0 {
				break
			}
			nlv, nwd := lfSegY(vmask, b, l, lOff, -b4Stride*4)
			if nlv != lv || nwd != wd {
				break
			}
		}

		h := int32(lv >> 4)
		e, i := int32(lut.e[lv]), int32(lut.i[lv])
		dsp.loopFilter(dst, start, e, i, h, 1, stride, wd, lines, bitdepthMax)
	}
}

func loopFilterHSb128uv[P pixel](dsp *dspContext[P], dst []P, dstOff, stride int,
	vmask *[2]uint32, l []uint8, lOff, b4Stride int, lut *filterLUT,
	bitdepthMax int32,
) {
	vm := vmask[0] | vmask[1]
	for b := uint32(1); vm&^(b-1) != 0; {
		lv, wd := lfSegUV(vmask, b, l, lOff, -4)
		if lv == 0 {
			b, dstOff, lOff = b<<1, dstOff+4*stride, lOff+b4Stride*4

			continue
		}

		start, lines := dstOff, 0
		for {
			lines += 4
			b, dstOff, lOff = b<<1, dstOff+4*stride, lOff+b4Stride*4
			if vm&^(b-1) == 0 {
				break
			}
			nlv, nwd := lfSegUV(vmask, b, l, lOff, -4)
			if nlv != lv || nwd != wd {
				break
			}
		}

		h := int32(lv >> 4)
		e, i := int32(lut.e[lv]), int32(lut.i[lv])
		dsp.loopFilter(dst, start, e, i, h, stride, 1, wd, lines, bitdepthMax)
	}
}

func loopFilterVSb128uv[P pixel](dsp *dspContext[P], dst []P, dstOff, stride int,
	vmask *[2]uint32, l []uint8, lOff, b4Stride int, lut *filterLUT,
	bitdepthMax int32,
) {
	vm := vmask[0] | vmask[1]
	for b := uint32(1); vm&^(b-1) != 0; {
		lv, wd := lfSegUV(vmask, b, l, lOff, -b4Stride*4)
		if lv == 0 {
			b, dstOff, lOff = b<<1, dstOff+4, lOff+4

			continue
		}

		start, lines := dstOff, 0
		for {
			lines += 4
			b, dstOff, lOff = b<<1, dstOff+4, lOff+4
			if vm&^(b-1) == 0 {
				break
			}
			nlv, nwd := lfSegUV(vmask, b, l, lOff, -b4Stride*4)
			if nlv != lv || nwd != wd {
				break
			}
		}

		h := int32(lv >> 4)
		e, i := int32(lut.e[lv]), int32(lut.i[lv])
		dsp.loopFilter(dst, start, e, i, h, 1, stride, wd, lines, bitdepthMax)
	}
}

// lfSegY and lfSegUV resolve one four-sample segment to its filter level and
// width, or a zero level where the edge is not filtered. Neighbouring segments
// along an edge usually agree, so the drivers gather the runs that do and hand
// the kernel all of their lines at once.
func lfSegY(vmask *[3]uint32, b uint32, l []uint8, lOff, back int) (uint8, int) {
	if (vmask[0]|vmask[1]|vmask[2])&b == 0 {
		return 0, 0
	}
	lv := l[lOff]
	if lv == 0 {
		lv = l[lOff+back]
	}
	if lv == 0 {
		return 0, 0
	}

	switch {
	case vmask[2]&b != 0:
		return lv, 16
	case vmask[1]&b != 0:
		return lv, 8
	}

	return lv, 4
}

func lfSegUV(vmask *[2]uint32, b uint32, l []uint8, lOff, back int) (uint8, int) {
	if (vmask[0]|vmask[1])&b == 0 {
		return 0, 0
	}
	lv := l[lOff]
	if lv == 0 {
		lv = l[lOff+back]
	}
	if lv == 0 {
		return 0, 0
	}
	if vmask[1]&b != 0 {
		return lv, 6
	}

	return lv, 4
}
