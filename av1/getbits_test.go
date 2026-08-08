package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestGetBits(t *testing.T) {
	sc, closeFixture := openFixture(t, "getbits.txt")
	defer closeFixture()

	var c getBits
	var ops int
	var err error

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "getbits" {
			continue
		}

		if fields[0] == "D" {
			data, err := hex.DecodeString(fields[1])
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			c.init(data)

			continue
		}

		eq := len(fields) - 5
		if fields[eq] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}

		args := make([]int64, eq-1)
		for i := range args {
			if args[i], err = strconv.ParseInt(fields[1+i], 10, 64); err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
		}

		want := make([]int64, 4)
		for i := range want {
			if want[i], err = strconv.ParseInt(fields[eq+1+i], 10, 64); err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
		}

		op := fields[0]

		var got int64
		switch op {
		case "B":
			got = int64(c.bit())
		case "G":
			got = int64(c.bits(int(args[0])))
		case "S":
			got = int64(c.sbits(int(args[0])))
		case "U":
			got = int64(c.uleb128())
		case "N":
			got = int64(c.uniform(uint32(args[0])))
		case "V":
			got = int64(c.vlc())
		case "X":
			got = int64(c.subexp(int32(args[0]), uint32(args[1])))
		case "A":
			c.bytealign()
		case "P":
			got = int64(c.pos())
		default:
			t.Fatalf("line %d: unknown op %q", line, op)
		}

		gotErr := int64(0)
		if c.err {
			gotErr = 1
		}

		if got != want[0] || int64(c.index) != want[1] || int64(c.bitsLeft) != want[2] || gotErr != want[3] {
			t.Fatalf("line %d: op %s%v:\n got %d index=%d bitsLeft=%d err=%d\nwant %d index=%d bitsLeft=%d err=%d",
				line, op, args, got, c.index, c.bitsLeft, gotErr, want[0], want[1], want[2], want[3])
		}

		ops++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if ops == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d ops", ops)
}
