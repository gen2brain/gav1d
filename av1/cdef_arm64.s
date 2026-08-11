//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of these.
#define USHL(Vd, Vn, Vm)  WORD $(0x6E604400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UQSUB(Vd, Vn, Vm) WORD $(0x6E602C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define MUL(Vd, Vn, Vm)   WORD $(0x4E609C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAX(Vd, Vn, Vm)  WORD $(0x4E606400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMIN(Vd, Vn, Vm)  WORD $(0x4E606C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

// TAP accumulates one neighbour. The out-of-frame fill saturates the threshold
// subtract to zero, so it contributes nothing without needing a mask. Absolute
// value and sign both come from the sign mask, so neither needs ABS or NEG.
#define LDROW(OP, OFFR)    \
	OP   OFFR, R2, R9; \
	VLD1 (R9), [V5.H8]

// LDPAIR takes four samples from each of two rows, so a block only four wide
// still fills all eight lanes.
#define LDPAIR(OP, OFFR)      \
	OP    OFFR, R2, R9;   \
	VLD1  (R9), [V5.D1];  \
	ADD   $24, R9, R9;    \
	VLD1  (R9), [V22.D1]; \
	VZIP1 V22.D2, V5.D2, V5.D2

#define TAP(LD, OP, OFFR, THR, SHR, TAPR) \
	LD(OP, OFFR);                 \
	VSUB   V1.H8, V5.H8, V6.H8;   \
	VUSHR  $15, V6.H8, V11.H8;    \
	VSUB   V11.H8, V31.H8, V11.H8; \
	VEOR   V11.B16, V6.B16, V7.B16; \
	VSUB   V11.H8, V7.H8, V7.H8;  \
	USHL(8, 7, SHR);              \
	UQSUB(9, THR, 8);             \
	VUMIN  V9.H8, V7.H8, V10.H8;  \
	VEOR   V11.B16, V10.B16, V10.B16; \
	VSUB   V11.H8, V10.H8, V10.H8; \
	MUL(10, 10, TAPR);            \
	VADD   V10.H8, V2.H8, V2.H8

#define TAPC(LD, OP, OFFR, THR, SHR, TAPR) \
	TAP(LD, OP, OFFR, THR, SHR, TAPR); \
	SMAX(3, 3, 5);                 \
	VUMIN V5.H8, V4.H8, V4.H8

#define SETUP                       \
	MOVD dst+0(FP), R0;         \
	MOVD dstStride+8(FP), R1;   \
	MOVD tmp+16(FP), R2;        \
	MOVD p+24(FP), R3;          \
	MOVW 0(R3), R4;             \
	MOVW 4(R3), R5;             \
	MOVW 8(R3), R6;             \
	MOVW 12(R3), R7;            \
	MOVW 16(R3), R8;            \
	MOVW 20(R3), R10;           \
	LSL  $1, R4, R4;            \
	LSL  $1, R5, R5;            \
	LSL  $1, R6, R6;            \
	LSL  $1, R7, R7;            \
	LSL  $1, R8, R8;            \
	LSL  $1, R10, R10;          \
	MOVW 24(R3), R9;            \
	VDUP R9, V12.H8;            \
	MOVW 28(R3), R9;            \
	VDUP R9, V13.H8;            \
	MOVW 32(R3), R9;            \
	NEG  R9, R9;                \
	VDUP R9, V14.H8;            \
	MOVW 36(R3), R9;            \
	NEG  R9, R9;                \
	VDUP R9, V15.H8;            \
	MOVW 56(R3), R11;           \
	MOVW 60(R3), R12;           \
	VEOR V31.B16, V31.B16, V31.B16; \
	MOVD $4104, R9;             \
	VDUP R9, V29.H8;            \
	MOVD $256, R9;              \
	VDUP R9, V30.H8;            \
	MOVW 40(R3), R9;            \
	VDUP R9, V18.H8;            \
	MOVW 44(R3), R9;            \
	VDUP R9, V19.H8;            \
	MOVW 48(R3), R9;            \
	VDUP R9, V20.H8;            \
	MOVW 52(R3), R9;            \
	VDUP R9, V21.H8

// LOADPX widens the row being filtered and zeroes the accumulator.
#define LOADPX                      \
	VLD1   (R0), [V0.B8];       \
	VUSHLL $0, V0.B8, V1.H8;    \
	VEOR   V2.B16, V2.B16, V2.B16

