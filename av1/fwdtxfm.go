package av1

const (
	cosBitMin    = 10
	newSqrt2Bits = 12
	newSqrt2     = 5793
)

var cospiArr = [4][64]int32{
	{1024, 1024, 1023, 1021, 1019, 1016, 1013, 1009, 1004, 999, 993, 987, 980, 972, 964, 955, 946, 936, 926, 915, 903, 891, 878, 865, 851, 837, 822, 807, 792, 775, 759, 742, 724, 706, 688, 669, 650, 630, 610, 590, 569, 548, 526, 505, 483, 460, 438, 415, 392, 369, 345, 321, 297, 273, 249, 224, 200, 175, 150, 125, 100, 75, 50, 25},
	{2048, 2047, 2046, 2042, 2038, 2033, 2026, 2018, 2009, 1998, 1987, 1974, 1960, 1945, 1928, 1911, 1892, 1872, 1851, 1829, 1806, 1782, 1757, 1730, 1703, 1674, 1645, 1615, 1583, 1551, 1517, 1483, 1448, 1412, 1375, 1338, 1299, 1260, 1220, 1179, 1138, 1096, 1053, 1009, 965, 921, 876, 830, 784, 737, 690, 642, 595, 546, 498, 449, 400, 350, 301, 251, 201, 151, 100, 50},
	{4096, 4095, 4091, 4085, 4076, 4065, 4052, 4036, 4017, 3996, 3973, 3948, 3920, 3889, 3857, 3822, 3784, 3745, 3703, 3659, 3612, 3564, 3513, 3461, 3406, 3349, 3290, 3229, 3166, 3102, 3035, 2967, 2896, 2824, 2751, 2675, 2598, 2520, 2440, 2359, 2276, 2191, 2106, 2019, 1931, 1842, 1751, 1660, 1567, 1474, 1380, 1285, 1189, 1092, 995, 897, 799, 700, 601, 501, 401, 301, 201, 101},
	{8192, 8190, 8182, 8170, 8153, 8130, 8103, 8071, 8035, 7993, 7946, 7895, 7839, 7779, 7713, 7643, 7568, 7489, 7405, 7317, 7225, 7128, 7027, 6921, 6811, 6698, 6580, 6458, 6333, 6203, 6070, 5933, 5793, 5649, 5501, 5351, 5197, 5040, 4880, 4717, 4551, 4383, 4212, 4038, 3862, 3683, 3503, 3320, 3135, 2948, 2760, 2570, 2378, 2185, 1990, 1795, 1598, 1401, 1202, 1003, 803, 603, 402, 201},
}

var sinpiArr = [4][5]int32{
	{0, 330, 621, 836, 951},
	{0, 660, 1241, 1672, 1901},
	{0, 1321, 2482, 3344, 3803},
	{0, 2642, 4964, 6689, 7606},
}

var fwdShift = [nRectTxSizes][3]int8{
	tx4x4:    {2, 0, 0},
	tx8x8:    {2, -1, 0},
	tx16x16:  {2, -2, 0},
	tx32x32:  {2, -4, 0},
	tx64x64:  {0, -2, -2},
	rtx4x8:   {2, -1, 0},
	rtx8x4:   {2, -1, 0},
	rtx8x16:  {2, -2, 0},
	rtx16x8:  {2, -2, 0},
	rtx16x32: {2, -4, 0},
	rtx32x16: {2, -4, 0},
	rtx32x64: {0, -2, -2},
	rtx64x32: {2, -4, -2},
	rtx4x16:  {2, -1, 0},
	rtx16x4:  {2, -1, 0},
	rtx8x32:  {2, -2, 0},
	rtx32x8:  {2, -2, 0},
	rtx16x64: {0, -2, 0},
	rtx64x16: {2, -4, 0},
}

var fwdCosBitCol = [5][5]int8{
	{13, 13, 13, 0, 0},
	{13, 13, 13, 12, 0},
	{13, 13, 13, 12, 13},
	{0, 13, 13, 12, 13},
	{0, 0, 13, 12, 13},
}

var fwdCosBitRow = [5][5]int8{
	{13, 13, 12, 0, 0},
	{13, 13, 13, 12, 0},
	{13, 13, 12, 13, 12},
	{0, 12, 13, 12, 11},
	{0, 0, 12, 11, 10},
}

func roundShift(v int64, bit int) int32 {
	return int32((v + 1<<uint(bit-1)) >> uint(bit))
}

func halfBtf(w0, in0, w1, in1 int32, bit int) int32 {
	return roundShift(int64(w0)*int64(in0)+int64(w1)*int64(in1), bit)
}

func roundShiftArray(a []int32, bit int) {
	switch {
	case bit > 0:
		for i, v := range a {
			a[i] = roundShift(int64(v), bit)
		}
	case bit < 0:
		for i, v := range a {
			a[i] = v << uint(-bit)
		}
	}
}

func fadst4(input, output []int32, cosBit int) {
	sinpi := &sinpiArr[cosBit-cosBitMin]
	x0, x1, x2, x3 := input[0], input[1], input[2], input[3]
	if x0|x1|x2|x3 == 0 {
		output[0], output[1], output[2], output[3] = 0, 0, 0, 0

		return
	}

	s0 := int64(sinpi[1]) * int64(x0)
	s1 := int64(sinpi[4]) * int64(x0)
	s2 := int64(sinpi[2]) * int64(x1)
	s3 := int64(sinpi[1]) * int64(x1)
	s4 := int64(sinpi[3]) * int64(x2)
	s5 := int64(sinpi[4]) * int64(x3)
	s6 := int64(sinpi[2]) * int64(x3)
	s7 := int64(x0) + int64(x1) - int64(x3)

	a0 := s0 + s2 + s5
	a1 := int64(sinpi[3]) * s7
	a2 := s1 - s3 + s6
	a3 := s4

	output[0] = roundShift(a0+a3, cosBit)
	output[1] = roundShift(a1, cosBit)
	output[2] = roundShift(a2-a3, cosBit)
	output[3] = roundShift(a2-a0+a3, cosBit)
}

