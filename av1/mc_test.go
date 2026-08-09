package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	mcStride = 256
	mcRows   = 192
	mcPad    = 16
)

func mcGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

type mcBufs[P pixel] struct {
	src, dst   []P
	t1, t2     []int16
	msk, wmask []uint8
	blendTmp   []P
}

func mcFill[P pixel](b, iter int, bitdepthMax int32) *mcBufs[P] {
	m := &mcBufs[P]{
		src:      make([]P, mcRows*mcStride),
		dst:      make([]P, mcRows*mcStride),
		t1:       make([]int16, 128*128),
		t2:       make([]int16, 128*128),
		msk:      make([]uint8, 128*128),
		wmask:    make([]uint8, 128*128),
		blendTmp: make([]P, 128*128),
	}
	for i := range m.src {
		m.src[i] = P(mcGen(iter, b*8+0, i) & uint16(bitdepthMax))
		m.dst[i] = P(mcGen(iter, b*8+1, i) & uint16(bitdepthMax))
	}
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)
	for i := range m.t1 {
		a := int32(mcGen(iter, b*8+2, i) & uint16(bitdepthMax))
		c := int32(mcGen(iter, b*8+3, i) & uint16(bitdepthMax))
		m.t1[i] = int16(a<<ib - bias)
		m.t2[i] = int16(c<<ib - bias)
		m.msk[i] = uint8(mcGen(iter, b*8+4, i) & 63)
		m.blendTmp[i] = P(mcGen(iter, b*8+5, i) & uint16(bitdepthMax))
	}

	return m
}

func (m *mcBufs[P]) dump(w, h int) []int {
	base := mcPad*mcStride + mcPad
	out := make([]int, 0, w*h)
	for y := range h {
		for x := range w {
			out = append(out, int(m.dst[base+y*mcStride+x]))
		}
	}

	return out
}

func runMC[P pixel](b, iter, filter2d, isPrep, w, h, mx, my int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	src := m.src
	base := mcPad*mcStride + mcPad

	out := make([]int, 0, w*h)
	if isPrep != 0 {
		tmp := make([]int16, 128*128)
		mct(tmp, src, base, mcStride, w, h, mx, my, filter2d, bitdepthMax)
		for i := range w * h {
			out = append(out, int(tmp[i]))
		}

		return out
	}

	mc(newDspGo[P](), m.dst, base, mcStride, src, base, mcStride, w, h, mx, my,
		filter2d, bitdepthMax)

	return m.dump(w, h)
}

func runMCCompound[P pixel](b, iter, kind, w, h, weight int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	switch kind {
	case 0:
		mcAvg(m.dst, base, mcStride, m.t1, m.t2, w, h, bitdepthMax)
	case 1:
		mcWAvg(m.dst, base, mcStride, m.t1, m.t2, w, h, weight, bitdepthMax)
	default:
		mcMask(m.dst, base, mcStride, m.t1, m.t2, w, h, m.msk, bitdepthMax)
	}

	return m.dump(w, h)
}

func runMCWMask[P pixel](b, iter, ss, sign, w, h int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad
	ssHor, ssVer := b2i(ss > 0), b2i(ss > 1)

	mcWMask(m.dst, base, mcStride, m.t1, m.t2, w, h, m.wmask, sign, ssHor, ssVer,
		bitdepthMax)

	out := m.dump(w, h)
	for i := range (w >> ssHor) * (h >> ssVer) {
		out = append(out, int(m.wmask[i]))
	}

	return out
}

func runMCBlend[P pixel](b, iter, kind, w, h int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	switch kind {
	case 0:
		mcBlend(m.dst, base, mcStride, m.blendTmp, w, h, m.msk)
	case 1:
		mcBlendV(m.dst, base, mcStride, m.blendTmp, w, h)
	default:
		mcBlendH(m.dst, base, mcStride, m.blendTmp, w, h)
	}

	return m.dump(w, h)
}