// LOADPX4 stacks the four samples of two rows into one vector.
#define LOADPX4                  \
	MOVWU  (R0), R9;         \
	ADD    R1, R0, R14;      \
	MOVWU  (R14), R13;       \
	ORR    R13<<32, R9, R9;  \
	VMOV   R9, V0.D[0];      \
	VUSHLL $0, V0.B8, V1.H8; \
	VEOR   V2.B16, V2.B16, V2.B16

#define STOREPX4         \
	VUZP1 V2.B16, V2.B16, V2.B16; \
	VMOV  V2.S[0], R9;            \
	MOVW  R9, (R0);               \
	VMOV  V2.S[1], R9;            \
	ADD   R1, R0, R14;            \
	MOVW  R9, (R14);              \
	ADD   R1, R0;                 \
	ADD   R1, R0;                 \
	ADD   $48, R2;                \
	SUB   $2, R12

// LOADPX416 is the same for halfword pixels, which need no widening.
#define LOADPX416                   \
	VLD1  (R0), [V1.D1];        \
	ADD   R1, R0, R14;          \
	VLD1  (R14), [V22.D1];      \
	VZIP1 V22.D2, V1.D2, V1.D2; \
	VEOR  V2.B16, V2.B16, V2.B16

#define STOREPX416       \
	VMOV V2.D[0], R9; \
	MOVD R9, (R0);    \
	VMOV V2.D[1], R9; \
	ADD  R1, R0, R14; \
	MOVD R9, (R14);   \
	ADD  R1, R0;      \
	ADD  R1, R0;      \
	ADD  $48, R2;     \
	SUB  $2, R12

// LOADPX16 needs no widening; the pixel is already a halfword.
#define LOADPX16                    \
	VLD1 (R0), [V1.H8];         \
	VEOR V2.B16, V2.B16, V2.B16

// SETUP16 adds the pixel maximum the byte store used to clamp to for free, and
// takes the stride to bytes.
#define SETUP16                     \
	SETUP;                      \
	LSL  $1, R1, R1;            \
	MOVW 64(R3), R9;            \
	VDUP R9, V17.H8

// ROUND is px + ((sum - (sum<0) + 8) >> 4). The bias makes the shift logical,
// which the assembler has, rather than arithmetic, which it does not.
#define ROUND                       \
	VUSHR $15, V2.H8, V11.H8;   \
	VSUB  V11.H8, V31.H8, V11.H8; \
	VADD  V11.H8, V2.H8, V2.H8; \
	VADD  V29.H8, V2.H8, V2.H8; \
	VUSHR $4, V2.H8, V2.H8;     \
	VSUB  V30.H8, V2.H8, V2.H8; \
	VADD  V1.H8, V2.H8, V2.H8

// func cdefFilterNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterNEON(SB), NOSPLIT, $0-32
	SETUP

loop:
	LOADPX

	TAP(LDROW, ADD, R4, 12, 14, 18)
	TAP(LDROW, SUB, R4, 12, 14, 18)
	TAP(LDROW, ADD, R5, 12, 14, 19)
	TAP(LDROW, SUB, R5, 12, 14, 19)

	TAP(LDROW, ADD, R6, 13, 15, 20)
	TAP(LDROW, SUB, R6, 13, 15, 20)
	TAP(LDROW, ADD, R8, 13, 15, 20)
	TAP(LDROW, SUB, R8, 13, 15, 20)
	TAP(LDROW, ADD, R7, 13, 15, 21)
	TAP(LDROW, SUB, R7, 13, 15, 21)
	TAP(LDROW, ADD, R10, 13, 15, 21)
	TAP(LDROW, SUB, R10, 13, 15, 21)

	ROUND
	VUZP1 V2.B16, V2.B16, V2.B16
	CMP   $8, R11
	BNE   store4
	VST1  [V2.B8], (R0)
	B     stored

store4:
	VMOV V2.S[0], R9
	MOVW R9, (R0)

stored:
	ADD  R1, R0
	ADD  $24, R2
	SUB  $1, R12
	CBNZ R12, loop

	RET

// func cdefFilterClipNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClipNEON(SB), NOSPLIT, $0-32
	SETUP