func fidentity(n int, input, output []int32) {
	switch n {
	case 4:
		for i := range 4 {
			output[i] = roundShift(int64(input[i])*newSqrt2, newSqrt2Bits)
		}
	case 8:
		for i := range 8 {
			output[i] = input[i] * 2
		}
	case 16:
		for i := range 16 {
			output[i] = roundShift(int64(input[i])*2*newSqrt2, newSqrt2Bits)
		}
	case 32:
		for i := range 32 {
			output[i] = input[i] * 4
		}
	}
}

func fwd1d(t, n int, input, output []int32, cosBit int) {
	switch t {
	case tx1dDct:
		switch n {
		case 4:
			fdct4(input, output, cosBit)
		case 8:
			fdct8(input, output, cosBit)
		case 16:
			fdct16(input, output, cosBit)
		case 32:
			fdct32(input, output, cosBit)
		}
	case tx1dAdst, tx1dFlipadst:
		switch n {
		case 4:
			fadst4(input, output, cosBit)
		case 8:
			fadst8(input, output, cosBit)
		case 16:
			fadst16(input, output, cosBit)
		}
	default:
		fidentity(n, input, output)
	}
}

func txtpFlip(txtp int) (bool, bool) {
	var ud, lr bool
	switch txtp {
	case flipadstDct, flipadstAdst, vFlipadst:
		ud = true
	case dctFlipadst, adstFlipadst, hFlipadst:
		lr = true
	case flipadstFlipadst:
		ud, lr = true, true
	}

	return ud, lr
}

func rectLogRatio(w, h int) int {
	switch {
	case w == h:
		return 0
	case w == h*2, h == w*2:
		return 1
	}

	return 2
}

const fwdScratch = 32*32 + 64 + 64

func fwdWht4x4(coeff []int32, src []int32, srcStride int) {
	var t [16]int32
	for i := range 4 {
		a := src[0*srcStride+i] + src[1*srcStride+i]
		d := src[3*srcStride+i] - src[2*srcStride+i]
		e := (a - d) >> 1
		b := e - src[1*srcStride+i]
		c := e - src[2*srcStride+i]
		t[0*4+i], t[1*4+i], t[2*4+i], t[3*4+i] = a-c, c, d+b, b
	}
	for i := range 4 {
		r := t[i*4:]
		a := r[0] + r[1]
		d := r[3] - r[2]
		e := (a - d) >> 1
		b := e - r[1]
		c := e - r[2]
		coeff[0*4+i], coeff[1*4+i] = (a-c)*4, c*4
		coeff[2*4+i], coeff[3*4+i] = (d+b)*4, b*4
	}
}

func fwdTxfm(coeff []int32, src []int32, srcStride, tx, txtp int, tmp []int32) {
	if txtp == whtWht {
		fwdWht4x4(coeff, src, srcStride)

		return
	}
	switch tx {
	case tx4x4:
		if fwdTxfm4x4(coeff, src, srcStride, txtp) {
			return
		}
	case tx8x8:
		if fwdTxfm8x8(coeff, src, srcStride, txtp) {
			return
		}
	case rtx4x8:
		if fwdTxfm4x8(coeff, src, srcStride, txtp) {
			return
		}
	case rtx8x4:
		if fwdTxfm8x4(coeff, src, srcStride, txtp) {
			return
		}
	case rtx8x16:
		if fwdTxfm8x16(coeff, src, srcStride, txtp) {
			return
		}
	case rtx16x8:
		if fwdTxfm16x8(coeff, src, srcStride, txtp) {
			return
		}
	case tx16x16:
		if fwdTxfm16x16(coeff, src, srcStride, txtp) {
			return
		}
	case tx32x32:
		if fwdTxfm32x32(coeff, src, srcStride, txtp) {
			return
		}
	case rtx16x32:
		if fwdTxfm16x32(coeff, src, srcStride, txtp) {
			return
		}
	case rtx32x16:
		if fwdTxfm32x16(coeff, src, srcStride, txtp) {
			return
		}
	}
	fwdTxfmGo(coeff, src, srcStride, tx, txtp, tmp)
}

func fwdTxfmGo(coeff []int32, src []int32, srcStride, tx, txtp int, tmp []int32) {
	tDim := &txfmDimensions[tx]
	w, h := 4*int(tDim.w), 4*int(tDim.h)
	shift := &fwdShift[tx]
	cosBitCol := int(fwdCosBitCol[tDim.lw][tDim.lh])
	cosBitRow := int(fwdCosBitRow[tDim.lw][tDim.lh])
	colType := int(tx1dTypes[txtp][0])
	rowType := int(tx1dTypes[txtp][1])
	ud, lr := txtpFlip(txtp)
	rect := rectLogRatio(w, h)

	buf := tmp[:32*32]
	tmpIn := tmp[32*32 : 32*32+64]
	tmpOut := tmp[32*32+64:]

	up := uint(shift[0])
	for c := range w {
		if ud {
			for r := range h {
				tmpIn[r] = src[(h-r-1)*srcStride+c] << up
			}
		} else {
			for r := range h {
				tmpIn[r] = src[r*srcStride+c] << up
			}
		}
		fwd1d(colType, h, tmpIn[:h], tmpOut[:h], cosBitCol)
		x := c
		if lr {
			x = w - c - 1
		}
		for r := range h {
			buf[r*w+x] = tmpOut[r]
		}
	}

	sw, sh := min(w, 32), min(h, 32)
	down := -int(shift[1]) - int(shift[2])
	for r := range sh {
		fwd1d(rowType, w, buf[r*w:r*w+w], tmpOut[:w], cosBitRow)
		roundShiftArray(tmpOut[:sw], down)
		if rect == 1 {
			for c := range sw {
				coeff[c*sh+r] = roundShift(int64(tmpOut[c])*newSqrt2, newSqrt2Bits)
			}

			continue
		}
		for c := range sw {
			coeff[c*sh+r] = tmpOut[c]
		}
	}
}

func fwdTxfmSupported(tx int) bool {
	tDim := &txfmDimensions[tx]

	return tDim.w <= 8 && tDim.h <= 8
}