func runMCWarp[P pixel](b, iter int, abcd *[4]int16, mx, my int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	warpAffine8x8(m.dst, base, mcStride, m.src, base, mcStride, abcd, mx, my, bitdepthMax)

	return m.dump(8, 8)
}

func runMCScaled[P pixel](b, iter, filter2d, isPrep, w, h, mx, my, dx, dy int,
	bitdepthMax int32,
) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	if isPrep != 0 {
		tmp := make([]int16, 128*128)
		mctScaled(tmp, m.src, base, mcStride, w, h, mx, my, dx, dy, filter2d, bitdepthMax)
		out := make([]int, 0, w*h)
		for i := range w * h {
			out = append(out, int(tmp[i]))
		}

		return out
	}

	mcScaled(m.dst, base, mcStride, m.src, base, mcStride, w, h, mx, my, dx, dy,
		filter2d, bitdepthMax)

	return m.dump(w, h)
}

func runMCWarpT[P pixel](b, iter int, abcd *[4]int16, mx, my int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	tmp := make([]int16, 64)
	warpAffine8x8t(tmp, 8, m.src, base, mcStride, abcd, mx, my, bitdepthMax)

	out := make([]int, 0, 64)
	for i := range 64 {
		out = append(out, int(tmp[i]))
	}

	return out
}

func runMCEmuEdge[P pixel](b, iter, w, h, iw, ih, x, y int, bitdepthMax int32) []int {
	m := mcFill[P](b, iter, bitdepthMax)
	base := mcPad*mcStride + mcPad

	emuEdge(w, h, iw, ih, x, y, m.dst, base, mcStride, m.src, base, mcStride)

	return m.dump(w, h)
}