loopc:
	LOADPX
	VMOV V1.B16, V3.B16
	VMOV V1.B16, V4.B16

	TAPC(LDROW, ADD, R4, 12, 14, 18)
	TAPC(LDROW, SUB, R4, 12, 14, 18)
	TAPC(LDROW, ADD, R5, 12, 14, 19)
	TAPC(LDROW, SUB, R5, 12, 14, 19)

	TAPC(LDROW, ADD, R6, 13, 15, 20)
	TAPC(LDROW, SUB, R6, 13, 15, 20)
	TAPC(LDROW, ADD, R8, 13, 15, 20)
	TAPC(LDROW, SUB, R8, 13, 15, 20)
	TAPC(LDROW, ADD, R7, 13, 15, 21)
	TAPC(LDROW, SUB, R7, 13, 15, 21)
	TAPC(LDROW, ADD, R10, 13, 15, 21)
	TAPC(LDROW, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 4)
	SMIN(2, 2, 3)
	VUZP1 V2.B16, V2.B16, V2.B16
	CMP   $8, R11
	BNE   store4c
	VST1  [V2.B8], (R0)
	B     storedc

store4c:
	VMOV V2.S[0], R9
	MOVW R9, (R0)

storedc:
	ADD  R1, R0
	ADD  $24, R2
	SUB  $1, R12
	CBNZ R12, loopc

	RET

// func cdefFilter4NEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter4NEON(SB), NOSPLIT, $0-32
	SETUP

loop4:
	LOADPX4

	TAP(LDPAIR, ADD, R4, 12, 14, 18)
	TAP(LDPAIR, SUB, R4, 12, 14, 18)
	TAP(LDPAIR, ADD, R5, 12, 14, 19)
	TAP(LDPAIR, SUB, R5, 12, 14, 19)

	TAP(LDPAIR, ADD, R6, 13, 15, 20)
	TAP(LDPAIR, SUB, R6, 13, 15, 20)
	TAP(LDPAIR, ADD, R8, 13, 15, 20)
	TAP(LDPAIR, SUB, R8, 13, 15, 20)
	TAP(LDPAIR, ADD, R7, 13, 15, 21)
	TAP(LDPAIR, SUB, R7, 13, 15, 21)
	TAP(LDPAIR, ADD, R10, 13, 15, 21)
	TAP(LDPAIR, SUB, R10, 13, 15, 21)

	ROUND
	STOREPX4
	CBNZ R12, loop4

	RET

// func cdefFilterClip4NEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip4NEON(SB), NOSPLIT, $0-32
	SETUP

loop4c:
	LOADPX4
	VMOV V1.B16, V3.B16
	VMOV V1.B16, V4.B16

	TAPC(LDPAIR, ADD, R4, 12, 14, 18)
	TAPC(LDPAIR, SUB, R4, 12, 14, 18)
	TAPC(LDPAIR, ADD, R5, 12, 14, 19)
	TAPC(LDPAIR, SUB, R5, 12, 14, 19)

	TAPC(LDPAIR, ADD, R6, 13, 15, 20)
	TAPC(LDPAIR, SUB, R6, 13, 15, 20)
	TAPC(LDPAIR, ADD, R8, 13, 15, 20)
	TAPC(LDPAIR, SUB, R8, 13, 15, 20)
	TAPC(LDPAIR, ADD, R7, 13, 15, 21)
	TAPC(LDPAIR, SUB, R7, 13, 15, 21)
	TAPC(LDPAIR, ADD, R10, 13, 15, 21)
	TAPC(LDPAIR, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 4)
	SMIN(2, 2, 3)
	STOREPX4
	CBNZ R12, loop4c

	RET

// func cdefFilter16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter16NEON(SB), NOSPLIT, $0-32
	SETUP16

loop16:
	LOADPX16

	TAP(LDROW, ADD, R4, 12, 14, 18)
	TAP(LDROW, SUB, R4, 12, 14, 18)
	TAP(LDROW, ADD, R5, 12, 14, 19)
	TAP(LDROW, SUB, R5, 12, 14, 19)

	TAP(LDROW, ADD, R6, 13, 15, 20)
	TAP(LDROW, SUB, R6, 13, 15, 20)
	TAP(LDROW, ADD, R8, 13, 15, 20)
	TAP(LDROW, SUB, R8, 13, 15, 20)
	TAP(LDROW, ADD, R7, 13, 15, 21)
	TAP(LDROW, SUB, R7, 13, 15, 21)
	TAP(LDROW, ADD, R10, 13, 15, 21)
	TAP(LDROW, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 31)
	SMIN(2, 2, 17)
	CMP  $8, R11
	BNE  store416
	VST1 [V2.H8], (R0)
	B    stored16

