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

// func itxAdd8RVV(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd8RVV(SB), NOSPLIT, $0-44
	MOV  dst+0(FP), X10
	MOV  stride+8(FP), X11
	MOV  tmp+16(FP), X12
	MOV  w+24(FP), X13
	MOV  h+32(FP), X14
	MOVW bitdepthMax+40(FP), X15

rows8:
	MOV X10, X16
	MOV X13, X17

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

	ADD  X11, X10
	ADDI $-1, X14
	BNEZ X14, rows8

	RET

// func itxAdd16RVV(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd16RVV(SB), NOSPLIT, $0-44
	MOV  dst+0(FP), X10
	MOV  stride+8(FP), X11
	MOV  tmp+16(FP), X12
	MOV  w+24(FP), X13
	MOV  h+32(FP), X14
	MOVW bitdepthMax+40(FP), X15
	SLLI $1, X11, X11

rows16:
	MOV X10, X16
	MOV X13, X17

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

	ADD  X11, X10
	ADDI $-1, X14
	BNEZ X14, rows16

	RET
