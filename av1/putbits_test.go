package av1

import "testing"

func TestPutBitsRoundtrip(t *testing.T) {
	type item struct {
		kind string
		v    uint32
		n    int
	}

	g := uint32(4242)
	next := func(m uint32) uint32 {
		g = g*1664525 + 1013904223

		return g >> 16 % m
	}

	var items []item
	for range 600 {
		switch next(5) {
		case 0:
			n := int(next(24) + 1)
			items = append(items, item{"bits", next(1 << uint(min(n, 16))), n})
		case 1:
			items = append(items, item{"bit", next(2), 1})
		case 2:
			items = append(items, item{"uleb", next(60000), 0})
		case 3:
			n := int(next(30) + 2)
			items = append(items, item{"uniform", next(uint32(n)), n})
		case 4:
			items = append(items, item{"vlc", next(5000), 0})
		}
	}

	var p putBits
	for _, it := range items {
		switch it.kind {
		case "bits":
			p.bits32(it.v, it.n)
		case "bit":
			p.bit(it.v)
		case "uleb":
			p.uleb128(it.v)
		case "uniform":
			p.uniform(it.v, uint32(it.n))
		case "vlc":
			p.vlc(it.v)
		}
	}

	var c getBits
	c.init(p.bytes())
	for i, it := range items {
		var got uint32
		switch it.kind {
		case "bits":
			got = c.bits(it.n)
		case "bit":
			got = c.bit()
		case "uleb":
			got = c.uleb128()
		case "uniform":
			got = c.uniform(uint32(it.n))
		case "vlc":
			got = c.vlc()
		}
		if got != it.v {
			t.Fatalf("item %d (%s n=%d): got %d, want %d", i, it.kind, it.n, got, it.v)
		}
	}
}
