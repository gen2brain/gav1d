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
	matrices := []int{mcBT601, mcBT709, mcBT2020NCL}
	cases := 0

	for iter := range 4 {
		for _, ssHor := range []int{0, 1} {
			asm, ref := rowFn(ssHor), convertRowGo
			if ssHor == 0 {
				ref = convertRow444Go
			}
			for _, matrix := range matrices {
				full := iter&1 != 0
				s := &colorState{matrix: matrix, fullRange: full, depth: 8, ssHor: ssHor}
				s.maxChannel = 255
				s.outMax = 255
				s.kr, s.kg, s.kb = yuvCoefficients(matrix)
				s.crCoef, s.cbCoef = 2*(1-s.kr), 2*(1-s.kb)
				s.gcr, s.gcb = s.kr*(1-s.kr), s.kb*(1-s.kb)
				c := s.rowConsts()

				for n := 1; n <= 40; n++ {
					uw := (n + ssHor) >> ssHor
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
					ref(want, yf, rows[0], rows[1], rows[2], rows[3], a, n, &c)

					for i := range want {
						if got[i] == want[i] {
							continue
						}
						t.Fatalf("ssHor=%d matrix=%d full=%v n=%d: buf[%d] = %d, want %d",
							ssHor, matrix, full, n, i, got[i], want[i])
					}
					cases++
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}

func TestConvertRow16Asm(t *testing.T) {
	matrices := []int{mcBT601, mcBT709, mcBT2020NCL}
	cases := 0

	for iter := range 4 {
		for _, ssHor := range []int{0, 1} {
			asm, ref := rowFn16(ssHor), convertRow16Go
			if ssHor == 0 {
				ref = convertRow16x444Go
			}
			for _, matrix := range matrices {
				full := iter&1 != 0
				for _, depth := range []int{10, 12} {
					s := &colorState{matrix: matrix, fullRange: full, depth: depth, ssHor: ssHor}
					s.maxChannel = 1<<depth - 1
					s.outMax = 65535
					s.kr, s.kg, s.kb = yuvCoefficients(matrix)
					s.crCoef, s.cbCoef = 2*(1-s.kr), 2*(1-s.kb)
					s.gcr, s.gcb = s.kr*(1-s.kr), s.kb*(1-s.kb)
					c := s.rowConsts()

					for n := 1; n <= 40; n++ {
						uw := (n + ssHor) >> ssHor
						yf := make([]float32, n)
						a := make([]uint16, n)
						for i := range yf {
							yf[i] = float32(rowGen(iter, 0, i)%1400)/1000 - 0.2
							a[i] = uint16(rowGen(iter, 1, i))
						}
						rows := make([][]float32, 4)
						for r := range rows {
							rows[r] = make([]float32, uw+8)
							for i := range rows[r] {
								rows[r][i] = float32(rowGen(iter, 2+r, i)%1200)/1000 - 0.6
							}
						}

						got := make([]uint8, 8*n+64)
						want := make([]uint8, 8*n+64)
						for i := range got {
							got[i] = uint8(rowGen(iter, 6, i))
							want[i] = got[i]
						}

						asm(got, yf, rows[0], rows[1], rows[2], rows[3], a, n, &c)
						ref(want, yf, rows[0], rows[1], rows[2], rows[3], a, n, &c)

						for i := range want {
							if got[i] == want[i] {
								continue
							}
							t.Fatalf("ssHor=%d matrix=%d full=%v depth=%d n=%d: buf[%d] = %d, want %d",
								ssHor, matrix, full, depth, n, i, got[i], want[i])
						}
						cases++
					}
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}
