//go:build arm64 && !noasm

#include "textflag.h"

// The rows are at most twelve samples, so the wide case covers them with two
// overlapping stores rather than a tail loop.

// func cdefCopy8NEON(tmp *int16, tmpStride int, src *uint8, srcStride, n, h int)
TEXT ·cdefCopy8NEON(SB), NOSPLIT, $0-48
	MOVD tmp+0(FP), R0
	MOVD tmpStride+8(FP), R1
	LSL  $1, R1, R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD n+32(FP), R4
	MOVD h+40(FP), R5

	CMP  $8, R4
	BLT  mid8
	SUB  $8, R4, R6
	ADD  R6, R2, R6
	SUB  $8, R4, R7
	ADD  R7<<1, R0, R7

wide8:
	VLD1   (R2), [V0.D1]
	VUSHLL $0, V0.B8, V1.H8
	VST1   [V1.H8], (R0)
	VLD1   (R6), [V0.D1]
	VUSHLL $0, V0.B8, V1.H8
	VST1   [V1.H8], (R7)
	ADD    R3, R2, R2
	ADD    R3, R6, R6
	ADD    R1, R0, R0
	ADD    R1, R7, R7
	SUBS   $1, R5, R5
	BNE    wide8
	RET

mid8:
	CMP $4, R4
	BLT slow8
	SUB $4, R4, R6
	ADD R6, R2, R6
	SUB $4, R4, R7
	ADD R7<<1, R0, R7

loopmid8:
	VLD1   (R2), [V0.S2]
	VUSHLL $0, V0.B8, V1.H8
	VST1   [V1.D1], (R0)
	VLD1   (R6), [V0.S2]
	VUSHLL $0, V0.B8, V1.H8
	VST1   [V1.D1], (R7)
	ADD    R3, R2, R2
	ADD    R3, R6, R6
	ADD    R1, R0, R0
	ADD    R1, R7, R7
	SUBS   $1, R5, R5
	BNE    loopmid8
	RET

slow8:
	CBZ R4, done8

rows8:
	MOVD $0, R8

cols8:
	MOVBU (R2)(R8), R9
	MOVH  R9, (R0)(R8<<1)
	ADD   $1, R8, R8
	CMP   R4, R8
	BLT   cols8
	ADD   R3, R2, R2
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   rows8

done8:
	RET

// func cdefCopy16NEON(tmp *int16, tmpStride int, src *uint16, srcStride, n, h int)
TEXT ·cdefCopy16NEON(SB), NOSPLIT, $0-48
	MOVD tmp+0(FP), R0
	MOVD tmpStride+8(FP), R1
	LSL  $1, R1, R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	LSL  $1, R3, R3
	MOVD n+32(FP), R4
	MOVD h+40(FP), R5

	CMP  $8, R4
	BLT  mid16
	SUB  $8, R4, R6
	ADD  R6<<1, R2, R6
	SUB  $8, R4, R7
	ADD  R7<<1, R0, R7

wide16:
	VLD1 (R2), [V0.H8]
	VST1 [V0.H8], (R0)
	VLD1 (R6), [V1.H8]
	VST1 [V1.H8], (R7)
	ADD  R3, R2, R2
	ADD  R3, R6, R6
	ADD  R1, R0, R0
	ADD  R1, R7, R7
	SUBS $1, R5, R5
	BNE  wide16
	RET

mid16:
	CMP $4, R4
	BLT slow16
	SUB $4, R4, R6
	ADD R6<<1, R2, R6
	SUB $4, R4, R7
	ADD R7<<1, R0, R7

loopmid16:
	VLD1 (R2), [V0.D1]
	VST1 [V0.D1], (R0)
	VLD1 (R6), [V1.D1]
	VST1 [V1.D1], (R7)
	ADD  R3, R2, R2
	ADD  R3, R6, R6
	ADD  R1, R0, R0
	ADD  R1, R7, R7
	SUBS $1, R5, R5
	BNE  loopmid16
	RET

slow16:
	CBZ R4, done16

rows16:
	MOVD $0, R8

cols16:
	MOVHU (R2)(R8<<1), R9
	MOVH  R9, (R0)(R8<<1)
	ADD   $1, R8, R8
	CMP   R4, R8
	BLT   cols16
	ADD   R3, R2, R2
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   rows16

done16:
	RET
