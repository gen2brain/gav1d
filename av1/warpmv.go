package av1

func clipWmp(v int32) int32 {
	cv := clip(v, -32768, 32767)

	return applySign((abs32(cv)+32)>>6, cv) * (1 << 6)
}

func resolveDivisor32(d uint32) (int32, int) {
	shift := ulog2(d)
	e := int32(d) - 1<<shift
	var f int32
	if shift > 8 {
		f = (e + 1<<(shift-9)) >> (shift - 8)
	} else {
		f = e << (8 - shift)
	}

	return int32(divLut[f]), shift + 14
}

func resolveDivisor64(d uint64) (int32, int) {
	shift := u64log2(d)
	e := int64(d) - 1<<shift
	var f int64
	if shift > 8 {
		f = (e + 1<<(shift-9)) >> (shift - 8)
	} else {
		f = e << (8 - shift)
	}

	return int32(divLut[f]), shift + 14
}

func getShearParams(wm *warpedMotionParams) bool {
	mat := &wm.matrix

	if mat[2] <= 0 {
		return true
	}

	wm.abcd[0] = int16(clipWmp(mat[2] - 0x10000))
	wm.abcd[1] = int16(clipWmp(mat[3]))

	div, shift := resolveDivisor32(uint32(abs32(mat[2])))
	y := applySign(div, mat[2])
	v1 := int64(mat[4]) * 0x10000 * int64(y)
	rnd := int64(1<<shift) >> 1
	wm.abcd[2] = int16(clipWmp(applySign64(int32((abs64(v1)+rnd)>>shift), v1)))
	v2 := int64(mat[3]) * int64(mat[4]) * int64(y)
	wm.abcd[3] = int16(clipWmp(mat[5] -
		applySign64(int32((abs64(v2)+rnd)>>shift), v2) - 0x10000))

	a := int32(wm.abcd[0])
	bb := int32(wm.abcd[1])
	g := int32(wm.abcd[2])
	dd := int32(wm.abcd[3])

	return 4*abs32(a)+7*abs32(bb) >= 0x10000 || 4*abs32(g)+4*abs32(dd) >= 0x10000
}

func getMultShiftNdiag(px int64, idet int32, shift int) int32 {
	v1 := px * int64(idet)
	v2 := applySign64(int32((abs64(v1)+(int64(1)<<shift)>>1)>>shift), v1)

	return clip(v2, -0x1fff, 0x1fff)
}

func getMultShiftDiag(px int64, idet int32, shift int) int32 {
	v1 := px * int64(idet)
	v2 := applySign64(int32((abs64(v1)+(int64(1)<<shift)>>1)>>shift), v1)

	return clip(v2, 0xe001, 0x11fff)
}

func setAffineMv2d(bw4, bh4 int, m mv, wm *warpedMotionParams, bx4, by4 int) {
	mat := &wm.matrix
	rsuy := int32(2*bh4 - 1)
	rsux := int32(2*bw4 - 1)
	isuy := int32(by4)*4 + rsuy
	isux := int32(bx4)*4 + rsux

	mat[0] = clip(int32(m.x)*0x2000-(isux*(mat[2]-0x10000)+isuy*mat[3]),
		-0x800000, 0x7fffff)
	mat[1] = clip(int32(m.y)*0x2000-(isux*mat[4]+isuy*(mat[5]-0x10000)),
		-0x800000, 0x7fffff)
}

func findAffineInt(pts [][2][2]int32, np, bw4, bh4 int, m mv,
	wm *warpedMotionParams, bx4, by4 int,
) bool {
	mat := &wm.matrix
	var a [2][2]int32
	var bx, by [2]int32
	rsuy := int32(2*bh4 - 1)
	rsux := int32(2*bw4 - 1)
	suy := rsuy * 8
	sux := rsux * 8
	duy := suy + int32(m.y)
	dux := sux + int32(m.x)
	isuy := int32(by4)*4 + rsuy
	isux := int32(bx4)*4 + rsux

	for i := range np {
		dx := pts[i][1][0] - dux
		dy := pts[i][1][1] - duy
		sx := pts[i][0][0] - sux
		sy := pts[i][0][1] - suy
		if abs32(sx-dx) < 256 && abs32(sy-dy) < 256 {
			a[0][0] += (sx*sx)>>2 + sx*2 + 8
			a[0][1] += (sx*sy)>>2 + sx + sy + 4
			a[1][1] += (sy*sy)>>2 + sy*2 + 8
			bx[0] += (sx*dx)>>2 + sx + dx + 8
			bx[1] += (sy*dx)>>2 + sy + dx + 4
			by[0] += (sx*dy)>>2 + sx + dy + 4
			by[1] += (sy*dy)>>2 + sy + dy + 8
		}
	}

	det := int64(a[0][0])*int64(a[1][1]) - int64(a[0][1])*int64(a[0][1])
	if det == 0 {
		return true
	}
	div, shift := resolveDivisor64(uint64(abs64(det)))
	idet := applySign64(div, det)
	shift -= 16
	if shift < 0 {
		idet <<= -shift
		shift = 0
	}

	mat[2] = getMultShiftDiag(int64(a[1][1])*int64(bx[0])-
		int64(a[0][1])*int64(bx[1]), idet, shift)
	mat[3] = getMultShiftNdiag(int64(a[0][0])*int64(bx[1])-
		int64(a[0][1])*int64(bx[0]), idet, shift)
	mat[4] = getMultShiftNdiag(int64(a[1][1])*int64(by[0])-
		int64(a[0][1])*int64(by[1]), idet, shift)
	mat[5] = getMultShiftDiag(int64(a[0][0])*int64(by[1])-
		int64(a[0][1])*int64(by[0]), idet, shift)

	mat[0] = clip(int32(m.x)*0x2000-(isux*(mat[2]-0x10000)+isuy*mat[3]),
		-0x800000, 0x7fffff)
	mat[1] = clip(int32(m.y)*0x2000-(isux*mat[4]+isuy*(mat[5]-0x10000)),
		-0x800000, 0x7fffff)

	return false
}
