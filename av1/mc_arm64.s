//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no vector multiply and no widening multiply.
#define MUL(Vd, Vn, Vm)     WORD $(0x4E609C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define MLA(Vd, Vn, Vm)     WORD $(0x4E609400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMULL(Vd, Vn, Vm)   WORD $(0x0E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMULL2(Vd, Vn, Vm)  WORD $(0x4E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMLAL(Vd, Vn, Vm)   WORD $(0x0E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMLAL2(Vd, Vn, Vm)  WORD $(0x4E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SQRDMULH(Vd, Vn, Vm) WORD $(0x6E60B400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SQXTUN(Vd, Vn)      WORD $(0x2E212800 | ((Vn) << 5) | (Vd))

// TAPS8 loads sixteen pixels and filters the first eight. The widened halves
// concatenate, so VEXT slides the eight-tap window one pixel at a time.
#define TAPS8(SRC)                        \
	VLD1    (SRC), [V0.B16];          \
	VUSHLL  $0, V0.B8, V1.H8;         \
	VUSHLL2 $0, V0.B16, V2.H8;        \
	MUL(3, 1, 16);                    \
	VEXT    $2, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 17);                    \
	VEXT    $4, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 18);                    \
	VEXT    $6, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 19);                    \
	VEXT    $8, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 20);                    \
	VEXT    $10, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 21);                    \
	VEXT    $12, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 22);                    \
	VEXT    $14, V2.B16, V1.B16, V4.B16; \
	MLA(3, 4, 23)

// LOADTAPS duplicates the eight signed coefficients into halfword lanes.
#define LOADTAPS(REG, T0, T1, T2, T3, T4, T5, T6, T7) \
	MOVB 0(REG), R9;  \
	VDUP R9, T0;      \
	MOVB 1(REG), R9;  \
	VDUP R9, T1;      \
	MOVB 2(REG), R9;  \
	VDUP R9, T2;      \
	MOVB 3(REG), R9;  \
	VDUP R9, T3;      \
	MOVB 4(REG), R9;  \
	VDUP R9, T4;      \
	MOVB 5(REG), R9;  \
	VDUP R9, T5;      \
	MOVB 6(REG), R9;  \
	VDUP R9, T6;      \
	MOVB 7(REG), R9;  \
	VDUP R9, T7

// func put8tapHNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapHNEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD f+32(FP), R4
	MOVD w+40(FP), R5
	MOVD h+48(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)

	MOVD $2, R9
	VDUP R9, V24.H8
	MOVD $512, R9
	VDUP R9, V25.H8

hrow:
	MOVD R2, R7
	MOVD R0, R8
	MOVD R5, R10

hcol:
	TAPS8(R7)
	VADD V24.H8, V3.H8, V3.H8
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	VST1 [V3.B8], (R8)

	ADD  $8, R7
	ADD  $8, R8
	SUB  $8, R10
	CBNZ R10, hcol

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, hrow

	RET

// VTAP accumulates one source row of the vertical filter.
#define VTAP(OP, TAPR)            \
	VLD1   (R9), [V0.B8];     \
	VUSHLL $0, V0.B8, V1.H8;  \
	OP(3, 1, TAPR);           \
	ADD    R3, R9

// func put8tapVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapVNEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD f+32(FP), R4
	MOVD w+40(FP), R5
	MOVD h+48(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)

	MOVD $512, R11
	VDUP R11, V25.H8

vrow:
	MOVD R2, R7
	MOVD R0, R8
	MOVD R5, R10

vcol:
	MOVD R7, R9
	VTAP(MUL, 16)
	VTAP(MLA, 17)
	VTAP(MLA, 18)
	VTAP(MLA, 19)
	VTAP(MLA, 20)
	VTAP(MLA, 21)
	VTAP(MLA, 22)
	VTAP(MLA, 23)
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	VST1 [V3.B8], (R8)

	ADD  $8, R7
	ADD  $8, R8
	SUB  $8, R10
	CBNZ R10, vcol

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, vrow

	RET

// HVTAP accumulates one scratch row. The scratch is too wide for a halfword
// product, so the taps widen into two word accumulators.
#define HVTAP(OP, OP2, TAPR)  \
	VLD1 (R12), [V0.H8];  \
	OP(5, 0, TAPR);       \
	OP2(6, 0, TAPR);      \
	ADD  $16, R12

