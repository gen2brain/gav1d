//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// LOADTAPS sign-extends the eight coefficients into scalar registers; the
// vector multiplies take them from there.
#define LOADTAPS(REG)     \
	MOVB 0(REG), X17; \
	MOVB 1(REG), X18; \
	MOVB 2(REG), X19; \
	MOVB 3(REG), X20; \
	MOVB 4(REG), X21; \
	MOVB 5(REG), X22; \
	MOVB 6(REG), X23; \
	MOVB 7(REG), X24

// TAPS8 filters vl pixels from X30, stepping between taps by STEP. Products and
// their sum stay inside a halfword, so the loads widen once and stop there.
#define TAPS8(STEP)                      \
	VSETVLI  X28, E8, MF2, TA, MA, X29; \
	VLE8V    (X30), V1;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V2;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V3;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V4;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V5;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V6;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V7;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V8;              \
	VSETVLI  X28, E16, M1, TA, MA, X29; \
	VZEXTVF2 V1, V9;                 \
	VMULVX   X17, V9, V10;           \
	VZEXTVF2 V2, V9;                 \
	VMULVX   X18, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V3, V9;                 \
	VMULVX   X19, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V4, V9;                 \
	VMULVX   X20, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V5, V9;                 \
	VMULVX   X21, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V6, V9;                 \
	VMULVX   X22, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V7, V9;                 \
	VMULVX   X23, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V8, V9;                 \
	VMULVX   X24, V9, V9;            \
	VADDVV   V9, V10, V10

// STOREPX clips to the pixel range and narrows back to bytes.
#define STOREPX(DST)                     \
	VMAXVX   X0, V10, V10;           \
	VMINVX   X26, V10, V10;          \
	VSETVLI  X28, E8, MF2, TA, MA, X29; \
	VNSRLWI  $0, V10, V11;           \
	VSE8V    V11, (DST);             \
	VSETVLI  X28, E16, M1, TA, MA, X29

// func put8tapHRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapHRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV f+32(FP), X14
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	LOADTAPS(X14)

	MOV $512, X25
	MOV $255, X26
	MOV $1, X6

hrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

hcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X6)
	VADDVI  $2, V10, V10
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, hcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, hrow

	RET

// func put8tapVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapVRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV f+32(FP), X14
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	LOADTAPS(X14)

	MOV $512, X25
	MOV $255, X26

vrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

vcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X13)
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, vcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, vrow

	RET

// func put8tapHVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh, fv *[8]int8, w, h int)
TEXT ·put8tapHVRVV(SB), NOSPLIT, $0-72
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV mid+32(FP), X5
	MOV w+56(FP), X15
	MOV h+64(FP), X16

	MOV $255, X26
	MOV $1, X6

hvstrip:
	// A strip is capped at one vector and at what the scratch holds, so
	// neither pass needs a column loop.
	MOV     $16, X29
	MIN     X15, X29, X29
	VSETVLI X29, E16, M1, TA, MA, X28
	SLLI    $1, X28, X14

	MOV  fh+40(FP), X25
	LOADTAPS(X25)
	MOV  $8192, X25
	MOV  X12, X8
	MOV  X5, X9
	MOV  X16, X7
	ADDI $7, X7

hvh:
	MOV     X8, X30
	TAPS8(X6)
	VSMULVX X25, V10, V10
	VSE16V  V10, (X9)

	ADD  X13, X8
	ADD  X14, X9
	ADDI $-1, X7
	BNEZ X7, hvh

	MOV fv+48(FP), X25
	LOADTAPS(X25)
	MOV X5, X9
	MOV X10, X8
	MOV X16, X7

hvv:
	MOV      X9, X30
	VLE16V   (X30), V1
	VWMULVX  X17, V1, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X18, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X19, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X20, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X21, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X22, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X23, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X24, V16

	// vnclip rounds the way rnd8 does, so the shift and the bias are one step.
	VNCLIPWI $10, V16, V10
	STOREPX(X8)

	ADD  X14, X9
	ADD  X11, X8
	ADDI $-1, X7
	BNEZ X7, hvv

	ADD  X28, X12
	ADD  X28, X10
	SUB  X28, X15
	BNEZ X15, hvstrip

	RET