func TestMC(t *testing.T) {
	sc, closeFixture := openFixture(t, "mc.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "mc" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		kind := fields[0]
		bpc := num(fields[1])
		iter := num(fields[2])
		bitdepthMax := int32(1<<bpc - 1)
		b := (bpc - 8) / 2
		hi := bpc != 8

		var nArgs int
		switch kind {
		case "M":
			nArgs = 6
		case "A":
			nArgs = 4
		case "W":
			nArgs = 4
		case "B":
			nArgs = 3
		case "P":
			nArgs = 6
		case "E":
			nArgs = 6
		case "S":
			nArgs = 8
		case "T2":
			nArgs = 6
		default:
			t.Fatalf("line %d: unknown record %q", line, kind)
		}
		if fields[3+nArgs] != "=" {
			t.Fatalf("line %d: malformed %s record", line, kind)
		}
		a := fields[3 : 3+nArgs]
		want := fields[4+nArgs:]

		var got []int
		var w int

		switch kind {
		case "M":
			filter2d, isPrep := num(a[0]), num(a[1])
			h, mx, my := num(a[3]), num(a[4]), num(a[5])
			w = num(a[2])
			if hi {
				got = runMC[uint16](b, iter, filter2d, isPrep, w, h, mx, my, bitdepthMax)
			} else {
				got = runMC[uint8](b, iter, filter2d, isPrep, w, h, mx, my, bitdepthMax)
			}
		case "A":
			ck, h, weight := num(a[0]), num(a[2]), num(a[3])
			w = num(a[1])
			if hi {
				got = runMCCompound[uint16](b, iter, ck, w, h, weight, bitdepthMax)
			} else {
				got = runMCCompound[uint8](b, iter, ck, w, h, weight, bitdepthMax)
			}
		case "W":
			ss, sign, h := num(a[0]), num(a[1]), num(a[3])
			w = num(a[2])
			if hi {
				got = runMCWMask[uint16](b, iter, ss, sign, w, h, bitdepthMax)
			} else {
				got = runMCWMask[uint8](b, iter, ss, sign, w, h, bitdepthMax)
			}
		case "B":
			ck, h := num(a[0]), num(a[2])
			w = num(a[1])
			if hi {
				got = runMCBlend[uint16](b, iter, ck, w, h, bitdepthMax)
			} else {
				got = runMCBlend[uint8](b, iter, ck, w, h, bitdepthMax)
			}
		case "P":
			abcd := [4]int16{
				int16(num(a[0])), int16(num(a[1])), int16(num(a[2])), int16(num(a[3])),
			}
			mx, my := num(a[4]), num(a[5])
			w = 8
			if hi {
				got = runMCWarp[uint16](b, iter, &abcd, mx, my, bitdepthMax)
			} else {
				got = runMCWarp[uint8](b, iter, &abcd, mx, my, bitdepthMax)
			}
		case "S":
			filter2d, isPrep := num(a[0]), num(a[1])
			h := num(a[3])
			mx, my := num(a[4]), num(a[5])
			dx, dy := num(a[6]), num(a[7])
			w = num(a[2])
			if hi {
				got = runMCScaled[uint16](b, iter, filter2d, isPrep, w, h, mx, my, dx, dy, bitdepthMax)
			} else {
				got = runMCScaled[uint8](b, iter, filter2d, isPrep, w, h, mx, my, dx, dy, bitdepthMax)
			}
		case "T2":
			abcd := [4]int16{
				int16(num(a[0])), int16(num(a[1])), int16(num(a[2])), int16(num(a[3])),
			}
			mx, my := num(a[4]), num(a[5])
			w = 8
			if hi {
				got = runMCWarpT[uint16](b, iter, &abcd, mx, my, bitdepthMax)
			} else {
				got = runMCWarpT[uint8](b, iter, &abcd, mx, my, bitdepthMax)
			}
		case "E":
			h, iw, ih := num(a[1]), num(a[2]), num(a[3])
			x, y := num(a[4]), num(a[5])
			w = num(a[0])
			if hi {
				got = runMCEmuEdge[uint16](b, iter, w, h, iw, ih, x, y, bitdepthMax)
			} else {
				got = runMCEmuEdge[uint8](b, iter, w, h, iw, ih, x, y, bitdepthMax)
			}
		}

		if len(got) != len(want) {
			t.Fatalf("line %d: %s: %d outputs, want %d", line, kind, len(got), len(want))
		}
		for i, v := range got {
			if v != num(want[i]) {
				t.Fatalf("line %d: %s bpc=%d args=%v out[%d][%d] = %d, want %s",
					line, kind, bpc, a, i/w, i%w, v, want[i])
			}
		}

		cases++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d cases", cases)
}

// TestPut8tapAsm checks the architecture-specific put8tap against the pure Go
// one over every block shape, filter type and subpel position. The whole
// destination buffer is compared, so a kernel that writes outside the block
// fails.
func TestPut8tapAsm(t *testing.T) {
	d, ref := newDsp[uint8](), newDspGo[uint8]()
	base := mcPad*mcStride + mcPad
	widths := []int{2, 4, 8, 16, 32, 64, 128}
	heights := []int{2, 4, 8, 16, 64}
	subpel := []int{0, 1, 7, 15}
	cases := 0

	got := make([]uint8, mcRows*mcStride)
	want := make([]uint8, mcRows*mcStride)

	for iter := range 2 {
		m := mcFill[uint8](0, iter, 255)
		for _, filter2d := range filter2dFilterType {
			for _, mx := range subpel {
				for _, my := range subpel {
					for _, w := range widths {
						for _, h := range heights {
							copy(got, m.dst)
							copy(want, m.dst)
							d.put8tap(got, base, mcStride, m.src, base, mcStride,
								w, h, mx, my, filter2d, 255)
							ref.put8tap(want, base, mcStride, m.src, base, mcStride,
								w, h, mx, my, filter2d, 255)

							for i := range want {
								if got[i] == want[i] {
									continue
								}
								t.Fatalf("%dx%d mx=%d my=%d filter=%d: [%d,%d] = %d, want %d",
									w, h, mx, my, filter2d,
									(i-base)/mcStride, (i-base)%mcStride, got[i], want[i])
							}
							cases++
						}
					}
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}
