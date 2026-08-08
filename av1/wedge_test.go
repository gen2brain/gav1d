package av1

import (
	"strconv"
	"strings"
	"testing"
)

func TestWedgeMasks(t *testing.T) {
	sc, closeFixture := openFixture(t, "wedge.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "wedge" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		var got []uint8
		var want []string
		var w, h int

		switch fields[0] {
		case "W":
			c, bs := num(fields[1]), num(fields[2])
			sign, n := num(fields[3]), num(fields[4])
			w, h = num(fields[5]), num(fields[6])
			got = wedgeMasks[c][bs][sign][n]
			want = fields[7:]

		case "I":
			c, bs, p := num(fields[1]), num(fields[2]), num(fields[3])
			w, h = num(fields[4]), num(fields[5])
			got = iiMasks[c][bs][p]
			want = fields[6:]

		default:
			t.Fatalf("line %d: unknown record %q", line, fields[0])
		}

		// The DC interintra mask is one constant-filled 32x32 buffer shared by
		// every size, so a shorter block reads a prefix of it.
		if len(got) < w*h {
			t.Fatalf("line %d: %s: mask is %d bytes, want at least %d",
				line, fields[0], len(got), w*h)
		}
		got = got[:w*h]
		if len(want) != w*h {
			t.Fatalf("line %d: %s: fixture has %d bytes, want %d",
				line, fields[0], len(want), w*h)
		}
		for i, v := range got {
			if int(v) != num(want[i]) {
				t.Fatalf("line %d: %s %s: mask[%d][%d] = %d, want %s",
					line, fields[0], strings.Join(fields[1:6], ","),
					i/w, i%w, v, want[i])
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
