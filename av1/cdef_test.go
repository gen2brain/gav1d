package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	cdefStride = 32
	cdefRows   = 32
	cdefPad    = 8
)

func cdefGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func runCdefDir[P pixel](t *testing.T, b, iter int, bitdepthMax int32) (int, uint32) {
	t.Helper()

	img := make([]P, 8*cdefStride)
	for i := range img {
		img[i] = P(cdefGen(iter, b*4+2, i) & uint16(bitdepthMax))
	}

	return cdefFindDir(img, 0, cdefStride, bitdepthMax)
}

func runCdefFilter[P pixel](t *testing.T, b, iter, w, h, edges, pri, sec, dir, damping int,
	bitdepthMax int32,
) []int {
	t.Helper()

	buf := make([]P, cdefRows*cdefStride)
	for i := range buf {
		buf[i] = P(cdefGen(iter, b*4+0, i) & uint16(bitdepthMax))
	}
	left := make([]P, 16)
	for i := range left {
		left[i] = P(cdefGen(iter, b*4+1, i) & uint16(bitdepthMax))
	}

	base := cdefPad*cdefStride + cdefPad
	cdefFilterBlock(newDspGo[P](), dirtyCdefTmp(3), buf, base, cdefStride, left, 0,
		buf, base-2*cdefStride, buf, base+h*cdefStride,
		pri, sec, dir, damping, w, h, edges, bitdepthMax)

	out := make([]int, 0, w*h)
	for y := range h {
		for x := range w {
			out = append(out, int(buf[base+y*cdefStride+x]))
		}
	}

	return out
}

// TestCdefAsm checks the architecture-specific CDEF filter against the pure Go
// one across every block shape, edge combination, direction and strength pair.
// The whole buffer is compared, so a kernel that writes outside the block fails.
func TestCdefAsm(t *testing.T) {
	cases := 0
	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		if bpc == 8 {
			cases += cdefAsmDepth(t, newDsp[uint8](), bitdepthMax)

			continue
		}
		cases += cdefAsmDepth(t, newDsp[uint16](), bitdepthMax)
	}

	t.Logf("%d cases", cases)
}

func cdefAsmDepth[P pixel](t *testing.T, d *dspContext[P], bitdepthMax int32) int {
	t.Helper()

	// The strengths are the encoded levels scaled by the bit depth, the way
	// cdefBrow derives them.
	min8 := bitdepthFromMax(bitdepthMax) - 8
	shapes := [3][2]int{{8, 8}, {4, 4}, {4, 8}}
	cases := 0

	for iter := range 4 {
		for _, flat := range []bool{false, true} {
			for _, s := range shapes {
				w, h := s[0], s[1]
				for edges := range 16 {
					for _, pri := range []int{0, 1 << min8, 5 << min8, 15 << min8} {
						for _, sec := range []int{0, 1 << min8, 2 << min8, 4 << min8} {
							for dir := range 8 {
								for _, damping := range []int{3 + min8, 6 + min8} {
									ref, got := cdefAsmCase(d, iter, w, h, edges,
										pri, sec, dir, damping, flat, bitdepthMax)
									for i := range ref {
										if got[i] == ref[i] {
											continue
										}
										t.Fatalf("bpc=%d %dx%d flat=%v edges=%d pri=%d sec=%d dir=%d damping=%d: "+
											"buf[%d] = %d, want %d",
											bitdepthFromMax(bitdepthMax), w, h, flat, edges,
											pri, sec, dir, damping, i, got[i], ref[i])
									}
									cases++
								}
							}
						}
					}
				}
			}
		}
	}

	return cases
}

// cdefAsmCase builds one block. The flat variant keeps the neighbourhood tight,
// which is what makes the clip to its range actually bind.
func cdefAsmCase[P pixel](d *dspContext[P], iter, w, h, edges, pri, sec, dir, damping int,
	flat bool, bitdepthMax int32,
) (ref, got []P) {
	gen := func(kind, i int) P {
		v := cdefGen(iter, kind, i)
		if flat {
			return P(int32(128<<(bitdepthFromMax(bitdepthMax)-8)) + int32(v&7))
		}

		return P(int32(v) & bitdepthMax)
	}

	got = make([]P, cdefRows*cdefStride)
	for i := range got {
		got[i] = gen(0, i)
	}
	left := make([]P, 16)
	for i := range left {
		left[i] = gen(1, i)
	}
	ref = append([]P(nil), got...)

	base := cdefPad*cdefStride + cdefPad
	cdefFilterBlock(newDspGo[P](), dirtyCdefTmp(3), ref, base, cdefStride, left, 0,
		ref, base-2*cdefStride, ref, base+h*cdefStride,
		pri, sec, dir, damping, w, h, edges, bitdepthMax)
	d.cdefFilter(dirtyCdefTmp(250), got, base, cdefStride, left, 0,
		got, base-2*cdefStride, got, base+h*cdefStride,
		pri, sec, dir, damping, w, h, edges, bitdepthMax)

	return ref, got
}

// dirtyCdefTmp returns a padding buffer full of one plausible sample value.
// The buffer is reused across blocks, so a block that reads any part of it the
// padding did not write has to be caught here rather than in a decode. The
// value has to be one a sample could take: the filter neutralises anything far
// outside the range the same way it neutralises the unavailable marker, so a
// wild value would hide the very thing this is here to find.
func dirtyCdefTmp(v int16) []int16 {
	tmp := make([]int16, 144)
	for i := range tmp {
		tmp[i] = v
	}

	return tmp
}

