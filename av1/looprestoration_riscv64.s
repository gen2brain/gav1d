//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// TAPS keeps the seven coefficients in scalar registers, which is what the
// vector-scalar multiply-accumulate wants.
#define TAPS                 \
	MOV  c+24(FP), X13;  \
	MOVH 0(X13), X14;    \
	MOVH 2(X13), X15;    \
	MOVH 4(X13), X16;    \
	MOVH 6(X13), X17;    \
	MOVH 8(X13), X18;    \
	MOVH 10(X13), X19;   \
	MOVH 12(X13), X20

// HLOAD walks one cursor across the seven overlapping windows.
#define HLOAD(LOAD, STEP)     \
	MOV   X11, X6;        \
	LOAD  (X6), V10;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V11;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V12;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V13;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V14;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V15;      \
	ADD   $STEP, X6, X6;  \
	LOAD  (X6), V16

// ACC widens each window and folds it in. The rounding term seeds the
// accumulator, so the taps need no separate add.
#define ACC(EXT)             \
	VMVVX    X21, V20;   \
	EXT      V10, V17;   \
	VMACCVX  V17, X14, V20; \
	EXT      V11, V17;   \
	VMACCVX  V17, X15, V20; \
	EXT      V12, V17;   \
	VMACCVX  V17, X16, V20; \
	EXT      V13, V17;   \
	VMACCVX  V17, X17, V20; \
	EXT      V14, V17;   \
	VMACCVX  V17, X18, V20; \
	EXT      V15, V17;   \
	VMACCVX  V17, X19, V20; \
	EXT      V16, V17;   \
	VMACCVX  V17, X20, V20; \
	VSRAVX   X22, V20, V20; \
	VMAXVX   X0, V20, V20; \
	VMINVX   X23, V20, V20

// VROWS resolves the seven row pointers, which advance independently.
#define VROWS                 \
	MOV  hor+8(FP), X7;   \
	MOV  rows+16(FP), X12; \
	MOV  0(X12), X11;     \
	SLLI $1, X11, X11;    \
	ADD  X7, X11, X11;    \
	MOV  8(X12), X24;     \
	SLLI $1, X24, X24;    \
	ADD  X7, X24, X24;    \
	MOV  16(X12), X25;    \
	SLLI $1, X25, X25;    \
	ADD  X7, X25, X25;    \
	MOV  24(X12), X26;    \
	SLLI $1, X26, X26;    \
	ADD  X7, X26, X26;    \
	MOV  32(X12), X28;    \
	SLLI $1, X28, X28;    \
	ADD  X7, X28, X28;    \
	MOV  40(X12), X29;    \
	SLLI $1, X29, X29;    \
	ADD  X7, X29, X29;    \
	MOV  48(X12), X30;    \
	SLLI $1, X30, X30;    \
	ADD  X7, X30, X30;    \
	MOV  n+32(FP), X8

#define VLOAD             \
	VLE16V (X11), V10; \
	VLE16V (X24), V11; \
	VLE16V (X25), V12; \
	VLE16V (X26), V13; \
	VLE16V (X28), V14; \
	VLE16V (X29), V15; \
	VLE16V (X30), V16

#define VSTEP             \
	SLLI $1, X9, X6;  \
	ADD  X6, X11, X11; \
	ADD  X6, X24, X24; \
	ADD  X6, X25, X25; \
	ADD  X6, X26, X26; \
	ADD  X6, X28, X28; \
	ADD  X6, X29, X29; \
	ADD  X6, X30, X30; \
	SUB  X9, X8, X8

// func wienerH8RVV(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH8RVV(SB), NOSPLIT, $0-52
	MOV  dst+0(FP), X10
	MOV  src+8(FP), X11
	MOV  n+16(FP), X8
	TAPS
	MOVW rnd+32(FP), X21
	MOV  shift+40(FP), X22
	MOVW limit+48(FP), X23

loop8:
	VSETVLI X8, E8, MF4, TA, MA, X9
	HLOAD(VLE8V, 1)
	VSETVLI X8, E32, M1, TA, MA, X9
	ACC(VZEXTVF4)
	VSETVLI X8, E16, MF2, TA, MA, X9
	VNSRLWI $0, V20, V21
	VSE16V  V21, (X10)

	ADD  X9, X11, X11
	SLLI $1, X9, X6
	ADD  X6, X10, X10
	SUB  X9, X8, X8
	BNE  X0, X8, loop8

	RET

// func wienerH16RVV(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH16RVV(SB), NOSPLIT, $0-52
	MOV  dst+0(FP), X10
	MOV  src+8(FP), X11
	MOV  n+16(FP), X8
	TAPS
	MOVW rnd+32(FP), X21
	MOV  shift+40(FP), X22
	MOVW limit+48(FP), X23

loop16:
	VSETVLI X8, E16, MF2, TA, MA, X9
	HLOAD(VLE16V, 2)
	VSETVLI X8, E32, M1, TA, MA, X9
	ACC(VZEXTVF2)
	VSETVLI X8, E16, MF2, TA, MA, X9
	VNSRLWI $0, V20, V21
	VSE16V  V21, (X10)

	SLLI $1, X9, X6
	ADD  X6, X11, X11
	ADD  X6, X10, X10
	SUB  X9, X8, X8
	BNE  X0, X8, loop16

	RET

// func wienerV8RVV(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV8RVV(SB), NOSPLIT, $0-60
	MOV  p+0(FP), X10
	VROWS
	TAPS
	MOVW rnd+40(FP), X21
	MOV  shift+48(FP), X22
	MOVW limit+56(FP), X23

vloop8:
	VSETVLI X8, E16, MF2, TA, MA, X9
	VLOAD
	VSETVLI X8, E32, M1, TA, MA, X9
	ACC(VZEXTVF2)
	VSETVLI X8, E16, MF2, TA, MA, X9
	VNSRLWI $0, V20, V21
	VSETVLI X8, E8, MF4, TA, MA, X9
	VNSRLWI $0, V21, V22
	VSE8V   V22, (X10)

	ADD X9, X10, X10
	VSTEP
	BNE X0, X8, vloop8

	RET

// func wienerV16RVV(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV16RVV(SB), NOSPLIT, $0-60
	MOV  p+0(FP), X10
	VROWS
	TAPS
	MOVW rnd+40(FP), X21
	MOV  shift+48(FP), X22
	MOVW limit+56(FP), X23

vloop16:
	VSETVLI X8, E16, MF2, TA, MA, X9
	VLOAD
	VSETVLI X8, E32, M1, TA, MA, X9
	ACC(VZEXTVF2)
	VSETVLI X8, E16, MF2, TA, MA, X9
	VNSRLWI $0, V20, V21
	VSE16V  V21, (X10)

	SLLI $1, X9, X6
	ADD  X6, X10, X10
	VSTEP
	BNE  X0, X8, vloop16

	RET
