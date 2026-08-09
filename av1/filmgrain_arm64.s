//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no vector multiply and no saturating narrow, so both
// are encoded by hand.
#define SQRDMULH(Vd, Vn, Vm) WORD $(0x6E60B400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SQXTUN(Vd, Vn)       WORD $(0x2E212800 | ((Vn) << 5) | (Vd))

// LOOKUP loads scaling[src[I]] shifted up by lshift into lane I. NEON has no
// gather, so the table lookup stays scalar.
#define LOOKUP(I)             \
	MOVBU I(R1), R8;      \
	MOVBU (R2)(R8), R9;   \
	LSL   R5, R9, R9;     \
	VMOV  R9, V1.H[I]

// func fgApplyRowNEON(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)
//
// Eight pixels an iteration. Pre-shifting the scaling value by 15-lshift turns
// the rounding shift into a rounding multiply-high, which SQRDMULH does in one
// instruction without overflowing a halfword.
TEXT ·fgApplyRowNEON(SB), NOSPLIT, $0-64
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD scaling+16(FP), R2
	MOVD grain+24(FP), R3
	MOVD n+32(FP), R4
	MOVD lshift+40(FP), R5
	MOVD minValue+48(FP), R6
	MOVD maxValue+56(FP), R7

	VDUP R6, V6.B16
	VDUP R7, V7.B16

loop:
	LOOKUP(0)
	LOOKUP(1)
	LOOKUP(2)
	LOOKUP(3)
	LOOKUP(4)
	LOOKUP(5)
	LOOKUP(6)
	LOOKUP(7)

	VLD1   (R1), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VLD1   (R3), [V2.H8]

	SQRDMULH(2, 2, 1)
	VADD V0.H8, V2.H8, V2.H8
	SQXTUN(2, 2)

	VUMAX V6.B8, V2.B8, V2.B8
	VUMIN V7.B8, V2.B8, V2.B8
	VST1  [V2.B8], (R0)

	ADD  $8, R0
	ADD  $8, R1
	ADD  $16, R3
	SUB  $8, R4
	CMP  $8, R4
	BGE  loop

	CBZ R4, done

	// Finish with a block overlapping the one just done. Every pixel is an
	// independent function of src and grain, and dst is never the source.
	MOVD $8, R10
	SUB  R4, R10, R10
	SUB  R10, R0
	SUB  R10, R1
	LSL  $1, R10, R11
	SUB  R11, R3
	MOVD $8, R4
	B    loop

done:
	RET
