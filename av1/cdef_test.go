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
