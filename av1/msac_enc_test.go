package av1

import (
	"testing"
)

type op struct {
	kind string
	val  uint32
	n    int
	cdf  []uint16
}

func defaultSymCdf(n int) []uint16 {
	cdf := make([]uint16, n+1)
	for i := range n - 1 {
		cdf[i] = uint16(32768 - (i+1)*32768/n)
	}

	return cdf
}

func TestMsacRoundtrip(t *testing.T) {
	for _, update := range []bool{false, true} {
		for iter := range 8 {
			var ops []op
			g := uint32(iter*7919 + 13)
			next := func(m uint32) uint32 {
				g = g*1664525 + 1013904223

				return g >> 16 % m
			}

			for range 400 {
				switch next(5) {
				case 0:
					ops = append(ops, op{kind: "equi", val: next(2)})
				case 1:
					ops = append(ops, op{kind: "boolf", val: next(2), n: int(next(30000) + 1024)})
				case 2:
					ops = append(ops, op{kind: "booladapt", val: next(2), cdf: []uint16{16384, 0}})
				case 3:
					n := int(next(14) + 2)
					ops = append(ops, op{kind: "sym", val: next(uint32(n)), n: n, cdf: defaultSymCdf(n)})
				case 4:
					n := int(next(16) + 1)
					ops = append(ops, op{kind: "lit", val: next(1 << uint(min(n, 16))), n: n})
				}
			}

			var e msacEncoder
			e.init(!update)
			for i := range ops {
				o := &ops[i]
				switch o.kind {
				case "equi":
					e.boolEqui(o.val)
				case "boolf":
					e.boolF(o.val, uint32(o.n))
				case "booladapt":
					e.boolAdapt(o.cdf, o.val)
				case "sym":
					e.symbolAdapt(o.cdf, o.n, o.val)
				case "lit":
					e.literal(o.val, o.n)
				}
			}
			data := e.done()

			var d msacContext
			d.init(data, !update)
			for i := range ops {
				o := &ops[i]
				var got uint32
				switch o.kind {
				case "equi":
					got = d.boolEquiGo()
				case "boolf":
					got = d.boolFGo(uint32(o.n))
				case "booladapt":
					cdf := []uint16{16384, 0}
					got = d.boolAdaptGo(cdf)
				case "sym":
					cdf := defaultSymCdf(o.n)
					got = d.symbolAdaptGo(cdf, o.n)
				case "lit":
					got = d.bools(o.n)
				}
				if got != o.val {
					t.Fatalf("update=%v iter=%d op %d (%s n=%d): got %d, want %d",
						update, iter, i, o.kind, o.n, got, o.val)
				}
			}
		}
	}
}