// func put8tapHVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh, fv *[8]int8, w, h int)
TEXT ·put8tapHVNEON(SB), NOSPLIT, $0-72
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD mid+32(FP), R11
	MOVD fh+40(FP), R4
	MOVD w+56(FP), R5
	MOVD h+64(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)
	MOVD fv+48(FP), R4
	LOADTAPS(R4, V24.H8, V25.H8, V26.H8, V27.H8, V28.H8, V29.H8, V30.H8, V31.H8)

	MOVD $8192, R9
	VDUP R9, V15.H8

	// The word shift is logical, so a bias that is a multiple of the divisor
	// and large enough to clear the sign carries the rounding term with it.
	MOVD $524800, R9
	VDUP R9, V14.S4
	MOVD $512, R9
	VDUP R9, V13.S4

hvstrip:
	MOVD R2, R7
	MOVD R11, R9
	MOVD R6, R10
	ADD  $7, R10

hvh:
	TAPS8(R7)
	SQRDMULH(3, 3, 15)
	VST1 [V3.H8], (R9)

	ADD  R3, R7
	ADD  $16, R9
	SUB  $1, R10
	CBNZ R10, hvh

	MOVD R11, R9
	MOVD R0, R8
	MOVD R6, R10

hvv:
	MOVD R9, R12
	HVTAP(SMULL, SMULL2, 24)
	HVTAP(SMLAL, SMLAL2, 25)
	HVTAP(SMLAL, SMLAL2, 26)
	HVTAP(SMLAL, SMLAL2, 27)
	HVTAP(SMLAL, SMLAL2, 28)
	HVTAP(SMLAL, SMLAL2, 29)
	HVTAP(SMLAL, SMLAL2, 30)
	HVTAP(SMLAL, SMLAL2, 31)

	VADD  V14.S4, V5.S4, V5.S4
	VADD  V14.S4, V6.S4, V6.S4
	VUSHR $10, V5.S4, V5.S4
	VUSHR $10, V6.S4, V6.S4
	VSUB  V13.S4, V5.S4, V5.S4
	VSUB  V13.S4, V6.S4, V6.S4
	VUZP1 V6.H8, V5.H8, V5.H8
	SQXTUN(5, 5)
	VST1  [V5.B8], (R8)

	ADD  $16, R9
	ADD  R1, R8
	SUB  $1, R10
	CBNZ R10, hvv

	ADD  $8, R2
	ADD  $8, R0
	SUB  $8, R5
	CBNZ R5, hvstrip

	RET

// STN4 and STN2 write the low four or two bytes of a filtered row.
#define STN4(V) \
	VMOV V.S[0], R13; \
	MOVW R13, (R8)

#define STN2(V) \
	VMOV V.H[0], R13; \
	MOVH R13, (R8)

// func put8tapH4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapH4NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD f+32(FP), R4
	MOVD w+40(FP), R5
	MOVD h+48(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)

	MOVD $2, R9
	VDUP R9, V24.H8
	MOVD $512, R9
	VDUP R9, V25.H8

	CMP  $2, R5
	BEQ  h4row2

h4row4:
	MOVD R0, R8
	TAPS8(R2)
	VADD V24.H8, V3.H8, V3.H8
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	STN4(V3)

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, h4row4

	RET

h4row2:
	MOVD R0, R8
	TAPS8(R2)
	VADD V24.H8, V3.H8, V3.H8
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	STN2(V3)

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, h4row2

	RET

// func put8tapV4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapV4NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD f+32(FP), R4
	MOVD w+40(FP), R5
	MOVD h+48(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)

	MOVD $512, R11
	VDUP R11, V25.H8

	CMP  $2, R5
	BEQ  v4row2

v4row4:
	MOVD R2, R9
	MOVD R0, R8
	VTAP(MUL, 16)
	VTAP(MLA, 17)
	VTAP(MLA, 18)
	VTAP(MLA, 19)
	VTAP(MLA, 20)
	VTAP(MLA, 21)
	VTAP(MLA, 22)
	VTAP(MLA, 23)
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	STN4(V3)

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, v4row4

	RET

