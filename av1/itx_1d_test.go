package av1

import (
	"strconv"
	"strings"
	"testing"
)

func TestItx1d(t *testing.T) {
	sc, closeFixture := openFixture(t, "itx1d.txt")
	defer closeFixture()

	num := func(s string) int {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatal(err)
		}

		return v
	}

	sizes := [5]int{4, 8, 16, 32, 64}
	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "itx1d" {
			continue
		}

		eq := 0
		for eq < len(fields) && fields[eq] != "=" {
			eq++
		}

		var n, stride int
		var name string
		var run func(c []int32)

		switch fields[0] {
		case "T":
			sz, typ := num(fields[1]), num(fields[2])
			stride = num(fields[3])
			mn, mx := int32(num(fields[4])), int32(num(fields[5]))
			n = sizes[sz]
			fn := tx1dFns[sz][typ]
			if fn == nil {
				t.Fatalf("line %d: no Go fn for size %d type %d", line, sz, typ)
			}
			name = "size=" + fields[1] + " type=" + fields[2]
			run = func(c []int32) { fn(c, 0, stride, mn, mx) }

		case "W":
			stride = num(fields[1])
			n = 4
			name = "wht4"
			run = func(c []int32) { invWht41d(c, 0, stride) }

		default:
			t.Fatalf("line %d: unknown op %q", line, fields[0])
		}

		in := fields[eq-n : eq]
		want := fields[eq+1:]
		if len(want) != n {
			t.Fatalf("line %d: got %d outputs, want %d", line, len(want), n)
		}

		c := make([]int32, 64*4)
		for i := range n {
			c[i*stride] = int32(num(in[i]))
		}

		run(c)

		for i := range n {
			if int(c[i*stride]) != num(want[i]) {
				t.Fatalf("line %d: %s stride=%d out[%d] = %d, want %s",
					line, name, stride, i, c[i*stride], want[i])
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
