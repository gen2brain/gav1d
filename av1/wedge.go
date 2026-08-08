package av1

const (
	wedgeHorizontal = iota
	wedgeVertical
	wedgeOblique27
	wedgeOblique63
	wedgeOblique117
	wedgeOblique153
	nWedgeDirections
)

type wedgeCode struct {
	direction, xOffset, yOffset uint8
}

var wedgeCodebook16Hgtw = [16]wedgeCode{
	{wedgeOblique27, 4, 4}, {wedgeOblique63, 4, 4},
	{wedgeOblique117, 4, 4}, {wedgeOblique153, 4, 4},
	{wedgeHorizontal, 4, 2}, {wedgeHorizontal, 4, 4},
	{wedgeHorizontal, 4, 6}, {wedgeVertical, 4, 4},
	{wedgeOblique27, 4, 2}, {wedgeOblique27, 4, 6},
	{wedgeOblique153, 4, 2}, {wedgeOblique153, 4, 6},
	{wedgeOblique63, 2, 4}, {wedgeOblique63, 6, 4},
	{wedgeOblique117, 2, 4}, {wedgeOblique117, 6, 4},
}

var wedgeCodebook16Hltw = [16]wedgeCode{
	{wedgeOblique27, 4, 4}, {wedgeOblique63, 4, 4},
	{wedgeOblique117, 4, 4}, {wedgeOblique153, 4, 4},
	{wedgeVertical, 2, 4}, {wedgeVertical, 4, 4},
	{wedgeVertical, 6, 4}, {wedgeHorizontal, 4, 4},
	{wedgeOblique27, 4, 2}, {wedgeOblique27, 4, 6},
	{wedgeOblique153, 4, 2}, {wedgeOblique153, 4, 6},
	{wedgeOblique63, 2, 4}, {wedgeOblique63, 6, 4},
	{wedgeOblique117, 2, 4}, {wedgeOblique117, 6, 4},
}

var wedgeCodebook16Heqw = [16]wedgeCode{
	{wedgeOblique27, 4, 4}, {wedgeOblique63, 4, 4},
	{wedgeOblique117, 4, 4}, {wedgeOblique153, 4, 4},
	{wedgeHorizontal, 4, 2}, {wedgeHorizontal, 4, 6},
	{wedgeVertical, 2, 4}, {wedgeVertical, 6, 4},
	{wedgeOblique27, 4, 2}, {wedgeOblique27, 4, 6},
	{wedgeOblique153, 4, 2}, {wedgeOblique153, 4, 6},
	{wedgeOblique63, 2, 4}, {wedgeOblique63, 6, 4},
	{wedgeOblique117, 2, 4}, {wedgeOblique117, 6, 4},
}

var (
	wedgeMasks [3][bs8x8 - bs32x32 + 1][2][16][]uint8
	iiMasks    [3][bs8x8 - bs32x32 + 1][nInterIntraPredModes][]uint8
)

func insertBorder(dst, src []uint8, ctr int) {
	if ctr > 4 {
		memset(dst[:ctr-4], 0)
	}
	copy(dst[max(ctr, 4)-4:], src[max(4-ctr, 0):min(64-ctr, 8)])
	if ctr < 64-4 {
		memset(dst[ctr+4:64], 64)
	}
}

func wedgeTranspose(dst, src []uint8) {
	for y, yOff := 0, 0; y < 64; y, yOff = y+1, yOff+64 {
		for x, xOff := 0, 0; x < 64; x, xOff = x+1, xOff+64 {
			dst[xOff+y] = src[yOff+x]
		}
	}
}

func wedgeHflip(dst, src []uint8) {
	for y, yOff := 0, 0; y < 64; y, yOff = y+1, yOff+64 {
		for x := range 64 {
			dst[yOff+64-1-x] = src[yOff+x]
		}
	}
}

func wedgeCopy2d(dst, src []uint8, sign, w, h, xOff, yOff int) {
	s := yOff*64 + xOff
	d := 0
	for range h {
		if sign != 0 {
			for x := range w {
				dst[d+x] = 64 - src[s+x]
			}
		} else {
			copy(dst[d:d+w], src[s:s+w])
		}
		s += 64
		d += w
	}
}

func wedgeInitChroma(chroma, luma []uint8, sign, w, h, ssVer int) {
	c, l := 0, 0
	for y := 0; y < h; y += 1 + ssVer {
		for x := 0; x < w; x += 2 {
			sum := int(luma[l+x]) + int(luma[l+x+1]) + 1
			if ssVer != 0 {
				sum += int(luma[l+w+x]) + int(luma[l+w+x+1]) + 1
			}
			chroma[c+(x>>1)] = uint8((sum - sign) >> (1 + ssVer))
		}
		l += w << ssVer
		c += w >> 1
	}
}

func wedgeFill2d16x2(w, h, bs int, master *[6][64 * 64]uint8, cb *[16]wedgeCode,
	signs uint32,
) {
	for n := range 16 {
		sign := int(signs & 1)

		m444 := make([]uint8, w*h)
		wedgeCopy2d(m444, master[cb[n].direction][:], sign, w, h,
			32-(w*int(cb[n].xOffset)>>3), 32-(h*int(cb[n].yOffset)>>3))

		wedgeMasks[0][bs][0][n] = m444
		wedgeMasks[0][bs][1][n] = m444

		for _, ss := range [2]struct{ layout, ssVer int }{{1, 0}, {2, 1}} {
			sz := w * h >> (1 + ss.ssVer)
			a := make([]uint8, sz)
			b := make([]uint8, sz)
			wedgeInitChroma(a, m444, 0, w, h, ss.ssVer)
			wedgeInitChroma(b, m444, 1, w, h, ss.ssVer)
			wedgeMasks[ss.layout][bs][0][n] = a
			wedgeMasks[ss.layout][bs][1][n] = b
		}

		signs >>= 1
	}
}