func fdct4(input, output []int32, cosBit int) {
	var step [4]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0] + input[3]
	bf1[1] = input[1] + input[2]
	bf1[2] = -input[2] + input[1]
	bf1[3] = -input[3] + input[0]
	bf0 = output
	bf1 = step[:]
	bf1[0] = halfBtf(cospi[32], bf0[0], cospi[32], bf0[1], cosBit)
	bf1[1] = halfBtf(-cospi[32], bf0[1], cospi[32], bf0[0], cosBit)
	bf1[2] = halfBtf(cospi[48], bf0[2], cospi[16], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[48], bf0[3], -cospi[16], bf0[2], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[2]
	bf1[2] = bf0[1]
	bf1[3] = bf0[3]
}

func fdct8(input, output []int32, cosBit int) {
	var step [8]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0] + input[7]
	bf1[1] = input[1] + input[6]
	bf1[2] = input[2] + input[5]
	bf1[3] = input[3] + input[4]
	bf1[4] = -input[4] + input[3]
	bf1[5] = -input[5] + input[2]
	bf1[6] = -input[6] + input[1]
	bf1[7] = -input[7] + input[0]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0] + bf0[3]
	bf1[1] = bf0[1] + bf0[2]
	bf1[2] = -bf0[2] + bf0[1]
	bf1[3] = -bf0[3] + bf0[0]
	bf1[4] = bf0[4]
	bf1[5] = halfBtf(-cospi[32], bf0[5], cospi[32], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[32], bf0[6], cospi[32], bf0[5], cosBit)
	bf1[7] = bf0[7]
	bf0 = step[:]
	bf1 = output
	bf1[0] = halfBtf(cospi[32], bf0[0], cospi[32], bf0[1], cosBit)
	bf1[1] = halfBtf(-cospi[32], bf0[1], cospi[32], bf0[0], cosBit)
	bf1[2] = halfBtf(cospi[48], bf0[2], cospi[16], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[48], bf0[3], -cospi[16], bf0[2], cosBit)
	bf1[4] = bf0[4] + bf0[5]
	bf1[5] = -bf0[5] + bf0[4]
	bf1[6] = -bf0[6] + bf0[7]
	bf1[7] = bf0[7] + bf0[6]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = halfBtf(cospi[56], bf0[4], cospi[8], bf0[7], cosBit)
	bf1[5] = halfBtf(cospi[24], bf0[5], cospi[40], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[24], bf0[6], -cospi[40], bf0[5], cosBit)
	bf1[7] = halfBtf(cospi[56], bf0[7], -cospi[8], bf0[4], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[4]
	bf1[2] = bf0[2]
	bf1[3] = bf0[6]
	bf1[4] = bf0[1]
	bf1[5] = bf0[5]
	bf1[6] = bf0[3]
	bf1[7] = bf0[7]
}

func fdct16(input, output []int32, cosBit int) {
	var step [16]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0] + input[15]
	bf1[1] = input[1] + input[14]
	bf1[2] = input[2] + input[13]
	bf1[3] = input[3] + input[12]
	bf1[4] = input[4] + input[11]
	bf1[5] = input[5] + input[10]
	bf1[6] = input[6] + input[9]
	bf1[7] = input[7] + input[8]
	bf1[8] = -input[8] + input[7]
	bf1[9] = -input[9] + input[6]
	bf1[10] = -input[10] + input[5]
	bf1[11] = -input[11] + input[4]
	bf1[12] = -input[12] + input[3]
	bf1[13] = -input[13] + input[2]
	bf1[14] = -input[14] + input[1]
	bf1[15] = -input[15] + input[0]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0] + bf0[7]
	bf1[1] = bf0[1] + bf0[6]
	bf1[2] = bf0[2] + bf0[5]
	bf1[3] = bf0[3] + bf0[4]
	bf1[4] = -bf0[4] + bf0[3]
	bf1[5] = -bf0[5] + bf0[2]
	bf1[6] = -bf0[6] + bf0[1]
	bf1[7] = -bf0[7] + bf0[0]
	bf1[8] = bf0[8]
	bf1[9] = bf0[9]
	bf1[10] = halfBtf(-cospi[32], bf0[10], cospi[32], bf0[13], cosBit)
	bf1[11] = halfBtf(-cospi[32], bf0[11], cospi[32], bf0[12], cosBit)
	bf1[12] = halfBtf(cospi[32], bf0[12], cospi[32], bf0[11], cosBit)
	bf1[13] = halfBtf(cospi[32], bf0[13], cospi[32], bf0[10], cosBit)
	bf1[14] = bf0[14]
	bf1[15] = bf0[15]
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[3]
	bf1[1] = bf0[1] + bf0[2]
	bf1[2] = -bf0[2] + bf0[1]
	bf1[3] = -bf0[3] + bf0[0]
	bf1[4] = bf0[4]
	bf1[5] = halfBtf(-cospi[32], bf0[5], cospi[32], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[32], bf0[6], cospi[32], bf0[5], cosBit)
	bf1[7] = bf0[7]
	bf1[8] = bf0[8] + bf0[11]
	bf1[9] = bf0[9] + bf0[10]
	bf1[10] = -bf0[10] + bf0[9]
	bf1[11] = -bf0[11] + bf0[8]
	bf1[12] = -bf0[12] + bf0[15]
	bf1[13] = -bf0[13] + bf0[14]
	bf1[14] = bf0[14] + bf0[13]
	bf1[15] = bf0[15] + bf0[12]
	bf0 = output
	bf1 = step[:]
	bf1[0] = halfBtf(cospi[32], bf0[0], cospi[32], bf0[1], cosBit)
	bf1[1] = halfBtf(-cospi[32], bf0[1], cospi[32], bf0[0], cosBit)
	bf1[2] = halfBtf(cospi[48], bf0[2], cospi[16], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[48], bf0[3], -cospi[16], bf0[2], cosBit)
	bf1[4] = bf0[4] + bf0[5]
	bf1[5] = -bf0[5] + bf0[4]
	bf1[6] = -bf0[6] + bf0[7]
	bf1[7] = bf0[7] + bf0[6]
	bf1[8] = bf0[8]
	bf1[9] = halfBtf(-cospi[16], bf0[9], cospi[48], bf0[14], cosBit)
	bf1[10] = halfBtf(-cospi[48], bf0[10], -cospi[16], bf0[13], cosBit)
	bf1[11] = bf0[11]
	bf1[12] = bf0[12]
	bf1[13] = halfBtf(cospi[48], bf0[13], -cospi[16], bf0[10], cosBit)
	bf1[14] = halfBtf(cospi[16], bf0[14], cospi[48], bf0[9], cosBit)
	bf1[15] = bf0[15]
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = halfBtf(cospi[56], bf0[4], cospi[8], bf0[7], cosBit)
	bf1[5] = halfBtf(cospi[24], bf0[5], cospi[40], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[24], bf0[6], -cospi[40], bf0[5], cosBit)
	bf1[7] = halfBtf(cospi[56], bf0[7], -cospi[8], bf0[4], cosBit)
	bf1[8] = bf0[8] + bf0[9]
	bf1[9] = -bf0[9] + bf0[8]
	bf1[10] = -bf0[10] + bf0[11]
	bf1[11] = bf0[11] + bf0[10]
	bf1[12] = bf0[12] + bf0[13]
	bf1[13] = -bf0[13] + bf0[12]
	bf1[14] = -bf0[14] + bf0[15]
	bf1[15] = bf0[15] + bf0[14]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = bf0[6]
	bf1[7] = bf0[7]
	bf1[8] = halfBtf(cospi[60], bf0[8], cospi[4], bf0[15], cosBit)
	bf1[9] = halfBtf(cospi[28], bf0[9], cospi[36], bf0[14], cosBit)
	bf1[10] = halfBtf(cospi[44], bf0[10], cospi[20], bf0[13], cosBit)
	bf1[11] = halfBtf(cospi[12], bf0[11], cospi[52], bf0[12], cosBit)
	bf1[12] = halfBtf(cospi[12], bf0[12], -cospi[52], bf0[11], cosBit)
	bf1[13] = halfBtf(cospi[44], bf0[13], -cospi[20], bf0[10], cosBit)
	bf1[14] = halfBtf(cospi[28], bf0[14], -cospi[36], bf0[9], cosBit)
	bf1[15] = halfBtf(cospi[60], bf0[15], -cospi[4], bf0[8], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[8]
	bf1[2] = bf0[4]
	bf1[3] = bf0[12]
	bf1[4] = bf0[2]
	bf1[5] = bf0[10]
	bf1[6] = bf0[6]
	bf1[7] = bf0[14]
	bf1[8] = bf0[1]
	bf1[9] = bf0[9]
	bf1[10] = bf0[5]
	bf1[11] = bf0[13]
	bf1[12] = bf0[3]
	bf1[13] = bf0[11]
	bf1[14] = bf0[7]
	bf1[15] = bf0[15]
}

