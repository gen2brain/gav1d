//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// func itxClipRVV(tmp *int32, n int, rnd int32, shift int, lo, hi int32)
TEXT ·itxClipRVV(SB), NOSPLIT, $0-40
	MOV  tmp+0(FP), X10
	MOV  n+8(FP), X11
	MOVW rnd+16(FP), X12
	MOV  shift+24(FP), X13
	MOVW lo+32(FP), X14
	MOVW hi+36(FP), X15

loop:
	VSETVLI X11, E32, M1, TA, MA, X5
	VLE32V  (X10), V1
	VADDVX  X12, V1, V1
	VSRAVX  X13, V1, V1
	VMAXVX  X14, V1, V1
	VMINVX  X15, V1, V1
	VSE32V  V1, (X10)

	SLLI $2, X5, X6
	ADD  X6, X10
	SUB  X5, X11
	BNEZ X11, loop

	RET

// RESIDUAL rounds one vector of residual and folds the pixels into it.
#define RESIDUAL           \
	VLE32V (X12), V1;  \
	VADDVI $8, V1, V1; \
	VSRAVI $4, V1, V1; \
	VADDVV V2, V1, V1; \
	VMAXVX X0, V1, V1; \
	VMINVX X15, V1, V1

// func itxAdd8RVV(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd8RVV(SB), NOSPLIT, $0-52
	MOV  dst+0(FP), X10
	MOV  stride+8(FP), X11
	MOV  tmp+16(FP), X12
	MOV  w+24(FP), X13
	MOV  h+32(FP), X14
	MOV  ts+40(FP), X18
	MOVW bitdepthMax+48(FP), X15
	SLLI $2, X18, X18

rows8:
	MOV X10, X16
	MOV X13, X17
	MOV X12, X19

cols8:
	VSETVLI  X17, E32, M1, TA, MA, X5
	VSETVLI  X5, E8, MF4, TA, MA, X6
	VLE8V    (X16), V3
	VSETVLI  X5, E32, M1, TA, MA, X6
	VZEXTVF4 V3, V2
	RESIDUAL
	VSETVLI  X5, E16, MF2, TA, MA, X6
	VNSRLWI  $0, V1, V5
	VSETVLI  X5, E8, MF4, TA, MA, X6
	VNSRLWI  $0, V5, V4
	VSE8V    V4, (X16)
	VSETVLI  X5, E32, M1, TA, MA, X6

	SLLI $2, X5, X6
	ADD  X6, X12
	ADD  X5, X16
	SUB  X5, X17
	BNEZ X17, cols8

	ADD  X18, X19, X12
	ADD  X11, X10
	ADDI $-1, X14
	BNEZ X14, rows8

	RET

// func itxAdd16RVV(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd16RVV(SB), NOSPLIT, $0-52
	MOV  dst+0(FP), X10
	MOV  stride+8(FP), X11
	MOV  tmp+16(FP), X12
	MOV  w+24(FP), X13
	MOV  h+32(FP), X14
	MOV  ts+40(FP), X18
	MOVW bitdepthMax+48(FP), X15
	SLLI $1, X11, X11
	SLLI $2, X18, X18

rows16:
	MOV X10, X16
	MOV X13, X17
	MOV X12, X19

cols16:
	VSETVLI  X17, E32, M1, TA, MA, X5
	VSETVLI  X5, E16, MF2, TA, MA, X6
	VLE16V   (X16), V3
	VSETVLI  X5, E32, M1, TA, MA, X6
	VZEXTVF2 V3, V2
	RESIDUAL
	VSETVLI  X5, E16, MF2, TA, MA, X6
	VNSRLWI  $0, V1, V4
	VSE16V   V4, (X16)
	VSETVLI  X5, E32, M1, TA, MA, X6

	SLLI $2, X5, X6
	ADD  X6, X12
	SLLI $1, X5, X6
	ADD  X6, X16
	SUB  X5, X17
	BNEZ X17, cols16

	ADD  X18, X19, X12
	ADD  X11, X10
	ADDI $-1, X14
	BNEZ X14, rows16

	RET

// func itxTransposeRVV(wide, out *int32, w, ws, ts, n int)
TEXT ·itxTransposeRVV(SB), NOSPLIT, $0-48
	MOV  wide+0(FP), X10
	MOV  out+8(FP), X11
	MOV  w+16(FP), X12
	MOV  ws+24(FP), X13
	MOV  ts+32(FP), X15
	MOV  n+40(FP), X14
	SLLI $2, X13, X13
	SLLI $2, X15, X15

trtap:
	MOV X10, X17
	MOV X11, X18
	MOV X14, X19

trlane:
	VSETVLI X19, E32, M1, TA, MA, X5
	VLE32V  (X17), V0
	VSSE32V V0, X15, (X18)

	SLLI $2, X5, X20
	ADD  X20, X17
	MUL  X15, X5, X21
	ADD  X21, X18
	SUB  X5, X19
	BNEZ X19, trlane

	ADD  X13, X10
	ADD  $4, X11
	SUB  $1, X12
	BNEZ X12, trtap

	RET

// func widenCoefs16RVV(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16RVV(SB), NOSPLIT, $0-56
	MOV  dst+0(FP), X10
	MOV  dstStride+8(FP), X11
	SLLI $2, X11, X11
	MOV  src+16(FP), X12
	MOV  srcStride+24(FP), X13
	SLLI $1, X13, X13
	MOV  cols+32(FP), X14
	MOV  n+40(FP), X15
	MOV  lanes+48(FP), X16
	BEQZ X14, wdone

