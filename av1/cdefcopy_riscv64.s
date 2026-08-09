//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// The vector length setting clamps itself to the row, so one pass covers any
// width without a tail.

// func cdefCopy8RVV(tmp *int16, tmpStride int, src *uint8, srcStride, n, h int)
TEXT ·cdefCopy8RVV(SB), NOSPLIT, $0-48
	MOV  tmp+0(FP), X10
	MOV  tmpStride+8(FP), X11
	SLLI $1, X11, X11
	MOV  src+16(FP), X12
	MOV  srcStride+24(FP), X13
	MOV  n+32(FP), X14
	MOV  h+40(FP), X15

rows8:
	VSETVLI  X14, E8, MF2, TA, MA, X7
	VLE8V    (X12), V1
	VSETVLI  X14, E16, M1, TA, MA, X7
	VZEXTVF2 V1, V2
	VSE16V   V2, (X10)
	ADD      X13, X12, X12
	ADD      X11, X10, X10
	SUB      $1, X15, X15
	BNE      X0, X15, rows8

	RET

// func cdefCopy16RVV(tmp *int16, tmpStride int, src *uint16, srcStride, n, h int)
TEXT ·cdefCopy16RVV(SB), NOSPLIT, $0-48
	MOV  tmp+0(FP), X10
	MOV  tmpStride+8(FP), X11
	SLLI $1, X11, X11
	MOV  src+16(FP), X12
	MOV  srcStride+24(FP), X13
	SLLI $1, X13, X13
	MOV  n+32(FP), X14
	MOV  h+40(FP), X15

rows16:
	VSETVLI X14, E16, M1, TA, MA, X7
	VLE16V  (X12), V1
	VSE16V  V1, (X10)
	ADD     X13, X12, X12
	ADD     X11, X10, X10
	SUB     $1, X15, X15
	BNE     X0, X15, rows16

	RET