v4row2:
	MOVD R2, R9
	MOVD R0, R8
	VTAP(MUL, 16)
	VTAP(MLA, 17)
	VTAP(MLA, 18)
	VTAP(MLA, 19)
	VTAP(MLA, 20)
	VTAP(MLA, 21)
	VTAP(MLA, 22)
	VTAP(MLA, 23)
	SQRDMULH(3, 3, 25)
	SQXTUN(3, 3)
	STN2(V3)

	ADD  R3, R2
	ADD  R1, R0
	SUB  $1, R6
	CBNZ R6, v4row2

	RET

// func put8tapHV4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·put8tapHV4NEON(SB), NOSPLIT, $0-72
	MOVD dst+0(FP), R0
	MOVD dstStride+8(FP), R1
	MOVD src+16(FP), R2
	MOVD srcStride+24(FP), R3
	MOVD mid+32(FP), R11
	MOVD fh+40(FP), R4
	MOVD w+56(FP), R5
	MOVD h+64(FP), R6

	LOADTAPS(R4, V16.H8, V17.H8, V18.H8, V19.H8, V20.H8, V21.H8, V22.H8, V23.H8)
	MOVD fv+48(FP), R4
	LOADTAPS(R4, V24.H8, V25.H8, V26.H8, V27.H8, V28.H8, V29.H8, V30.H8, V31.H8)

	MOVD $8192, R9
	VDUP R9, V15.H8
	MOVD $524800, R9
	VDUP R9, V14.S4
	MOVD $512, R9
	VDUP R9, V13.S4

	MOVD R2, R7
	MOVD R11, R9
	MOVD R6, R10
	ADD  $7, R10

hv4h:
	TAPS8(R7)
	SQRDMULH(3, 3, 15)
	VST1 [V3.H8], (R9)

	ADD  R3, R7
	ADD  $16, R9
	SUB  $1, R10
	CBNZ R10, hv4h

	MOVD R11, R9
	MOVD R0, R8
	MOVD R6, R10

	CMP  $2, R5
	BEQ  hv4v2

hv4v4:
	MOVD R9, R12
	HVTAP(SMULL, SMULL2, 24)
	HVTAP(SMLAL, SMLAL2, 25)
	HVTAP(SMLAL, SMLAL2, 26)
	HVTAP(SMLAL, SMLAL2, 27)
	HVTAP(SMLAL, SMLAL2, 28)
	HVTAP(SMLAL, SMLAL2, 29)
	HVTAP(SMLAL, SMLAL2, 30)
	HVTAP(SMLAL, SMLAL2, 31)

	VADD  V14.S4, V5.S4, V5.S4
	VADD  V14.S4, V6.S4, V6.S4
	VUSHR $10, V5.S4, V5.S4
	VUSHR $10, V6.S4, V6.S4
	VSUB  V13.S4, V5.S4, V5.S4
	VSUB  V13.S4, V6.S4, V6.S4
	VUZP1 V6.H8, V5.H8, V5.H8
	SQXTUN(5, 5)
	STN4(V5)

	ADD  $16, R9
	ADD  R1, R8
	SUB  $1, R10
	CBNZ R10, hv4v4

	RET

hv4v2:
	MOVD R9, R12
	HVTAP(SMULL, SMULL2, 24)
	HVTAP(SMLAL, SMLAL2, 25)
	HVTAP(SMLAL, SMLAL2, 26)
	HVTAP(SMLAL, SMLAL2, 27)
	HVTAP(SMLAL, SMLAL2, 28)
	HVTAP(SMLAL, SMLAL2, 29)
	HVTAP(SMLAL, SMLAL2, 30)
	HVTAP(SMLAL, SMLAL2, 31)

	VADD  V14.S4, V5.S4, V5.S4
	VADD  V14.S4, V6.S4, V6.S4
	VUSHR $10, V5.S4, V5.S4
	VUSHR $10, V6.S4, V6.S4
	VSUB  V13.S4, V5.S4, V5.S4
	VSUB  V13.S4, V6.S4, V6.S4
	VUZP1 V6.H8, V5.H8, V5.H8
	SQXTUN(5, 5)
	STN2(V5)

	ADD  $16, R9
	ADD  R1, R8
	SUB  $1, R10
	CBNZ R10, hv4v2

	RET