func fdct32(input, output []int32, cosBit int) {
	var step [32]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0] + input[31]
	bf1[1] = input[1] + input[30]
	bf1[2] = input[2] + input[29]
	bf1[3] = input[3] + input[28]
	bf1[4] = input[4] + input[27]
	bf1[5] = input[5] + input[26]
	bf1[6] = input[6] + input[25]
	bf1[7] = input[7] + input[24]
	bf1[8] = input[8] + input[23]
	bf1[9] = input[9] + input[22]
	bf1[10] = input[10] + input[21]
	bf1[11] = input[11] + input[20]
	bf1[12] = input[12] + input[19]
	bf1[13] = input[13] + input[18]
	bf1[14] = input[14] + input[17]
	bf1[15] = input[15] + input[16]
	bf1[16] = -input[16] + input[15]
	bf1[17] = -input[17] + input[14]
	bf1[18] = -input[18] + input[13]
	bf1[19] = -input[19] + input[12]
	bf1[20] = -input[20] + input[11]
	bf1[21] = -input[21] + input[10]
	bf1[22] = -input[22] + input[9]
	bf1[23] = -input[23] + input[8]
	bf1[24] = -input[24] + input[7]
	bf1[25] = -input[25] + input[6]
	bf1[26] = -input[26] + input[5]
	bf1[27] = -input[27] + input[4]
	bf1[28] = -input[28] + input[3]
	bf1[29] = -input[29] + input[2]
	bf1[30] = -input[30] + input[1]
	bf1[31] = -input[31] + input[0]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0] + bf0[15]
	bf1[1] = bf0[1] + bf0[14]
	bf1[2] = bf0[2] + bf0[13]
	bf1[3] = bf0[3] + bf0[12]
	bf1[4] = bf0[4] + bf0[11]
	bf1[5] = bf0[5] + bf0[10]
	bf1[6] = bf0[6] + bf0[9]
	bf1[7] = bf0[7] + bf0[8]
	bf1[8] = -bf0[8] + bf0[7]
	bf1[9] = -bf0[9] + bf0[6]
	bf1[10] = -bf0[10] + bf0[5]
	bf1[11] = -bf0[11] + bf0[4]
	bf1[12] = -bf0[12] + bf0[3]
	bf1[13] = -bf0[13] + bf0[2]
	bf1[14] = -bf0[14] + bf0[1]
	bf1[15] = -bf0[15] + bf0[0]
	bf1[16] = bf0[16]
	bf1[17] = bf0[17]
	bf1[18] = bf0[18]
	bf1[19] = bf0[19]
	bf1[20] = halfBtf(-cospi[32], bf0[20], cospi[32], bf0[27], cosBit)
	bf1[21] = halfBtf(-cospi[32], bf0[21], cospi[32], bf0[26], cosBit)
	bf1[22] = halfBtf(-cospi[32], bf0[22], cospi[32], bf0[25], cosBit)
	bf1[23] = halfBtf(-cospi[32], bf0[23], cospi[32], bf0[24], cosBit)
	bf1[24] = halfBtf(cospi[32], bf0[24], cospi[32], bf0[23], cosBit)
	bf1[25] = halfBtf(cospi[32], bf0[25], cospi[32], bf0[22], cosBit)
	bf1[26] = halfBtf(cospi[32], bf0[26], cospi[32], bf0[21], cosBit)
	bf1[27] = halfBtf(cospi[32], bf0[27], cospi[32], bf0[20], cosBit)
	bf1[28] = bf0[28]
	bf1[29] = bf0[29]
	bf1[30] = bf0[30]
	bf1[31] = bf0[31]
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[7]
	bf1[1] = bf0[1] + bf0[6]
	bf1[2] = bf0[2] + bf0[5]
	bf1[3] = bf0[3] + bf0[4]
	bf1[4] = -bf0[4] + bf0[3]
	bf1[5] = -bf0[5] + bf0[2]
	bf1[6] = -bf0[6] + bf0[1]
	bf1[7] = -bf0[7] + bf0[0]
	bf1[8] = bf0[8]
	bf1[9] = bf0[9]
	bf1[10] = halfBtf(-cospi[32], bf0[10], cospi[32], bf0[13], cosBit)
	bf1[11] = halfBtf(-cospi[32], bf0[11], cospi[32], bf0[12], cosBit)
	bf1[12] = halfBtf(cospi[32], bf0[12], cospi[32], bf0[11], cosBit)
	bf1[13] = halfBtf(cospi[32], bf0[13], cospi[32], bf0[10], cosBit)
	bf1[14] = bf0[14]
	bf1[15] = bf0[15]
	bf1[16] = bf0[16] + bf0[23]
	bf1[17] = bf0[17] + bf0[22]
	bf1[18] = bf0[18] + bf0[21]
	bf1[19] = bf0[19] + bf0[20]
	bf1[20] = -bf0[20] + bf0[19]
	bf1[21] = -bf0[21] + bf0[18]
	bf1[22] = -bf0[22] + bf0[17]
	bf1[23] = -bf0[23] + bf0[16]
	bf1[24] = -bf0[24] + bf0[31]
	bf1[25] = -bf0[25] + bf0[30]
	bf1[26] = -bf0[26] + bf0[29]
	bf1[27] = -bf0[27] + bf0[28]
	bf1[28] = bf0[28] + bf0[27]
	bf1[29] = bf0[29] + bf0[26]
	bf1[30] = bf0[30] + bf0[25]
	bf1[31] = bf0[31] + bf0[24]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0] + bf0[3]
	bf1[1] = bf0[1] + bf0[2]
	bf1[2] = -bf0[2] + bf0[1]
	bf1[3] = -bf0[3] + bf0[0]
	bf1[4] = bf0[4]
	bf1[5] = halfBtf(-cospi[32], bf0[5], cospi[32], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[32], bf0[6], cospi[32], bf0[5], cosBit)
	bf1[7] = bf0[7]
	bf1[8] = bf0[8] + bf0[11]
	bf1[9] = bf0[9] + bf0[10]
	bf1[10] = -bf0[10] + bf0[9]
	bf1[11] = -bf0[11] + bf0[8]
	bf1[12] = -bf0[12] + bf0[15]
	bf1[13] = -bf0[13] + bf0[14]
	bf1[14] = bf0[14] + bf0[13]
	bf1[15] = bf0[15] + bf0[12]
	bf1[16] = bf0[16]
	bf1[17] = bf0[17]
	bf1[18] = halfBtf(-cospi[16], bf0[18], cospi[48], bf0[29], cosBit)
	bf1[19] = halfBtf(-cospi[16], bf0[19], cospi[48], bf0[28], cosBit)
	bf1[20] = halfBtf(-cospi[48], bf0[20], -cospi[16], bf0[27], cosBit)
	bf1[21] = halfBtf(-cospi[48], bf0[21], -cospi[16], bf0[26], cosBit)
	bf1[22] = bf0[22]
	bf1[23] = bf0[23]
	bf1[24] = bf0[24]
	bf1[25] = bf0[25]
	bf1[26] = halfBtf(cospi[48], bf0[26], -cospi[16], bf0[21], cosBit)
	bf1[27] = halfBtf(cospi[48], bf0[27], -cospi[16], bf0[20], cosBit)
	bf1[28] = halfBtf(cospi[16], bf0[28], cospi[48], bf0[19], cosBit)
	bf1[29] = halfBtf(cospi[16], bf0[29], cospi[48], bf0[18], cosBit)
	bf1[30] = bf0[30]
	bf1[31] = bf0[31]
	bf0 = step[:]
	bf1 = output
	bf1[0] = halfBtf(cospi[32], bf0[0], cospi[32], bf0[1], cosBit)
	bf1[1] = halfBtf(-cospi[32], bf0[1], cospi[32], bf0[0], cosBit)
	bf1[2] = halfBtf(cospi[48], bf0[2], cospi[16], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[48], bf0[3], -cospi[16], bf0[2], cosBit)
	bf1[4] = bf0[4] + bf0[5]
	bf1[5] = -bf0[5] + bf0[4]
	bf1[6] = -bf0[6] + bf0[7]
	bf1[7] = bf0[7] + bf0[6]
	bf1[8] = bf0[8]
	bf1[9] = halfBtf(-cospi[16], bf0[9], cospi[48], bf0[14], cosBit)
	bf1[10] = halfBtf(-cospi[48], bf0[10], -cospi[16], bf0[13], cosBit)
	bf1[11] = bf0[11]
	bf1[12] = bf0[12]
	bf1[13] = halfBtf(cospi[48], bf0[13], -cospi[16], bf0[10], cosBit)
	bf1[14] = halfBtf(cospi[16], bf0[14], cospi[48], bf0[9], cosBit)
	bf1[15] = bf0[15]
	bf1[16] = bf0[16] + bf0[19]
	bf1[17] = bf0[17] + bf0[18]
	bf1[18] = -bf0[18] + bf0[17]
	bf1[19] = -bf0[19] + bf0[16]
	bf1[20] = -bf0[20] + bf0[23]
	bf1[21] = -bf0[21] + bf0[22]
	bf1[22] = bf0[22] + bf0[21]
	bf1[23] = bf0[23] + bf0[20]
	bf1[24] = bf0[24] + bf0[27]
	bf1[25] = bf0[25] + bf0[26]
	bf1[26] = -bf0[26] + bf0[25]
	bf1[27] = -bf0[27] + bf0[24]
	bf1[28] = -bf0[28] + bf0[31]
	bf1[29] = -bf0[29] + bf0[30]
	bf1[30] = bf0[30] + bf0[29]
	bf1[31] = bf0[31] + bf0[28]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = halfBtf(cospi[56], bf0[4], cospi[8], bf0[7], cosBit)
	bf1[5] = halfBtf(cospi[24], bf0[5], cospi[40], bf0[6], cosBit)
	bf1[6] = halfBtf(cospi[24], bf0[6], -cospi[40], bf0[5], cosBit)
	bf1[7] = halfBtf(cospi[56], bf0[7], -cospi[8], bf0[4], cosBit)
	bf1[8] = bf0[8] + bf0[9]
	bf1[9] = -bf0[9] + bf0[8]
	bf1[10] = -bf0[10] + bf0[11]
	bf1[11] = bf0[11] + bf0[10]
	bf1[12] = bf0[12] + bf0[13]
	bf1[13] = -bf0[13] + bf0[12]
	bf1[14] = -bf0[14] + bf0[15]
	bf1[15] = bf0[15] + bf0[14]
	bf1[16] = bf0[16]
	bf1[17] = halfBtf(-cospi[8], bf0[17], cospi[56], bf0[30], cosBit)
	bf1[18] = halfBtf(-cospi[56], bf0[18], -cospi[8], bf0[29], cosBit)
	bf1[19] = bf0[19]
	bf1[20] = bf0[20]
	bf1[21] = halfBtf(-cospi[40], bf0[21], cospi[24], bf0[26], cosBit)
	bf1[22] = halfBtf(-cospi[24], bf0[22], -cospi[40], bf0[25], cosBit)
	bf1[23] = bf0[23]
	bf1[24] = bf0[24]
	bf1[25] = halfBtf(cospi[24], bf0[25], -cospi[40], bf0[22], cosBit)
	bf1[26] = halfBtf(cospi[40], bf0[26], cospi[24], bf0[21], cosBit)
	bf1[27] = bf0[27]
	bf1[28] = bf0[28]
	bf1[29] = halfBtf(cospi[56], bf0[29], -cospi[8], bf0[18], cosBit)
	bf1[30] = halfBtf(cospi[8], bf0[30], cospi[56], bf0[17], cosBit)
	bf1[31] = bf0[31]
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = bf0[6]
	bf1[7] = bf0[7]
	bf1[8] = halfBtf(cospi[60], bf0[8], cospi[4], bf0[15], cosBit)
	bf1[9] = halfBtf(cospi[28], bf0[9], cospi[36], bf0[14], cosBit)
	bf1[10] = halfBtf(cospi[44], bf0[10], cospi[20], bf0[13], cosBit)
	bf1[11] = halfBtf(cospi[12], bf0[11], cospi[52], bf0[12], cosBit)
	bf1[12] = halfBtf(cospi[12], bf0[12], -cospi[52], bf0[11], cosBit)
	bf1[13] = halfBtf(cospi[44], bf0[13], -cospi[20], bf0[10], cosBit)
	bf1[14] = halfBtf(cospi[28], bf0[14], -cospi[36], bf0[9], cosBit)
	bf1[15] = halfBtf(cospi[60], bf0[15], -cospi[4], bf0[8], cosBit)
	bf1[16] = bf0[16] + bf0[17]
	bf1[17] = -bf0[17] + bf0[16]
	bf1[18] = -bf0[18] + bf0[19]
	bf1[19] = bf0[19] + bf0[18]
	bf1[20] = bf0[20] + bf0[21]
	bf1[21] = -bf0[21] + bf0[20]
	bf1[22] = -bf0[22] + bf0[23]
	bf1[23] = bf0[23] + bf0[22]
	bf1[24] = bf0[24] + bf0[25]
	bf1[25] = -bf0[25] + bf0[24]
	bf1[26] = -bf0[26] + bf0[27]
	bf1[27] = bf0[27] + bf0[26]
	bf1[28] = bf0[28] + bf0[29]
	bf1[29] = -bf0[29] + bf0[28]
	bf1[30] = -bf0[30] + bf0[31]
	bf1[31] = bf0[31] + bf0[30]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = bf0[6]
	bf1[7] = bf0[7]
	bf1[8] = bf0[8]
	bf1[9] = bf0[9]
	bf1[10] = bf0[10]
	bf1[11] = bf0[11]
	bf1[12] = bf0[12]
	bf1[13] = bf0[13]
	bf1[14] = bf0[14]
	bf1[15] = bf0[15]
	bf1[16] = halfBtf(cospi[62], bf0[16], cospi[2], bf0[31], cosBit)
	bf1[17] = halfBtf(cospi[30], bf0[17], cospi[34], bf0[30], cosBit)
	bf1[18] = halfBtf(cospi[46], bf0[18], cospi[18], bf0[29], cosBit)
	bf1[19] = halfBtf(cospi[14], bf0[19], cospi[50], bf0[28], cosBit)
	bf1[20] = halfBtf(cospi[54], bf0[20], cospi[10], bf0[27], cosBit)
	bf1[21] = halfBtf(cospi[22], bf0[21], cospi[42], bf0[26], cosBit)
	bf1[22] = halfBtf(cospi[38], bf0[22], cospi[26], bf0[25], cosBit)
	bf1[23] = halfBtf(cospi[6], bf0[23], cospi[58], bf0[24], cosBit)
	bf1[24] = halfBtf(cospi[6], bf0[24], -cospi[58], bf0[23], cosBit)
	bf1[25] = halfBtf(cospi[38], bf0[25], -cospi[26], bf0[22], cosBit)
	bf1[26] = halfBtf(cospi[22], bf0[26], -cospi[42], bf0[21], cosBit)
	bf1[27] = halfBtf(cospi[54], bf0[27], -cospi[10], bf0[20], cosBit)
	bf1[28] = halfBtf(cospi[14], bf0[28], -cospi[50], bf0[19], cosBit)
	bf1[29] = halfBtf(cospi[46], bf0[29], -cospi[18], bf0[18], cosBit)
	bf1[30] = halfBtf(cospi[30], bf0[30], -cospi[34], bf0[17], cosBit)
	bf1[31] = halfBtf(cospi[62], bf0[31], -cospi[2], bf0[16], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0]
	bf1[1] = bf0[16]
	bf1[2] = bf0[8]
	bf1[3] = bf0[24]
	bf1[4] = bf0[4]
	bf1[5] = bf0[20]
	bf1[6] = bf0[12]
	bf1[7] = bf0[28]
	bf1[8] = bf0[2]
	bf1[9] = bf0[18]
	bf1[10] = bf0[10]
	bf1[11] = bf0[26]
	bf1[12] = bf0[6]
	bf1[13] = bf0[22]
	bf1[14] = bf0[14]
	bf1[15] = bf0[30]
	bf1[16] = bf0[1]
	bf1[17] = bf0[17]
	bf1[18] = bf0[9]
	bf1[19] = bf0[25]
	bf1[20] = bf0[5]
	bf1[21] = bf0[21]
	bf1[22] = bf0[13]
	bf1[23] = bf0[29]
	bf1[24] = bf0[3]
	bf1[25] = bf0[19]
	bf1[26] = bf0[11]
	bf1[27] = bf0[27]
	bf1[28] = bf0[7]
	bf1[29] = bf0[23]
	bf1[30] = bf0[15]
	bf1[31] = bf0[31]
}

