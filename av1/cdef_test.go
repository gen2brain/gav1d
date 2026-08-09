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
	cdefFilterBlock(buf, base, cdefStride, left, 0,
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
	d := newDsp[uint8]()
	shapes := [3][2]int{{8, 8}, {4, 4}, {4, 8}}
	cases := 0

	for iter := range 4 {
		for _, flat := range []bool{false, true} {
			for _, s := range shapes {
				w, h := s[0], s[1]
				for edges := range 16 {
					for _, pri := range []int{0, 1, 5, 15} {
						for _, sec := range []int{0, 1, 2, 4} {
							for dir := range 8 {
								for _, damping := range []int{3, 6} {
									ref, got := cdefAsmCase(d, iter, w, h, edges,
										pri, sec, dir, damping, flat)
									for i := range ref {
										if got[i] == ref[i] {
											continue
										}
										t.Fatalf("%dx%d flat=%v edges=%d pri=%d sec=%d dir=%d damping=%d: "+
											"buf[%d] = %d, want %d",
											w, h, flat, edges, pri, sec, dir, damping,
											i, got[i], ref[i])
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

	t.Logf("%d cases", cases)
}

// cdefAsmCase builds one block. The flat variant keeps the neighbourhood tight,
// which is what makes the clip to its range actually bind.
func cdefAsmCase(d *dspContext[uint8], iter, w, h, edges, pri, sec, dir, damping int, flat bool,
) (ref, got []uint8) {
	const bitdepthMax = 255

	gen := func(kind, i int) uint8 {
		v := cdefGen(iter, kind, i)
		if flat {
			return uint8(128 + v&7)
		}

		return uint8(v & bitdepthMax)
	}

	got = make([]uint8, cdefRows*cdefStride)
	for i := range got {
		got[i] = gen(0, i)
	}
	left := make([]uint8, 16)
	for i := range left {
		left[i] = gen(1, i)
	}
	ref = append([]uint8(nil), got...)

	base := cdefPad*cdefStride + cdefPad
	cdefFilterBlock(ref, base, cdefStride, left, 0,
		ref, base-2*cdefStride, ref, base+h*cdefStride,
		pri, sec, dir, damping, w, h, edges, bitdepthMax)
	d.cdefFilter(got, base, cdefStride, left, 0,
		got, base-2*cdefStride, got, base+h*cdefStride,
		pri, sec, dir, damping, w, h, edges, bitdepthMax)

	return ref, got
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
