//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// func fgApplyRowRVV(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)
//
// Length agnostic: vsetvli clamps the last iteration, so there is no tail.
// Pre-shifting the scaling value turns the rounding shift into vsmul.
TEXT ·fgApplyRowRVV(SB), NOSPLIT, $0-64
	MOV dst+0(FP), X10
	MOV src+8(FP), X11
	MOV scaling+16(FP), X12
	MOV grain+24(FP), X13
	MOV n+32(FP), X14
	MOV lshift+40(FP), X15
	MOV minValue+48(FP), X16
	MOV maxValue+56(FP), X17

loop:
	VSETVLI X14, E16, M1, TA, MA, X5

	// src bytes, widened to the index and to the addend
	VSETVLI   X5, E8, MF2, TA, MA, X6
	VLE8V     (X11), V1
	VSETVLI   X5, E16, M1, TA, MA, X6
	VZEXTVF2  V1, V2

	// scaling[src], gathered by the widened index
	VSETVLI   X5, E8, MF2, TA, MA, X6
	VLUXEI16V (X12), V2, V3
	VSETVLI   X5, E16, M1, TA, MA, X6
	VZEXTVF2  V3, V4
	VSLLVX    X15, V4, V4

	VLE16V  (X13), V5
	VSMULVV V4, V5, V6
	VADDVV  V2, V6, V6

	VMAXVX X16, V6, V6
	VMINVX X17, V6, V6

	VSETVLI  X5, E8, MF2, TA, MA, X6
	VNSRLWI  $0, V6, V7
	VSE8V    V7, (X10)

	ADD  X5, X10
	ADD  X5, X11
	SLLI $1, X5, X6
	ADD  X6, X13
	SUB  X5, X14
	BNEZ X14, loop

	RET

// func fguvApplyRowRVV(dst, src, luma, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int, p *fguvParams)
//
// As above, but the index comes from the co-located luma; a segment load
// splits the subsampled pairs for the average.
TEXT ·fguvApplyRowRVV(SB), NOSPLIT, $0-80
	MOV dst+0(FP), X10
	MOV src+8(FP), X11
	MOV luma+16(FP), X12
	MOV scaling+24(FP), X13
	MOV grain+32(FP), X14
	MOV n+40(FP), X15
	MOV lshift+48(FP), X16
	MOV minValue+56(FP), X17
	MOV maxValue+64(FP), X18
	MOV p+72(FP), X19

	MOVH 0(X19), X20  // lumaMult
	MOVH 2(X19), X21  // mult
	MOVH 4(X19), X22  // pixelMax
	MOVW 8(X19), X23  // offset
	MOVW 12(X19), X24 // sx
	MOVW 16(X19), X25 // csfl

loopuv:
	VSETVLI X15, E16, M1, TA, MA, X5

	// avg, pairwise averaged when chroma is subsampled
	VSETVLI X5, E8, MF2, TA, MA, X6
	BNEZ    X24, subsampled
	VLE8V   (X12), V1
	VSETVLI X5, E16, M1, TA, MA, X6
	VZEXTVF2 V1, V2
	JMP     haveavg

subsampled:
	VLSEG2E8V (X12), V1
	VWADDUVV  V1, V2, V3
	VSETVLI   X5, E16, M1, TA, MA, X6
	VADDVI    $1, V3, V3
	VSRLVI    $1, V3, V2

haveavg:
	VSETVLI  X5, E8, MF2, TA, MA, X6
	VLE8V    (X11), V6
	VSETVLI  X5, E16, M1, TA, MA, X6
	VZEXTVF2 V6, V7

	BNEZ X25, fromluma

	// val = clip((avg*lumaMult + src*mult)>>6 + offset, 0, pixelMax)
	VWMULVX X20, V2, V16
	VWMULVX X21, V7, V18
	VSETVLI X5, E32, M2, TA, MA, X6
	VADDVV  V16, V18, V16
	VSETVLI X5, E16, M1, TA, MA, X6
	VNSRAWI $6, V16, V8
	VADDVX  X23, V8, V8
	VMAXVX  X0, V8, V8
	VMINVX  X22, V8, V8
	JMP     haveval

fromluma:
	VMVVV V2, V8

haveval:
	VSETVLI   X5, E8, MF2, TA, MA, X6
	VLUXEI16V (X13), V8, V9
	VSETVLI   X5, E16, M1, TA, MA, X6
	VZEXTVF2  V9, V12
	VSLLVX    X16, V12, V12

	VLE16V  (X14), V13
	VSMULVV V12, V13, V14
	VADDVV  V7, V14, V14

	VMAXVX X17, V14, V14
	VMINVX X18, V14, V14

	VSETVLI X5, E8, MF2, TA, MA, X6
	VNSRLWI $0, V14, V15
	VSE8V   V15, (X10)

	ADD  X5, X10
	ADD  X5, X11
	SLL  X24, X5, X6
	ADD  X6, X12
	SLLI $1, X5, X6
	ADD  X6, X14
	SUB  X5, X15
	BNEZ X15, loopuv

	RET
