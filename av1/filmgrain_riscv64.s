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

// func fgApplyRow16RVV(dst, src *uint16, scaling *uint8, grain *int16, n, shift, minValue, maxValue int)
TEXT ·fgApplyRow16RVV(SB), NOSPLIT, $0-64
	MOV  dst+0(FP), X10
	MOV  src+8(FP), X11
	MOV  scaling+16(FP), X12
	MOV  grain+24(FP), X13
	MOV  n+32(FP), X14
	MOV  shift+40(FP), X15
	MOV  $1, X16
	SLL  X15, X16, X16
	SRL  $1, X16, X16
	MOV  minValue+48(FP), X17
	MOV  maxValue+56(FP), X18

fgrow16:
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VLE16V   (X11), V1
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF2 V1, V2
	VSETVLI  X14, E8, MF4, TA, MA, X5
	VLUXEI32V (X12), V2, V4
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF4 V4, V5
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VLE16V   (X13), V6
	VSETVLI  X14, E32, M1, TA, MA, X5
	VSEXTVF2 V6, V7
	VMULVV   V7, V5, V5
	VADDVX   X16, V5, V5
	VSRAVX   X15, V5, V5
	VADDVV   V2, V5, V5
	VMAXVX   X17, V5, V5
	VMINVX   X18, V5, V5
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VNSRLWI  $0, V5, V8
	VSE16V   V8, (X10)
	VSETVLI  X14, E32, M1, TA, MA, X5

	SLLI $1, X5, X6
	ADD  X6, X10, X10
	ADD  X6, X11, X11
	ADD  X6, X13, X13
	SUB  X5, X14, X14
	BNE  X0, X14, fgrow16

	RET

// func fguvApplyRow16RVV(dst, src, luma *uint16, scaling *uint8, grain *int16, p *fguv16Params)
TEXT ·fguvApplyRow16RVV(SB), NOSPLIT, $0-48
	MOV  dst+0(FP), X10
	MOV  src+8(FP), X11
	MOV  luma+16(FP), X19
	MOV  scaling+24(FP), X12
	MOV  grain+32(FP), X13
	MOV  p+40(FP), X20
	MOV  0(X20), X14
	MOV  8(X20), X15
	MOV  $1, X16
	SLL  X15, X16, X16
	SRL  $1, X16, X16
	MOV  16(X20), X17
	MOV  24(X20), X18
	MOV  32(X20), X21
	MOV  40(X20), X22
	MOV  48(X20), X23
	MOV  56(X20), X24
	MOV  64(X20), X25
	MOV  72(X20), X26
	MOV  $2, X28
	SLL  X26, X28, X28

fguvrow16:
	VSETVLI  X14, E16, MF2, TA, MA, X5
	BNE      X0, X26, subx
	VLE16V   (X19), V1
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF2 V1, V2
	JMP      havelum

subx:
	VLSE16V  (X19), X28, V1
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF2 V1, V2
	ADD      $2, X19, X6
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VLSE16V  (X6), X28, V3
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF2 V3, V4
	VADDVV   V4, V2, V2
	VADDVI   $1, V2, V2
	VSRLVI   $1, V2, V2

havelum:
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VLE16V   (X11), V9
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF2 V9, V10
	BNE      X0, X21, idxdone
	VMULVX   X22, V2, V2
	VMULVX   X23, V10, V11
	VADDVV   V11, V2, V2
	VSRAVI   $6, V2, V2
	VADDVX   X24, V2, V2
	VMAXVX   X0, V2, V2
	VMINVX   X25, V2, V2

idxdone:
	VSETVLI  X14, E8, MF4, TA, MA, X5
	VLUXEI32V (X12), V2, V12
	VSETVLI  X14, E32, M1, TA, MA, X5
	VZEXTVF4 V12, V13
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VLE16V   (X13), V14
	VSETVLI  X14, E32, M1, TA, MA, X5
	VSEXTVF2 V14, V15
	VMULVV   V15, V13, V13
	VADDVX   X16, V13, V13
	VSRAVX   X15, V13, V13
	VADDVV   V10, V13, V13
	VMAXVX   X17, V13, V13
	VMINVX   X18, V13, V13
	VSETVLI  X14, E16, MF2, TA, MA, X5
	VNSRLWI  $0, V13, V16
	VSE16V   V16, (X10)
	VSETVLI  X14, E32, M1, TA, MA, X5

	SLLI $1, X5, X6
	ADD  X6, X10, X10
	ADD  X6, X11, X11
	ADD  X6, X13, X13
	MUL  X5, X28, X6
	ADD  X6, X19, X19
	SUB  X5, X14, X14
	BNE  X0, X14, fguvrow16

	RET

// func grainARRVV(pre *int32, row *int16, rowStride, lag int, coeffs *int8, n int)
TEXT ·grainARRVV(SB), NOSPLIT, $0-48
	MOV  pre+0(FP), X10
	MOV  row+8(FP), X11
	MOV  rowStride+16(FP), X12
	MOV  lag+24(FP), X13
	MOV  coeffs+32(FP), X14
	MOV  n+40(FP), X15

	SLLI $1, X12, X12
	MOV  X13, X16
	SLLI $1, X13, X17
	ADDI $1, X17, X17

argrow:
	MOV X11, X18
	MOV X17, X19

argtap:
	MOVB (X14), X20
	ADDI $1, X14, X14

	MOV X18, X21
	MOV X10, X22
	MOV X15, X23

argcol:
	VSETVLI  X23, E16, MF2, TA, MA, X24
	VLE16V   (X21), V1
	VSETVLI  X23, E32, M1, TA, MA, X24
	VSEXTVF2 V1, V2
	VLE32V   (X22), V3
	VMACCVX  V2, X20, V3
	VSE32V   V3, (X22)

	SLLI $1, X24, X25
	ADD  X25, X21
	SLLI $2, X24, X25
	ADD  X25, X22
	SUB  X24, X23
	BNEZ X23, argcol

	ADDI $2, X18, X18
	ADDI $-1, X19
	BNEZ X19, argtap

	ADD  X12, X11
	ADDI $-1, X16
	BNEZ X16, argrow

	RET