store416:
	VMOV V2.D[0], R9
	MOVD R9, (R0)

stored16:
	ADD  R1, R0
	ADD  $24, R2
	SUB  $1, R12
	CBNZ R12, loop16

	RET

// func cdefFilterClip16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip16NEON(SB), NOSPLIT, $0-32
	SETUP16

loopc16:
	LOADPX16
	VMOV V1.B16, V3.B16
	VMOV V1.B16, V4.B16

	TAPC(LDROW, ADD, R4, 12, 14, 18)
	TAPC(LDROW, SUB, R4, 12, 14, 18)
	TAPC(LDROW, ADD, R5, 12, 14, 19)
	TAPC(LDROW, SUB, R5, 12, 14, 19)

	TAPC(LDROW, ADD, R6, 13, 15, 20)
	TAPC(LDROW, SUB, R6, 13, 15, 20)
	TAPC(LDROW, ADD, R8, 13, 15, 20)
	TAPC(LDROW, SUB, R8, 13, 15, 20)
	TAPC(LDROW, ADD, R7, 13, 15, 21)
	TAPC(LDROW, SUB, R7, 13, 15, 21)
	TAPC(LDROW, ADD, R10, 13, 15, 21)
	TAPC(LDROW, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 4)
	SMIN(2, 2, 3)
	CMP  $8, R11
	BNE  store4c16
	VST1 [V2.H8], (R0)
	B    storedc16

store4c16:
	VMOV V2.D[0], R9
	MOVD R9, (R0)

storedc16:
	ADD  R1, R0
	ADD  $24, R2
	SUB  $1, R12
	CBNZ R12, loopc16

	RET

// func cdefFilter416NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter416NEON(SB), NOSPLIT, $0-32
	SETUP16

loop416:
	LOADPX416

	TAP(LDPAIR, ADD, R4, 12, 14, 18)
	TAP(LDPAIR, SUB, R4, 12, 14, 18)
	TAP(LDPAIR, ADD, R5, 12, 14, 19)
	TAP(LDPAIR, SUB, R5, 12, 14, 19)

	TAP(LDPAIR, ADD, R6, 13, 15, 20)
	TAP(LDPAIR, SUB, R6, 13, 15, 20)
	TAP(LDPAIR, ADD, R8, 13, 15, 20)
	TAP(LDPAIR, SUB, R8, 13, 15, 20)
	TAP(LDPAIR, ADD, R7, 13, 15, 21)
	TAP(LDPAIR, SUB, R7, 13, 15, 21)
	TAP(LDPAIR, ADD, R10, 13, 15, 21)
	TAP(LDPAIR, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 31)
	SMIN(2, 2, 17)
	STOREPX416
	CBNZ R12, loop416

	RET

// func cdefFilterClip416NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip416NEON(SB), NOSPLIT, $0-32
	SETUP16

loopc416:
	LOADPX416
	VMOV V1.B16, V3.B16
	VMOV V1.B16, V4.B16

	TAPC(LDPAIR, ADD, R4, 12, 14, 18)
	TAPC(LDPAIR, SUB, R4, 12, 14, 18)
	TAPC(LDPAIR, ADD, R5, 12, 14, 19)
	TAPC(LDPAIR, SUB, R5, 12, 14, 19)

	TAPC(LDPAIR, ADD, R6, 13, 15, 20)
	TAPC(LDPAIR, SUB, R6, 13, 15, 20)
	TAPC(LDPAIR, ADD, R8, 13, 15, 20)
	TAPC(LDPAIR, SUB, R8, 13, 15, 20)
	TAPC(LDPAIR, ADD, R7, 13, 15, 21)
	TAPC(LDPAIR, SUB, R7, 13, 15, 21)
	TAPC(LDPAIR, ADD, R10, 13, 15, 21)
	TAPC(LDPAIR, SUB, R10, 13, 15, 21)

	ROUND
	SMAX(2, 2, 4)
	SMIN(2, 2, 3)
	STOREPX416
	CBNZ R12, loopc416

	RET