func TestCdef(t *testing.T) {
	sc, closeFixture := openFixture(t, "cdef.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "cdef" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		bpc := num(fields[1])
		iter := num(fields[2])
		bitdepthMax := int32(1<<bpc - 1)
		b := (bpc - 8) / 2

		switch fields[0] {
		case "D":
			var dir int
			var variance uint32
			if bpc == 8 {
				dir, variance = runCdefDir[uint8](t, b, iter, bitdepthMax)
			} else {
				dir, variance = runCdefDir[uint16](t, b, iter, bitdepthMax)
			}
			if dir != num(fields[4]) || variance != uint32(num(fields[5])) {
				t.Fatalf("line %d: find_dir = %d %d, want %s %s",
					line, dir, variance, fields[4], fields[5])
			}

		case "F":
			w, h := num(fields[3]), num(fields[4])
			edges := num(fields[5])
			pri, sec := num(fields[6]), num(fields[7])
			dir, damping := num(fields[8]), num(fields[9])
			if fields[10] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want := fields[11:]

			var got []int
			if bpc == 8 {
				got = runCdefFilter[uint8](t, b, iter, w, h, edges, pri, sec, dir, damping, bitdepthMax)
			} else {
				got = runCdefFilter[uint16](t, b, iter, w, h, edges, pri, sec, dir, damping, bitdepthMax)
			}
			if len(got) != len(want) {
				t.Fatalf("line %d: %d outputs, want %d", line, len(got), len(want))
			}
			for i, v := range got {
				if v != num(want[i]) {
					t.Fatalf("line %d: bpc=%d %dx%d edges=%d pri=%d sec=%d dir=%d damping=%d "+
						"out[%d][%d] = %d, want %s",
						line, bpc, w, h, edges, pri, sec, dir, damping,
						i/w, i%w, v, want[i])
				}
			}

		default:
			t.Fatalf("line %d: unknown record %q", line, fields[0])
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

// TestCdefDirAsm checks the architecture-specific direction search against the
// pure Go one over blocks that are flat, noisy and strongly directional, since
// the winning cost is what the result turns on.
func TestCdefDirAsm(t *testing.T) {
	cases := 0
	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		if bpc == 8 {
			cases += cdefDirAsmDepth[uint8](t, bitdepthMax)

			continue
		}
		cases += cdefDirAsmDepth[uint16](t, bitdepthMax)
	}

	t.Logf("%d cases", cases)
}

func cdefDirAsmDepth[P pixel](t *testing.T, bitdepthMax int32) int {
	t.Helper()

	ref, got := newDspGo[P](), newDsp[P]()
	cases := 0

	for iter := range 64 {
		for kind := range 4 {
			img := make([]P, 8*cdefStride)
			for i := range img {
				y, x := i/cdefStride, i%cdefStride
				var v uint16
				switch kind {
				case 0:
					v = cdefGen(iter, kind, i)
				case 1:
					v = uint16(128 + (x+y)*3 + int(cdefGen(iter, kind, i)&3))
				case 2:
					v = uint16(128 + (x-y)*5 + int(cdefGen(iter, kind, i)&1))
				default:
					v = uint16(128 + y*7)
				}
				img[i] = P(v & uint16(bitdepthMax))
			}

			wantDir, wantVar := ref.cdefDir(img, 0, cdefStride, bitdepthMax)
			gotDir, gotVar := got.cdefDir(img, 0, cdefStride, bitdepthMax)
			if gotDir != wantDir || gotVar != wantVar {
				t.Fatalf("bpc=%d iter=%d kind=%d: dir,var = %d,%d, want %d,%d",
					bitdepthFromMax(bitdepthMax), iter, kind, gotDir, gotVar, wantDir, wantVar)
			}
			cases++
		}
	}

	return cases
}

// TestCdefDirCostAsm checks the direction costs against the Go loops they
// replace. The sums are whatever the block sums can produce, including the
// negatives and the wide magnitudes that make the squares wrap.
func TestCdefDirCostAsm(t *testing.T) {
	_ = dsp8()
	if cdefDirCosts == nil {
		t.Skip("no direction cost assembly on this architecture")
	}

	for iter := range 64 {
		var s cdefSums
		fill := func(v []int32, seed int) {
			for i := range v {
				g := int32(cdefGen(iter*31+seed, i, iter))
				switch iter & 3 {
				case 0:
					v[i] = g & 0x7ff
				case 1:
					v[i] = -(g & 0x7ff)
				case 2:
					v[i] = g
				default:
					v[i] = g &^ 0xffff
				}
			}
		}
		for i := range s.hv {
			fill(s.hv[i][:], i)
		}
		for i := range s.diag {
			fill(s.diag[i][:], 4+i)
		}
		for i := range s.alt {
			fill(s.alt[i][:], 8+i)
		}

		var want, got [8]uint32
		cdefDirCostsGo(&s, &want)
		cdefDirCosts(&s, &got)
		if got != want {
			t.Fatalf("iter=%d: costs %v, want %v", iter, got, want)
		}

		wd, wc := cdefBestDir(&want)
		gd, gc := cdefBestDir(&got)
		if wd != gd || wc != gc {
			t.Fatalf("iter=%d: dir %d/%d cost %d/%d", iter, gd, wd, gc, wc)
		}
	}
}
