//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// TAP accumulates one neighbour. The out-of-frame fill saturates the threshold
// subtract to zero, so it contributes nothing without needing a mask.
#define TAP(OP, OFFR, THRV, SHR, TAPR) \
	OP       OFFR, X12, X6;  \
	VLE16V   (X6), V5;       \
	VSUBVV   V1, V5, V6;     \
	VRSUBVX  X0, V6, V7;     \
	VMAXVV   V6, V7, V7;     \
	VSRLVX   SHR, V7, V8;    \
	VSSUBUVV V8, THRV, V9;   \
	VMINUVV  V7, V9, V10;    \
	VSRAVI   $15, V6, V11;   \
	VXORVV   V11, V10, V10;  \
	VSUBVV   V11, V10, V10;  \
	VMULVX   TAPR, V10, V10; \
	VADDVV   V10, V2, V2

// TAPC also tracks the range the result is clipped to; the fill loses the
// signed maximum, and read as unsigned it loses the unsigned minimum.
#define TAPC(OP, OFFR, THRV, SHR, TAPR) \
	TAP(OP, OFFR, THRV, SHR, TAPR); \
	VMAXVV  V5, V3, V3;      \
	VMINUVV V5, V4, V4

#define SETUP                     \
	MOV     dst+0(FP), X10;   \
	MOV     dstStride+8(FP), X11; \
	MOV     tmp+16(FP), X12;  \
	MOV     p+24(FP), X13;    \
	MOVW    0(X13), X18;      \
	MOVW    4(X13), X19;      \
	MOVW    8(X13), X20;      \
	MOVW    12(X13), X21;     \
	MOVW    16(X13), X22;     \
	MOVW    20(X13), X23;     \
	SLLI    $1, X18, X18;     \
	SLLI    $1, X19, X19;     \
	SLLI    $1, X20, X20;     \
	SLLI    $1, X21, X21;     \
	SLLI    $1, X22, X22;     \
	SLLI    $1, X23, X23;     \
	MOVW    32(X13), X8;      \
	MOVW    36(X13), X9;      \
	MOVW    56(X13), X14;     \
	MOVW    60(X13), X7;      \
	VSETVLI X14, E16, M1, TA, MA, X5; \
	MOVW    24(X13), X6;      \
	VMVVX   X6, V12;          \
	MOVW    28(X13), X6;      \
	VMVVX   X6, V13

#define LOADPX                    \
	VSETVLI  X5, E8, MF2, TA, MA, X6; \
	VLE8V    (X10), V14;      \
	VSETVLI  X5, E16, M1, TA, MA, X6; \
	VZEXTVF2 V14, V1;         \
	VMVVX    X0, V2

#define FINISH                    \
	VSRAVI   $15, V2, V11;    \
	VADDVV   V11, V2, V2;     \
	VADDVI   $8, V2, V2;      \
	VSRAVI   $4, V2, V2;      \
	VADDVV   V1, V2, V2;      \
	VSETVLI  X5, E8, MF2, TA, MA, X6; \
	VNSRLWI  $0, V2, V15;     \
	VSE8V    V15, (X10);      \
	VSETVLI  X5, E16, M1, TA, MA, X6; \
	ADD      X11, X10;        \
	ADDI     $24, X12;        \
	ADDI     $-1, X7

// func cdefFilterRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterRVV(SB), NOSPLIT, $0-32
	SETUP

loop:
	LOADPX

	MOVW 40(X13), X15
	TAP(ADD, X18, V12, X8, X15)
	TAP(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAP(ADD, X19, V12, X8, X15)
	TAP(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAP(ADD, X20, V13, X9, X15)
	TAP(SUB, X20, V13, X9, X15)
	TAP(ADD, X22, V13, X9, X15)
	TAP(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAP(ADD, X21, V13, X9, X15)
	TAP(SUB, X21, V13, X9, X15)
	TAP(ADD, X23, V13, X9, X15)
	TAP(SUB, X23, V13, X9, X15)

	FINISH
	BNEZ X7, loop

	RET

// func cdefFilterClipRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClipRVV(SB), NOSPLIT, $0-32
	SETUP

loopc:
	LOADPX
	VMVVV V1, V3
	VMVVV V1, V4

	MOVW 40(X13), X15
	TAPC(ADD, X18, V12, X8, X15)
	TAPC(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPC(ADD, X19, V12, X8, X15)
	TAPC(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPC(ADD, X20, V13, X9, X15)
	TAPC(SUB, X20, V13, X9, X15)
	TAPC(ADD, X22, V13, X9, X15)
	TAPC(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPC(ADD, X21, V13, X9, X15)
	TAPC(SUB, X21, V13, X9, X15)
	TAPC(ADD, X23, V13, X9, X15)
	TAPC(SUB, X23, V13, X9, X15)

	VSRAVI  $15, V2, V11
	VADDVV  V11, V2, V2
	VADDVI  $8, V2, V2
	VSRAVI  $4, V2, V2
	VADDVV  V1, V2, V2
	VMAXVV  V4, V2, V2
	VMINVV  V3, V2, V2
	VSETVLI X5, E8, MF2, TA, MA, X6
	VNSRLWI $0, V2, V15
	VSE8V   V15, (X10)
	VSETVLI X5, E16, M1, TA, MA, X6
	ADD     X11, X10
	ADDI    $24, X12
	ADDI    $-1, X7
	BNEZ    X7, loopc

	RET