wcol:
	MOV X10, X17
	MOV X12, X18
	MOV X15, X19

wrun:
	BEQZ     X19, wzinit
	VSETVLI  X19, E32, M1, TA, MA, X5
	VLE16V   (X18), V1
	VSEXTVF2 V1, V2
	VSE32V   V2, (X17)
	SLLI     $2, X5, X6
	ADD      X6, X17
	SLLI     $1, X5, X6
	ADD      X6, X18
	SUB      X5, X19
	JMP      wrun

wzinit:
	SUB X15, X16, X19

wzero:
	BEQZ    X19, wnext
	VSETVLI X19, E32, M1, TA, MA, X5
	VMVVX   X0, V1
	VSE32V  V1, (X17)
	SLLI    $2, X5, X6
	ADD     X6, X17
	SUB     X5, X19
	JMP     wzero

wnext:
	ADD  X11, X10
	ADD  X13, X12
	ADDI $-1, X14
	BNEZ X14, wcol

wdone:
	RET

// func widenCoefs16Rect2RVV(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16Rect2RVV(SB), NOSPLIT, $0-56
	MOV  dst+0(FP), X10
	MOV  dstStride+8(FP), X11
	SLLI $2, X11, X11
	MOV  src+16(FP), X12
	MOV  srcStride+24(FP), X13
	SLLI $1, X13, X13
	MOV  cols+32(FP), X14
	MOV  n+40(FP), X15
	MOV  lanes+48(FP), X16
	MOV  $181, X20
	MOV  $128, X21
	BEQZ X14, rdone

rcol:
	MOV X10, X17
	MOV X12, X18
	MOV X15, X19

rrun:
	BEQZ     X19, rzinit
	VSETVLI  X19, E32, M1, TA, MA, X5
	VLE16V   (X18), V1
	VSEXTVF2 V1, V2
	VMULVX   X20, V2, V2
	VADDVX   X21, V2, V2
	VSRAVI   $8, V2, V2
	VSE32V   V2, (X17)
	SLLI     $2, X5, X6
	ADD      X6, X17
	SLLI     $1, X5, X6
	ADD      X6, X18
	SUB      X5, X19
	JMP      rrun

rzinit:
	SUB X15, X16, X19

rzero:
	BEQZ    X19, rnext
	VSETVLI X19, E32, M1, TA, MA, X5
	VMVVX   X0, V1
	VSE32V  V1, (X17)
	SLLI    $2, X5, X6
	ADD     X6, X17
	SUB     X5, X19
	JMP     rzero

rnext:
	ADD  X11, X10
	ADD  X13, X12
	ADDI $-1, X14
	BNEZ X14, rcol

rdone:
	RET

// func itxDC8RVV(dst *uint8, stride int, dc int32, w, h int, bitdepthMax int32)
TEXT ·itxDC8RVV(SB), NOSPLIT, $0-44
	MOV  dst+0(FP), X10
	MOV  stride+8(FP), X11
	MOVW dc+16(FP), X12
	MOV  w+24(FP), X13
	MOV  h+32(FP), X14

	MOV  $0, X15
	BGE  X12, ZERO, pos8
	SUB  X12, ZERO, X12
	MOV  $1, X15

pos8:
	MOV  $255, X16
	BLTU X12, X16, sat8
	MOV  X16, X12

sat8:
	MOV X10, X17

row8:
	MOV X13, X18
	MOV X17, X19

col8:
	VSETVLI X18, E8, M1, TA, MA, X20
	VLE8V   (X19), V1
	BNEZ    X15, s8
	VSADDUVX X12, V1, V1
	JMP      st8

s8:
	VSSUBUVX X12, V1, V1

st8:
	VSE8V V1, (X19)
	ADD   X20, X19, X19
	SUB   X20, X18, X18
	BNEZ  X18, col8

	ADD  X11, X17, X17
	ADD  $-1, X14, X14
	BNEZ X14, row8
	RET

// func itxDC16RVV(dst *uint16, stride int, dc int32, w, h int, bitdepthMax int32)
TEXT ·itxDC16RVV(SB), NOSPLIT, $0-44
	MOV   dst+0(FP), X10
	MOV   stride+8(FP), X11
	MOVW  dc+16(FP), X12
	MOV   w+24(FP), X13
	MOV   h+32(FP), X14
	MOVWU bitdepthMax+40(FP), X21
	SLLI  $1, X11, X11

	MOV  $0, X15
	BGE  X12, ZERO, pos16
	SUB  X12, ZERO, X12
	MOV  $1, X15

pos16:
	MOV  $65535, X16
	BLTU X12, X16, sat16
	MOV  X16, X12

sat16:
	MOV X10, X17

row16:
	MOV X13, X18
	MOV X17, X19

col16:
	VSETVLI X18, E16, M1, TA, MA, X20
	VLE16V  (X19), V1
	BNEZ    X15, s16
	VSADDUVX X12, V1, V1
	VMINUVX  X21, V1, V1
	JMP      st16

s16:
	VSSUBUVX X12, V1, V1

st16:
	VSE16V V1, (X19)
	SLLI   $1, X20, X22
	ADD    X22, X19, X19
	SUB    X20, X18, X18
	BNEZ   X18, col16

	ADD  X11, X17, X17
	ADD  $-1, X14, X14
	BNEZ X14, row16
	RET
