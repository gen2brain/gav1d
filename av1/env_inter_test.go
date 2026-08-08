package av1

import (
	"strconv"
	"strings"
	"testing"
)

func TestInterCtx(t *testing.T) {
	sc, closeFixture := openFixture(t, "interctx.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "interctx" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		p := 1
		xb4, yb4 := num(fields[p]), num(fields[p+1])
		haveTop, haveLeft := num(fields[p+2]) != 0, num(fields[p+3]) != 0
		comp, dir, ref := num(fields[p+4]), num(fields[p+5]), num(fields[p+6])
		ohb, poc := num(fields[p+7]), num(fields[p+8])
		r0poc, r1poc := num(fields[p+9]), num(fields[p+10])
		p += 11

		var a, l blockContext
		for i := range 32 {
			a.intra[i] = uint8(num(fields[p]))
			a.compType[i] = uint8(num(fields[p+1]))
			a.ref[0][i] = int8(num(fields[p+2]))
			a.ref[1][i] = int8(num(fields[p+3]))
			a.filter[0][i] = uint8(num(fields[p+4]))
			a.filter[1][i] = uint8(num(fields[p+5]))
			l.intra[i] = uint8(num(fields[p+6]))
			l.compType[i] = uint8(num(fields[p+7]))
			l.ref[0][i] = int8(num(fields[p+8]))
			l.ref[1][i] = int8(num(fields[p+9]))
			l.filter[0][i] = uint8(num(fields[p+10]))
			l.filter[1][i] = uint8(num(fields[p+11]))
			p += 12
		}

		drlIdx := num(fields[p])
		p++
		stack := make([]refmvsCandidate, 8)
		for i := range 8 {
			stack[i].weight = num(fields[p])
			p++
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		got := []int{
			getFilterCtx(&a, &l, comp, dir, ref, yb4, xb4),
			getCompCtx(&a, &l, yb4, xb4, haveTop, haveLeft),
			getCompDirCtx(&a, &l, yb4, xb4, haveTop, haveLeft),
			getJntCompCtx(ohb, poc, r0poc, r1poc, &a, &l, yb4, xb4),
			getMaskCompCtx(&a, &l, yb4, xb4),
			av1GetRefCtx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetFwdRefCtx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetFwdRef1Ctx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetFwdRef2Ctx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetBwdRefCtx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetBwdRef1Ctx(&a, &l, yb4, xb4, haveTop, haveLeft),
			av1GetUniP1Ctx(&a, &l, yb4, xb4, haveTop, haveLeft),
			getDrlContext(stack, drlIdx),
		}

		names := []string{
			"filter", "comp", "comp_dir", "jnt_comp", "mask_comp", "ref",
			"fwd_ref", "fwd_ref_1", "fwd_ref_2", "bwd_ref", "bwd_ref_1",
			"uni_p1", "drl",
		}

		for i, v := range got {
			if v != num(fields[p+i]) {
				t.Fatalf("line %d: %s = %d, want %s", line, names[i], v, fields[p+i])
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
