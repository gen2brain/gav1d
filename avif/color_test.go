package avif

import (
	"testing"
)

func rowGen(iter, kind, i int) uint32 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15

	return v * 2654435761
}

// TestConvertRowAsm checks the architecture-specific row against the pure Go
// one over every width and each matrix, comparing the whole destination so a
// kernel that writes past the row fails.
func TestConvertRowAsm(t *testing.T) {
	asm := rowFn()
	matrices := []int{mcBT601, mcBT709, mcBT2020NCL}
	cases := 0

	for iter := range 4 {
		for _, matrix := range matrices {
			for _, full := range []bool{false, true} {
				s := &colorState{matrix: matrix, fullRange: full, depth: 8, ssHor: 1}
				s.maxChannel = 255
				s.outMax = 255
				s.kr, s.kg, s.kb = yuvCoefficients(matrix)
				s.crCoef, s.cbCoef = 2*(1-s.kr), 2*(1-s.kb)
				s.gcr, s.gcb = s.kr*(1-s.kr), s.kb*(1-s.kb)
				c := s.rowConsts()

				for n := 1; n <= 40; n++ {
					uw := (n + 1) >> 1
					yf := make([]float32, n)
					a := make([]uint8, n)
					for i := range yf {
						yf[i] = float32(rowGen(iter, 0, i)%1400)/1000 - 0.2
						a[i] = uint8(rowGen(iter, 1, i))
					}
					rows := make([][]float32, 4)
					for r := range rows {
						rows[r] = make([]float32, uw+8)
						for i := range rows[r] {
							rows[r][i] = float32(rowGen(iter, 2+r, i)%1200)/1000 - 0.6
						}
					}

					got := make([]uint8, 4*n+32)
					want := make([]uint8, 4*n+32)
					for i := range got {
						got[i] = uint8(rowGen(iter, 6, i))
						want[i] = got[i]
					}

					asm(got, yf, rows[0], rows[1], rows[2], rows[3], a, n, &c)
					convertRowGo(want, yf, rows[0], rows[1], rows[2], rows[3], a, n, &c)

					for i := range want {
						if got[i] == want[i] {
							continue
						}
						t.Fatalf("matrix=%d full=%v n=%d: buf[%d] = %d, want %d",
							matrix, full, n, i, got[i], want[i])
					}
					cases++
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}
