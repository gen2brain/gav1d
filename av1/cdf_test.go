package av1

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unsafe"
)

func TestCdf(t *testing.T) {
	sc, closeFixture := openFixture(t, "cdf.txt")
	defer closeFixture()

	paths := fieldPaths(reflect.TypeOf(cdfContext{}), "", nil)

	check := func(line int, tag string, c *cdfContext, want []string) {
		got := flatten(reflect.ValueOf(*c), nil)
		if len(got) != len(want) {
			t.Fatalf("line %d: %s has %d values, dav1d has %d", line, tag, len(got), len(want))
		}
		for i := range got {
			w, err := strconv.ParseInt(want[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			if got[i] != w {
				t.Fatalf("line %d: %s %s (flat %d) = %d, want %d", line, tag, paths[i], i, got[i], w)
			}
		}
	}

	var prev *cdfContext
	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "cdf" {
			continue
		}

		switch fields[0] {
		case "I":
			qidx, err := strconv.Atoi(fields[1])
			if err != nil {
				t.Fatal(err)
			}
			var c cdfContext
			c.initStatic(uint8(qidx))
			check(line, "init qidx="+fields[1], &c, fields[2:])
			cases++

		case "R":
			c := new(cdfContext)
			vals := fields[1:]
			raw := unsafe.Slice((*uint16)(unsafe.Pointer(c)), unsafe.Sizeof(*c)/2)
			if len(vals) != len(raw) || len(raw) != len(flatten(reflect.ValueOf(*c), nil)) {
				t.Fatalf("line %d: fixture has %d values, struct has %d", line, len(vals), len(raw))
			}
			for i := range raw {
				n, err := strconv.ParseUint(vals[i], 10, 16)
				if err != nil {
					t.Fatal(err)
				}
				raw[i] = uint16(n)
			}
			prev = c

		case "Z":
			if prev == nil {
				t.Fatalf("line %d: Z without R", line)
			}
			ft, err := strconv.Atoi(fields[1])
			if err != nil {
				t.Fatal(err)
			}
			var c cdfContext
			c.initStatic(0)
			cdfThreadUpdate(&frameHeader{frameType: ft}, &c, prev)
			check(line, "cdfThreadUpdate ft="+fields[1], &c, fields[2:])
			cases++
		}
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d cases", cases)
}

func fieldPaths(t reflect.Type, prefix string, out []string) []string {
	switch t.Kind() {
	case reflect.Array:
		for i := range t.Len() {
			out = fieldPaths(t.Elem(), fmt.Sprintf("%s[%d]", prefix, i), out)
		}
	case reflect.Struct:
		for i := range t.NumField() {
			f := t.Field(i)
			name := f.Name
			if prefix != "" {
				name = prefix + "." + name
			}
			out = fieldPaths(f.Type, name, out)
		}
	default:
		out = append(out, prefix)
	}

	return out
}