func fadst8(input, output []int32, cosBit int) {
	var step [8]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0]
	bf1[1] = -input[7]
	bf1[2] = -input[3]
	bf1[3] = input[4]
	bf1[4] = -input[1]
	bf1[5] = input[6]
	bf1[6] = input[2]
	bf1[7] = -input[5]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = halfBtf(cospi[32], bf0[2], cospi[32], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[32], bf0[2], -cospi[32], bf0[3], cosBit)
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = halfBtf(cospi[32], bf0[6], cospi[32], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[32], bf0[6], -cospi[32], bf0[7], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[2]
	bf1[1] = bf0[1] + bf0[3]
	bf1[2] = bf0[0] - bf0[2]
	bf1[3] = bf0[1] - bf0[3]
	bf1[4] = bf0[4] + bf0[6]
	bf1[5] = bf0[5] + bf0[7]
	bf1[6] = bf0[4] - bf0[6]
	bf1[7] = bf0[5] - bf0[7]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = halfBtf(cospi[16], bf0[4], cospi[48], bf0[5], cosBit)
	bf1[5] = halfBtf(cospi[48], bf0[4], -cospi[16], bf0[5], cosBit)
	bf1[6] = halfBtf(-cospi[48], bf0[6], cospi[16], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[16], bf0[6], cospi[48], bf0[7], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[4]
	bf1[1] = bf0[1] + bf0[5]
	bf1[2] = bf0[2] + bf0[6]
	bf1[3] = bf0[3] + bf0[7]
	bf1[4] = bf0[0] - bf0[4]
	bf1[5] = bf0[1] - bf0[5]
	bf1[6] = bf0[2] - bf0[6]
	bf1[7] = bf0[3] - bf0[7]
	bf0 = output
	bf1 = step[:]
	bf1[0] = halfBtf(cospi[4], bf0[0], cospi[60], bf0[1], cosBit)
	bf1[1] = halfBtf(cospi[60], bf0[0], -cospi[4], bf0[1], cosBit)
	bf1[2] = halfBtf(cospi[20], bf0[2], cospi[44], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[44], bf0[2], -cospi[20], bf0[3], cosBit)
	bf1[4] = halfBtf(cospi[36], bf0[4], cospi[28], bf0[5], cosBit)
	bf1[5] = halfBtf(cospi[28], bf0[4], -cospi[36], bf0[5], cosBit)
	bf1[6] = halfBtf(cospi[52], bf0[6], cospi[12], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[12], bf0[6], -cospi[52], bf0[7], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[1]
	bf1[1] = bf0[6]
	bf1[2] = bf0[3]
	bf1[3] = bf0[4]
	bf1[4] = bf0[5]
	bf1[5] = bf0[2]
	bf1[6] = bf0[7]
	bf1[7] = bf0[0]
}

func fadst16(input, output []int32, cosBit int) {
	var step [16]int32
	cospi := cospiArr[cosBit-cosBitMin][:]
	var bf0, bf1 []int32
	bf1 = output
	bf1[0] = input[0]
	bf1[1] = -input[15]
	bf1[2] = -input[7]
	bf1[3] = input[8]
	bf1[4] = -input[3]
	bf1[5] = input[12]
	bf1[6] = input[4]
	bf1[7] = -input[11]
	bf1[8] = -input[1]
	bf1[9] = input[14]
	bf1[10] = input[6]
	bf1[11] = -input[9]
	bf1[12] = input[2]
	bf1[13] = -input[13]
	bf1[14] = -input[5]
	bf1[15] = input[10]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = halfBtf(cospi[32], bf0[2], cospi[32], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[32], bf0[2], -cospi[32], bf0[3], cosBit)
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = halfBtf(cospi[32], bf0[6], cospi[32], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[32], bf0[6], -cospi[32], bf0[7], cosBit)
	bf1[8] = bf0[8]
	bf1[9] = bf0[9]
	bf1[10] = halfBtf(cospi[32], bf0[10], cospi[32], bf0[11], cosBit)
	bf1[11] = halfBtf(cospi[32], bf0[10], -cospi[32], bf0[11], cosBit)
	bf1[12] = bf0[12]
	bf1[13] = bf0[13]
	bf1[14] = halfBtf(cospi[32], bf0[14], cospi[32], bf0[15], cosBit)
	bf1[15] = halfBtf(cospi[32], bf0[14], -cospi[32], bf0[15], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[2]
	bf1[1] = bf0[1] + bf0[3]
	bf1[2] = bf0[0] - bf0[2]
	bf1[3] = bf0[1] - bf0[3]
	bf1[4] = bf0[4] + bf0[6]
	bf1[5] = bf0[5] + bf0[7]
	bf1[6] = bf0[4] - bf0[6]
	bf1[7] = bf0[5] - bf0[7]
	bf1[8] = bf0[8] + bf0[10]
	bf1[9] = bf0[9] + bf0[11]
	bf1[10] = bf0[8] - bf0[10]
	bf1[11] = bf0[9] - bf0[11]
	bf1[12] = bf0[12] + bf0[14]
	bf1[13] = bf0[13] + bf0[15]
	bf1[14] = bf0[12] - bf0[14]
	bf1[15] = bf0[13] - bf0[15]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = halfBtf(cospi[16], bf0[4], cospi[48], bf0[5], cosBit)
	bf1[5] = halfBtf(cospi[48], bf0[4], -cospi[16], bf0[5], cosBit)
	bf1[6] = halfBtf(-cospi[48], bf0[6], cospi[16], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[16], bf0[6], cospi[48], bf0[7], cosBit)
	bf1[8] = bf0[8]
	bf1[9] = bf0[9]
	bf1[10] = bf0[10]
	bf1[11] = bf0[11]
	bf1[12] = halfBtf(cospi[16], bf0[12], cospi[48], bf0[13], cosBit)
	bf1[13] = halfBtf(cospi[48], bf0[12], -cospi[16], bf0[13], cosBit)
	bf1[14] = halfBtf(-cospi[48], bf0[14], cospi[16], bf0[15], cosBit)
	bf1[15] = halfBtf(cospi[16], bf0[14], cospi[48], bf0[15], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[4]
	bf1[1] = bf0[1] + bf0[5]
	bf1[2] = bf0[2] + bf0[6]
	bf1[3] = bf0[3] + bf0[7]
	bf1[4] = bf0[0] - bf0[4]
	bf1[5] = bf0[1] - bf0[5]
	bf1[6] = bf0[2] - bf0[6]
	bf1[7] = bf0[3] - bf0[7]
	bf1[8] = bf0[8] + bf0[12]
	bf1[9] = bf0[9] + bf0[13]
	bf1[10] = bf0[10] + bf0[14]
	bf1[11] = bf0[11] + bf0[15]
	bf1[12] = bf0[8] - bf0[12]
	bf1[13] = bf0[9] - bf0[13]
	bf1[14] = bf0[10] - bf0[14]
	bf1[15] = bf0[11] - bf0[15]
	bf0 = output
	bf1 = step[:]
	bf1[0] = bf0[0]
	bf1[1] = bf0[1]
	bf1[2] = bf0[2]
	bf1[3] = bf0[3]
	bf1[4] = bf0[4]
	bf1[5] = bf0[5]
	bf1[6] = bf0[6]
	bf1[7] = bf0[7]
	bf1[8] = halfBtf(cospi[8], bf0[8], cospi[56], bf0[9], cosBit)
	bf1[9] = halfBtf(cospi[56], bf0[8], -cospi[8], bf0[9], cosBit)
	bf1[10] = halfBtf(cospi[40], bf0[10], cospi[24], bf0[11], cosBit)
	bf1[11] = halfBtf(cospi[24], bf0[10], -cospi[40], bf0[11], cosBit)
	bf1[12] = halfBtf(-cospi[56], bf0[12], cospi[8], bf0[13], cosBit)
	bf1[13] = halfBtf(cospi[8], bf0[12], cospi[56], bf0[13], cosBit)
	bf1[14] = halfBtf(-cospi[24], bf0[14], cospi[40], bf0[15], cosBit)
	bf1[15] = halfBtf(cospi[40], bf0[14], cospi[24], bf0[15], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[0] + bf0[8]
	bf1[1] = bf0[1] + bf0[9]
	bf1[2] = bf0[2] + bf0[10]
	bf1[3] = bf0[3] + bf0[11]
	bf1[4] = bf0[4] + bf0[12]
	bf1[5] = bf0[5] + bf0[13]
	bf1[6] = bf0[6] + bf0[14]
	bf1[7] = bf0[7] + bf0[15]
	bf1[8] = bf0[0] - bf0[8]
	bf1[9] = bf0[1] - bf0[9]
	bf1[10] = bf0[2] - bf0[10]
	bf1[11] = bf0[3] - bf0[11]
	bf1[12] = bf0[4] - bf0[12]
	bf1[13] = bf0[5] - bf0[13]
	bf1[14] = bf0[6] - bf0[14]
	bf1[15] = bf0[7] - bf0[15]
	bf0 = output
	bf1 = step[:]
	bf1[0] = halfBtf(cospi[2], bf0[0], cospi[62], bf0[1], cosBit)
	bf1[1] = halfBtf(cospi[62], bf0[0], -cospi[2], bf0[1], cosBit)
	bf1[2] = halfBtf(cospi[10], bf0[2], cospi[54], bf0[3], cosBit)
	bf1[3] = halfBtf(cospi[54], bf0[2], -cospi[10], bf0[3], cosBit)
	bf1[4] = halfBtf(cospi[18], bf0[4], cospi[46], bf0[5], cosBit)
	bf1[5] = halfBtf(cospi[46], bf0[4], -cospi[18], bf0[5], cosBit)
	bf1[6] = halfBtf(cospi[26], bf0[6], cospi[38], bf0[7], cosBit)
	bf1[7] = halfBtf(cospi[38], bf0[6], -cospi[26], bf0[7], cosBit)
	bf1[8] = halfBtf(cospi[34], bf0[8], cospi[30], bf0[9], cosBit)
	bf1[9] = halfBtf(cospi[30], bf0[8], -cospi[34], bf0[9], cosBit)
	bf1[10] = halfBtf(cospi[42], bf0[10], cospi[22], bf0[11], cosBit)
	bf1[11] = halfBtf(cospi[22], bf0[10], -cospi[42], bf0[11], cosBit)
	bf1[12] = halfBtf(cospi[50], bf0[12], cospi[14], bf0[13], cosBit)
	bf1[13] = halfBtf(cospi[14], bf0[12], -cospi[50], bf0[13], cosBit)
	bf1[14] = halfBtf(cospi[58], bf0[14], cospi[6], bf0[15], cosBit)
	bf1[15] = halfBtf(cospi[6], bf0[14], -cospi[58], bf0[15], cosBit)
	bf0 = step[:]
	bf1 = output
	bf1[0] = bf0[1]
	bf1[1] = bf0[14]
	bf1[2] = bf0[3]
	bf1[3] = bf0[12]
	bf1[4] = bf0[5]
	bf1[5] = bf0[10]
	bf1[6] = bf0[7]
	bf1[7] = bf0[8]
	bf1[8] = bf0[9]
	bf1[9] = bf0[6]
	bf1[10] = bf0[11]
	bf1[11] = bf0[4]
	bf1[12] = bf0[13]
	bf1[13] = bf0[2]
	bf1[14] = bf0[15]
	bf1[15] = bf0[0]
}
