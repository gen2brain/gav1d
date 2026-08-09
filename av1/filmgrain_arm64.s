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

// UVLOOKUP computes the chroma scaling index for lane I and loads the shifted
// table entry into it. NEON has no gather, so the index arithmetic is scalar
// too, which is why no widening multiply is needed here. Branchless: sx folds
// into the average, and a conditional select picks between it and the mix.
#define UVLOOKUP(I)            \
	MOVD  $I, R13;         \
	LSL   R7, R13, R13;    \
	MOVBU (R2)(R13), R14;  \
	ADD   R7, R13, R15;    \
	MOVBU (R2)(R15), R15;  \
	MUL   R7, R15, R15;    \
	ADD   R15, R14, R14;   \
	ADD   R7, R14, R14;    \
	LSR   R7, R14, R14;    \
	MOVBU I(R1), R15;      \
	MUL   R9, R14, R13;    \
	MADD  R10, R13, R15, R13; \
	ASR   $6, R13, R13;    \
	ADD   R11, R13, R13;   \
	CMP   ZR, R13;         \
	CSEL  LT, ZR, R13, R13; \
	CMP   R12, R13;        \
	CSEL  GT, R12, R13, R13; \
	CMP   ZR, R8;          \
	CSEL  NE, R14, R13, R13; \
	MOVBU (R3)(R13), R15;  \
	LSL   R5, R15, R15;    \
	VMOV  R15, V1.H[I]

// func fguvApplyRowNEON(dst, src, luma, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int, p *fguvParams)
TEXT ·fguvApplyRowNEON(SB), NOSPLIT, $0-80
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD luma+16(FP), R2
	MOVD scaling+24(FP), R3
	MOVD grain+32(FP), R4
	MOVD lshift+48(FP), R5
	MOVD n+40(FP), R6

	MOVD minValue+56(FP), R13
	VDUP R13, V6.B16
	MOVD maxValue+64(FP), R13
	VDUP R13, V7.B16

	MOVD p+72(FP), R13
	MOVH 0(R13), R9   // lumaMult
	MOVH 2(R13), R10  // mult
	MOVH 4(R13), R12  // pixelMax
	MOVW 8(R13), R11  // offset
	MOVW 12(R13), R7  // sx
	MOVW 16(R13), R8  // chromaScalingFromLuma

uvloop:
	UVLOOKUP(0)
	UVLOOKUP(1)
	UVLOOKUP(2)
	UVLOOKUP(3)
	UVLOOKUP(4)
	UVLOOKUP(5)
	UVLOOKUP(6)
	UVLOOKUP(7)

	VLD1   (R1), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VLD1   (R4), [V2.H8]

	SQRDMULH(2, 2, 1)
	VADD V0.H8, V2.H8, V2.H8
	SQXTUN(2, 2)

	VUMAX V6.B8, V2.B8, V2.B8
	VUMIN V7.B8, V2.B8, V2.B8
	VST1  [V2.B8], (R0)

	ADD  $8, R0
	ADD  $8, R1
	MOVD $8, R13
	LSL  R7, R13, R13
	ADD  R13, R2
	ADD  $16, R4
	SUB  $8, R6
	CMP  $8, R6
	BGE  uvloop

	CBZ R6, uvdone

	MOVD $8, R13
	SUB  R6, R13, R13
	SUB  R13, R0
	SUB  R13, R1
	MOVD R13, R14
	LSL  R7, R14, R14
	SUB  R14, R2
	LSL  $1, R13, R14
	SUB  R14, R4
	MOVD $8, R6
	B    uvloop

uvdone:
	RET