var iiWeights1d = [32]uint8{
	60, 52, 45, 39, 34, 30, 26, 22, 19, 17, 15, 13, 11, 10, 8, 7,
	6, 6, 5, 4, 4, 3, 3, 2, 2, 2, 2, 1, 1, 1, 1, 1,
}

func buildNondcIIMasks(w, h, step int) [3][]uint8 {
	maskV := make([]uint8, w*h)
	maskH := make([]uint8, w*h)
	maskSm := make([]uint8, w*h)

	for y, off := 0, 0; y < h; y, off = y+1, off+w {
		memset(maskV[off:off+w], iiWeights1d[y*step])
		for x := range w {
			maskSm[off+x] = iiWeights1d[min(x, y)*step]
			maskH[off+x] = iiWeights1d[x*step]
		}
	}

	return [3][]uint8{maskV, maskH, maskSm}
}

func initIIWedgeMasks() {
	const (
		lineOdd = iota
		lineEven
		lineVert
	)
	masterBorder := [3][8]uint8{
		lineOdd:  {1, 2, 6, 18, 37, 53, 60, 63},
		lineEven: {1, 4, 11, 27, 46, 58, 62, 63},
		lineVert: {0, 2, 7, 21, 43, 57, 62, 64},
	}
	var master [6][64 * 64]uint8

	for y, off := 0, 0; y < 64; y, off = y+1, off+64 {
		insertBorder(master[wedgeVertical][off:], masterBorder[lineVert][:], 32)
	}
	for y, off, ctr := 0, 0, 48; y < 64; y, off, ctr = y+2, off+128, ctr-1 {
		insertBorder(master[wedgeOblique63][off:], masterBorder[lineEven][:], ctr)
		insertBorder(master[wedgeOblique63][off+64:], masterBorder[lineOdd][:], ctr-1)
	}

	wedgeTranspose(master[wedgeOblique27][:], master[wedgeOblique63][:])
	wedgeTranspose(master[wedgeHorizontal][:], master[wedgeVertical][:])
	wedgeHflip(master[wedgeOblique117][:], master[wedgeOblique63][:])
	wedgeHflip(master[wedgeOblique153][:], master[wedgeOblique27][:])

	type fillSpec struct {
		w, h  int
		bs    int
		cb    *[16]wedgeCode
		signs uint32
	}
	for _, s := range []fillSpec{
		{32, 32, bs32x32 - bs32x32, &wedgeCodebook16Heqw, 0x7bfb},
		{32, 16, bs32x16 - bs32x32, &wedgeCodebook16Hltw, 0x7beb},
		{32, 8, bs32x8 - bs32x32, &wedgeCodebook16Hltw, 0x6beb},
		{16, 32, bs16x32 - bs32x32, &wedgeCodebook16Hgtw, 0x7beb},
		{16, 16, bs16x16 - bs32x32, &wedgeCodebook16Heqw, 0x7bfb},
		{16, 8, bs16x8 - bs32x32, &wedgeCodebook16Hltw, 0x7beb},
		{8, 32, bs8x32 - bs32x32, &wedgeCodebook16Hgtw, 0x7aeb},
		{8, 16, bs8x16 - bs32x32, &wedgeCodebook16Hgtw, 0x7beb},
		{8, 8, bs8x8 - bs32x32, &wedgeCodebook16Heqw, 0x7bfb},
	} {
		wedgeFill2d16x2(s.w, s.h, s.bs, &master, s.cb, s.signs)
	}

	iiDc := make([]uint8, 32*32)
	memset(iiDc, 32)

	nondc := map[[2]int][3][]uint8{}
	for _, s := range [][3]int{
		{32, 32, 1}, {16, 32, 1}, {16, 16, 2}, {8, 32, 1}, {8, 16, 2},
		{8, 8, 4}, {4, 16, 2}, {4, 8, 4}, {4, 4, 8},
	} {
		nondc[[2]int{s[0], s[1]}] = buildNondcIIMasks(s[0], s[1], s[2])
	}

	for _, s := range [][7]int{
		{bs32x32, 32, 32, 16, 32, 16, 16},
		{bs32x16, 32, 32, 16, 16, 16, 16},
		{bs16x32, 16, 32, 8, 32, 8, 16},
		{bs16x16, 16, 16, 8, 16, 8, 8},
		{bs16x8, 16, 16, 8, 8, 8, 8},
		{bs8x16, 8, 16, 4, 16, 4, 8},
		{bs8x8, 8, 8, 4, 8, 4, 4},
	} {
		bs := s[0] - bs32x32
		for c := range 3 {
			iiMasks[c][bs][iiDcPred] = iiDc
		}
		for p := range 3 {
			iiMasks[0][bs][p+1] = nondc[[2]int{s[1], s[2]}][p]
			iiMasks[1][bs][p+1] = nondc[[2]int{s[3], s[4]}][p]
			iiMasks[2][bs][p+1] = nondc[[2]int{s[5], s[6]}][p]
		}
	}
}

func init() {
	initIIWedgeMasks()
}
