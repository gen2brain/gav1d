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
