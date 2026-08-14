//go:build arm64 && !noasm

#include "textflag.h"

#define SXTL(n, d)   WORD $(0x0f10a400 | ((n) << 5) | (d))
#define SXTL2(n, d)  WORD $(0x4f10a400 | ((n) << 5) | (d))
#define SMLAL(m, n, d)  WORD $(0x0e608000 | ((m) << 16) | ((n) << 5) | (d))
#define SMLAL2(m, n, d) WORD $(0x4e608000 | ((m) << 16) | ((n) << 5) | (d))
#define ADDV4S(n, d) WORD $(0x4eb1b800 | ((n) << 5) | (d))
#define ABS4S(n, d)  WORD $(0x4ea0b800 | ((n) << 5) | (d))
#define MUL4S(m, n, d)  WORD $(0x4ea09c00 | ((m) << 16) | ((n) << 5) | (d))
#define SSHR13(n, d)    WORD $(0x4f330400 | ((n) << 5) | (d))
#define SSHRN(k, n, d)  WORD $(0x4f000400 | ((64 - (k)) << 16) | ((n) << 5) | (d))

#define RSN(v, n) \
	VADD V19.S4, v.S4, v.S4; \
	SSHR13(n, n)

#define FDCT4N \
	VADD V3.S4, V0.S4, V20.S4; \
	VADD V2.S4, V1.S4, V21.S4; \
	VSUB V2.S4, V1.S4, V22.S4; \
	VSUB V3.S4, V0.S4, V23.S4; \
	VADD V21.S4, V20.S4, V24.S4; \
	VSUB V21.S4, V20.S4, V25.S4; \
	MUL4S(16, 24, 26); \
	RSN(V26, 26); \
	MUL4S(16, 25, 27); \
	RSN(V27, 27); \
	MUL4S(17, 22, 28); \
	MUL4S(18, 23, 29); \
	VADD V29.S4, V28.S4, V28.S4; \
	RSN(V28, 28); \
	MUL4S(17, 23, 30); \
	MUL4S(18, 22, 31); \
	VSUB V31.S4, V30.S4, V30.S4; \
	RSN(V30, 30); \
	VMOV V26.B16, V0.B16; \
	VMOV V28.B16, V1.B16; \
	VMOV V27.B16, V2.B16; \
	VMOV V30.B16, V3.B16

#define FADST4N \
	MUL4S(16, 0, 20); \
	MUL4S(18, 1, 21); \
	VADD V21.S4, V20.S4, V20.S4; \
	MUL4S(15, 3, 21); \
	VADD V21.S4, V20.S4, V20.S4; \
	MUL4S(15, 0, 21); \
	MUL4S(16, 1, 22); \
	VSUB V22.S4, V21.S4, V21.S4; \
	MUL4S(18, 3, 22); \
	VADD V22.S4, V21.S4, V21.S4; \
	MUL4S(17, 2, 22); \
	VADD V1.S4, V0.S4, V23.S4; \
	VSUB V3.S4, V23.S4, V23.S4; \
	MUL4S(17, 23, 23); \
	VADD V22.S4, V20.S4, V24.S4; \
	RSN(V24, 24); \
	RSN(V23, 23); \
	VSUB V22.S4, V21.S4, V25.S4; \
	RSN(V25, 25); \
	VSUB V20.S4, V21.S4, V26.S4; \
	VADD V22.S4, V26.S4, V26.S4; \
	RSN(V26, 26); \
	VMOV V24.B16, V0.B16; \
	VMOV V23.B16, V1.B16; \
	VMOV V25.B16, V2.B16; \
	VMOV V26.B16, V3.B16

#define DCTC \
	MOVD $5793, R4; \
	VDUP R4, V16.S4; \
	MOVD $3135, R4; \
	VDUP R4, V17.S4; \
	MOVD $7568, R4; \
	VDUP R4, V18.S4

#define ADSTC \
	MOVD $2642, R4; \
	VDUP R4, V16.S4; \
	MOVD $6689, R4; \
	VDUP R4, V17.S4; \
	MOVD $4964, R4; \
	VDUP R4, V18.S4; \
	MOVD $7606, R4; \
	VDUP R4, V15.S4

#define HAD4N(i0, i1, i2, i3, o0, o1, o2, o3) \
	VADD i2.S4, i0.S4, V20.S4 \
	VADD i3.S4, i1.S4, V21.S4 \
	VSUB i2.S4, i0.S4, V22.S4 \
	VSUB i3.S4, i1.S4, V23.S4 \
	VADD V21.S4, V20.S4, o0.S4 \
	VSUB V21.S4, V20.S4, o1.S4 \
	VADD V23.S4, V22.S4, o2.S4 \
	VSUB V23.S4, V22.S4, o3.S4

#define T4N(i0, i1, i2, i3, o0, o1, o2, o3) \
	TT4(i0, i1, i2, i3, o0, o1, o2, o3, V20, V21, V22, V23)

#define TT4(i0, i1, i2, i3, o0, o1, o2, o3, t0, t1, t2, t3) \
	VZIP1 i1.S4, i0.S4, t0.S4 \
	VZIP2 i1.S4, i0.S4, t1.S4 \
	VZIP1 i3.S4, i2.S4, t2.S4 \
	VZIP2 i3.S4, i2.S4, t3.S4 \
	VZIP1 t2.D2, t0.D2, o0.D2 \
	VZIP2 t2.D2, t0.D2, o1.D2 \
	VZIP1 t3.D2, t1.D2, o2.D2 \
	VZIP2 t3.D2, t1.D2, o3.D2

#define HH8(i0, i1, i2, i3, i4, i5, i6, i7, t0, t1, t2, t3, t4, t5, t6, t7) \
	VADD i4.S4, i0.S4, t0.S4 \
	VADD i5.S4, i1.S4, t1.S4 \
	VADD i6.S4, i2.S4, t2.S4 \
	VADD i7.S4, i3.S4, t3.S4 \
	VSUB i4.S4, i0.S4, t4.S4 \
	VSUB i5.S4, i1.S4, t5.S4 \
	VSUB i6.S4, i2.S4, t6.S4 \
	VSUB i7.S4, i3.S4, t7.S4 \
	VADD t2.S4, t0.S4, i0.S4 \
	VADD t3.S4, t1.S4, i1.S4 \
	VSUB t2.S4, t0.S4, i2.S4 \
	VSUB t3.S4, t1.S4, i3.S4 \
	VADD t6.S4, t4.S4, i4.S4 \
	VADD t7.S4, t5.S4, i5.S4 \
	VSUB t6.S4, t4.S4, i6.S4 \
	VSUB t7.S4, t5.S4, i7.S4 \
	VADD i1.S4, i0.S4, t0.S4 \
	VSUB i1.S4, i0.S4, t1.S4 \
	VADD i3.S4, i2.S4, t2.S4 \
	VSUB i3.S4, i2.S4, t3.S4 \
	VADD i5.S4, i4.S4, t4.S4 \
	VSUB i5.S4, i4.S4, t5.S4 \
	VADD i7.S4, i6.S4, t6.S4 \
	VSUB i7.S4, i6.S4, t7.S4 \
	VMOV t0.B16, i0.B16 \
	VMOV t1.B16, i1.B16 \
	VMOV t2.B16, i2.B16 \
	VMOV t3.B16, i3.B16 \
	VMOV t4.B16, i4.B16 \
	VMOV t5.B16, i5.B16 \
	VMOV t6.B16, i6.B16 \
	VMOV t7.B16, i7.B16

// residualNEON subtracts the prediction from the source, stores the difference
// and returns its sum of squares. Eight pixels a step: two rows at 4 px wide,
// half a row otherwise.
TEXT ·residualNEON(SB), NOSPLIT, $0-64
	MOVD res+0(FP), R0
	MOVD dst+8(FP), R1
	MOVD stride+16(FP), R2
	MOVD src+24(FP), R3
	MOVD sstride+32(FP), R4
	MOVD w+40(FP), R5
	MOVD h+48(FP), R6
	VEOR V7.B16, V7.B16, V7.B16

	CMP $4, R5
	BEQ narrow

wide:
	MOVD R5, R7
	MOVD R3, R8
	MOVD R1, R9

wcol:
	MOVD  (R8), R10
	VMOV  R10, V0.D[0]
	MOVD  (R9), R11
	VMOV  R11, V1.D[0]
	VUSHLL $0, V0.B8, V2.H8
	VUSHLL $0, V1.B8, V3.H8
	VSUB  V3.H8, V2.H8, V4.H8
	SXTL(4, 5)
	SXTL2(4, 6)
	VST1  [V5.S4, V6.S4], (R0)
	ADD   $32, R0
	SMLAL(4, 4, 7)
	SMLAL2(4, 4, 7)
	ADD   $8, R8
	ADD   $8, R9
	SUB   $8, R7
	CBNZ  R7, wcol

	ADD  R4, R3
	ADD  R2, R1
	SUB  $1, R6
	CBNZ R6, wide
	B    done

narrow:
	MOVWU (R3), R10
	MOVWU (R3)(R4), R11
	ORR   R11<<32, R10, R10
	VMOV  R10, V0.D[0]
	MOVWU (R1), R12
	MOVWU (R1)(R2), R13
	ORR   R13<<32, R12, R12
	VMOV  R12, V1.D[0]
	VUSHLL $0, V0.B8, V2.H8
	VUSHLL $0, V1.B8, V3.H8
	VSUB  V3.H8, V2.H8, V4.H8
	SXTL(4, 5)
	SXTL2(4, 6)
	VST1  [V5.S4, V6.S4], (R0)
	ADD   $32, R0
	SMLAL(4, 4, 7)
	SMLAL2(4, 4, 7)
	ADD   R4<<1, R3
	ADD   R2<<1, R1
	SUB   $2, R6
	CBNZ  R6, narrow

done:
	ADDV4S(7, 0)
	VMOV V0.S[0], R10
	MOVWU R10, R10
	MOVD R10, ret+56(FP)
	RET

// sseNEON is the sum of squared differences, eight pixels a step.
TEXT ·sseNEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R1
	MOVD stride+8(FP), R2
	MOVD src+16(FP), R3
	MOVD sstride+24(FP), R4
	MOVD w+32(FP), R5
	MOVD h+40(FP), R6
	VEOR V7.B16, V7.B16, V7.B16

	CMP $4, R5
	BEQ snarrow

swide:
	MOVD R5, R7
	MOVD R3, R8
	MOVD R1, R9

swcol:
	MOVD   (R8), R10
	VMOV   R10, V0.D[0]
	MOVD   (R9), R11
	VMOV   R11, V1.D[0]
	VUSHLL $0, V0.B8, V2.H8
	VUSHLL $0, V1.B8, V3.H8
	VSUB   V3.H8, V2.H8, V4.H8
	SMLAL(4, 4, 7)
	SMLAL2(4, 4, 7)
	ADD    $8, R8
	ADD    $8, R9
	SUB    $8, R7
	CBNZ   R7, swcol

	ADD  R4, R3
	ADD  R2, R1
	SUB  $1, R6
	CBNZ R6, swide
	B    sdone

snarrow:
	MOVWU  (R3), R10
	MOVWU  (R3)(R4), R11
	ORR    R11<<32, R10, R10
	VMOV   R10, V0.D[0]
	MOVWU  (R1), R12
	MOVWU  (R1)(R2), R13
	ORR    R13<<32, R12, R12
	VMOV   R12, V1.D[0]
	VUSHLL $0, V0.B8, V2.H8
	VUSHLL $0, V1.B8, V3.H8
	VSUB   V3.H8, V2.H8, V4.H8
	SMLAL(4, 4, 7)
	SMLAL2(4, 4, 7)
	ADD    R4<<1, R3
	ADD    R2<<1, R1
	SUB    $2, R6
	CBNZ   R6, snarrow

sdone:
	ADDV4S(7, 0)
	VMOV  V0.S[0], R10
	MOVWU R10, R10
	MOVD  R10, ret+48(FP)
	RET

// satd4NEON is the sum of absolute 4x4 Hadamard coefficients.
TEXT ·satd4NEON(SB), NOSPLIT, $0-20
	MOVD res+0(FP), R0
	MOVD stride+8(FP), R1
	LSL  $2, R1, R1

	VLD1 (R0), [V0.S4]
	ADD  R1, R0, R2
	VLD1 (R2), [V1.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V2.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V3.S4]

	HAD4N(V0, V1, V2, V3, V4, V5, V6, V7)
	T4N(V4, V5, V6, V7, V0, V1, V2, V3)
	HAD4N(V0, V1, V2, V3, V4, V5, V6, V7)

	ABS4S(4, 4)
	ABS4S(5, 5)
	ABS4S(6, 6)
	ABS4S(7, 7)
	VADD V5.S4, V4.S4, V4.S4
	VADD V7.S4, V6.S4, V6.S4
	VADD V6.S4, V4.S4, V4.S4
	ADDV4S(4, 0)
	VMOV V0.S[0], R3
	ADDW $2, R3, R3
	ASRW $2, R3, R3
	MOVW R3, ret+16(FP)
	RET

// satd8NEON is the sum of absolute 8x8 Hadamard coefficients. A row is two
// registers, so both butterfly passes run down the columns and the transpose
// is four 4x4 blocks with the off-diagonal pair swapped. Every macro takes its
// temporaries, because the obvious fixed set collides with its own inputs.
TEXT ·satd8NEON(SB), NOSPLIT, $0-20
	MOVD res+0(FP), R0
	MOVD stride+8(FP), R1
	LSL  $2, R1, R1

	VLD1 (R0), [V0.S4, V1.S4]
	ADD  R1, R0, R2
	VLD1 (R2), [V2.S4, V3.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V4.S4, V5.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V6.S4, V7.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V8.S4, V9.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V10.S4, V11.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V12.S4, V13.S4]
	ADD  R1, R2, R2
	VLD1 (R2), [V14.S4, V15.S4]

	HH8(V0, V2, V4, V6, V8, V10, V12, V14, V16, V17, V18, V19, V20, V21, V22, V23)
	HH8(V1, V3, V5, V7, V9, V11, V13, V15, V16, V17, V18, V19, V20, V21, V22, V23)

	TT4(V0, V2, V4, V6, V16, V17, V18, V19, V24, V25, V26, V27)
	TT4(V8, V10, V12, V14, V20, V21, V22, V23, V24, V25, V26, V27)
	TT4(V1, V3, V5, V7, V24, V25, V26, V27, V0, V2, V4, V6)
	TT4(V9, V11, V13, V15, V28, V29, V30, V31, V0, V2, V4, V6)

	HH8(V16, V17, V18, V19, V24, V25, V26, V27, V0, V1, V2, V3, V4, V5, V6, V7)
	HH8(V20, V21, V22, V23, V28, V29, V30, V31, V8, V9, V10, V11, V12, V13, V14, V15)

	ABS4S(16, 16)
	ABS4S(17, 17)
	ABS4S(18, 18)
	ABS4S(19, 19)
	ABS4S(20, 20)
	ABS4S(21, 21)
	ABS4S(22, 22)
	ABS4S(23, 23)
	ABS4S(24, 24)
	ABS4S(25, 25)
	ABS4S(26, 26)
	ABS4S(27, 27)
	ABS4S(28, 28)
	ABS4S(29, 29)
	ABS4S(30, 30)
	ABS4S(31, 31)
	VADD V17.S4, V16.S4, V16.S4
	VADD V19.S4, V18.S4, V18.S4
	VADD V21.S4, V20.S4, V20.S4
	VADD V23.S4, V22.S4, V22.S4
	VADD V25.S4, V24.S4, V24.S4
	VADD V27.S4, V26.S4, V26.S4
	VADD V29.S4, V28.S4, V28.S4
	VADD V31.S4, V30.S4, V30.S4
	VADD V18.S4, V16.S4, V16.S4
	VADD V22.S4, V20.S4, V20.S4
	VADD V26.S4, V24.S4, V24.S4
	VADD V30.S4, V28.S4, V28.S4
	VADD V20.S4, V16.S4, V16.S4
	VADD V28.S4, V24.S4, V24.S4
	VADD V24.S4, V16.S4, V16.S4
	ADDV4S(16, 0)
	VMOV V0.S[0], R3
	ADDW $4, R3, R3
	ASRW $3, R3, R3
	MOVW R3, ret+16(FP)
	RET

// fwdTxfm4x4NEON is the 4x4 forward transform for the four non-flipping
// DCT/ADST combinations. cos_bit is 13 at this size, so the constants live in
// V15 to V18 and the rounding term in V19.
TEXT ·fwdTxfm4x4NEON(SB), NOSPLIT, $0-40
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2

	VLD1 (R1), [V0.S4]
	ADD  R2, R1, R3
	VLD1 (R3), [V1.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V2.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V3.S4]
	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4

	MOVD $4096, R4
	VDUP R4, V19.S4

	MOVD colAdst+24(FP), R5
	CBNZ R5, ncoladst
	DCTC
	FDCT4N
	B    ncolfin

ncoladst:
	ADSTC
	FADST4N

ncolfin:
	TT4(V0, V1, V2, V3, V4, V5, V6, V7, V20, V21, V22, V23)
	VMOV V4.B16, V0.B16
	VMOV V5.B16, V1.B16
	VMOV V6.B16, V2.B16
	VMOV V7.B16, V3.B16

	MOVD rowAdst+32(FP), R5
	CBNZ R5, nrowadst
	DCTC
	FDCT4N
	B    nstore

nrowadst:
	ADSTC
	FADST4N

nstore:
	VST1 [V0.S4, V1.S4, V2.S4, V3.S4], (R0)
	RET

// The eight point transform needs seven cosines and the rounding term. V16..V19
// keep the three the four point transform already uses so DCTC still applies;
// V12..V15 hold the rest.
#define DCT8C \
	DCTC; \
	MOVD $4096, R4; \
	VDUP R4, V19.S4; \
	MOVD $1598, R4; \
	VDUP R4, V12.S4; \
	MOVD $6811, R4; \
	VDUP R4, V13.S4; \
	MOVD $4551, R4; \
	VDUP R4, V14.S4; \
	MOVD $8035, R4; \
	VDUP R4, V15.S4

#define BTF8N(ka, an, kb, bn, tn, tv, dn, dv) \
	MUL4S(ka, an, dn); \
	MUL4S(kb, bn, tn); \
	VADD tv.S4, dv.S4, dv.S4; \
	VADD V19.S4, dv.S4, dv.S4; \
	SSHR13(dn, dn)

#define BTFN8N(ka, an, kb, bn, tn, tv, dn, dv) \
	MUL4S(ka, an, dn); \
	MUL4S(kb, bn, tn); \
	VSUB tv.S4, dv.S4, dv.S4; \
	VADD V19.S4, dv.S4, dv.S4; \
	SSHR13(dn, dn)

// FDCT8N is an eight point DCT down V0..V7, four lanes wide, with V8..V11 and
// V20..V31 as temporaries. The final permutation is folded into the destination
// registers, so V0..V7 come out in frequency order.
#define FDCT8N \
	VADD V7.S4, V0.S4, V20.S4; \
	VADD V6.S4, V1.S4, V21.S4; \
	VADD V5.S4, V2.S4, V22.S4; \
	VADD V4.S4, V3.S4, V23.S4; \
	VSUB V4.S4, V3.S4, V24.S4; \
	VSUB V5.S4, V2.S4, V25.S4; \
	VSUB V6.S4, V1.S4, V26.S4; \
	VSUB V7.S4, V0.S4, V27.S4; \
	VADD V23.S4, V20.S4, V28.S4; \
	VADD V22.S4, V21.S4, V29.S4; \
	VSUB V22.S4, V21.S4, V30.S4; \
	VSUB V23.S4, V20.S4, V31.S4; \
	BTFN8N(16, 26, 16, 25, 11, V11, 8, V8); \
	BTF8N(16, 26, 16, 25, 10, V10, 9, V9); \
	BTF8N(16, 28, 16, 29, 11, V11, 0, V0); \
	BTFN8N(16, 28, 16, 29, 11, V11, 4, V4); \
	BTF8N(17, 30, 18, 31, 11, V11, 2, V2); \
	BTFN8N(17, 31, 18, 30, 11, V11, 6, V6); \
	VADD V8.S4, V24.S4, V28.S4; \
	VSUB V8.S4, V24.S4, V29.S4; \
	VSUB V9.S4, V27.S4, V30.S4; \
	VADD V9.S4, V27.S4, V31.S4; \
	BTF8N(12, 28, 15, 31, 11, V11, 1, V1); \
	BTF8N(13, 29, 14, 30, 11, V11, 5, V5); \
	BTFN8N(13, 30, 14, 29, 11, V11, 3, V3); \
	BTFN8N(12, 31, 15, 28, 11, V11, 7, V7)

// RECT1N is the round shift by one the 4x8 and 8x4 shift schedules ask for,
// then the rectangular Sqrt2 correction. V13 holds one and V14 holds 2048,
// reloaded over the cosines the column pass no longer needs.
#define RECT1N(v, n) \
	VADD V13.S4, v.S4, v.S4; \
	SSHRN(1, n, n); \
	MUL4S(16, n, n); \
	VADD V14.S4, v.S4, v.S4; \
	SSHRN(12, n, n)

#define RECTC \
	MOVD $1, R4; \
	VDUP R4, V13.S4; \
	MOVD $2048, R4; \
	VDUP R4, V14.S4

// fwdTxfm4x8NEON is the 4x8 DCT_DCT forward transform: eight points down the
// columns over four lanes, then four points along each of the eight rows, run
// as two groups of four because a NEON register holds four.
TEXT ·fwdTxfm4x8NEON(SB), NOSPLIT, $0-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2

	VLD1 (R1), [V0.S4]
	ADD  R2, R1, R3
	VLD1 (R3), [V1.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V2.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V3.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V4.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V5.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V6.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V7.S4]

	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4

	DCT8C
	FDCT8N

	TT4(V0, V1, V2, V3, V8, V9, V10, V11, V20, V21, V22, V23)
	TT4(V4, V5, V6, V7, V0, V1, V2, V3, V20, V21, V22, V23)

	DCTC
	FDCT4N
	VMOV V0.B16, V4.B16
	VMOV V1.B16, V5.B16
	VMOV V2.B16, V6.B16
	VMOV V3.B16, V7.B16

	VMOV V8.B16, V0.B16
	VMOV V9.B16, V1.B16
	VMOV V10.B16, V2.B16
	VMOV V11.B16, V3.B16
	FDCT4N

	RECTC
	RECT1N(V0, 0)
	RECT1N(V1, 1)
	RECT1N(V2, 2)
	RECT1N(V3, 3)
	RECT1N(V4, 4)
	RECT1N(V5, 5)
	RECT1N(V6, 6)
	RECT1N(V7, 7)

	VST1 [V0.S4], (R0)
	ADD  $16, R0, R3
	VST1 [V4.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V1.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V5.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V2.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V6.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V3.S4], (R3)
	ADD  $16, R3, R3
	VST1 [V7.S4], (R3)
	RET

// fwdTxfm8x4NEON is the 8x4 DCT_DCT forward transform, the same two passes as
// fwdTxfm4x8NEON in the other order: four points down each half of the columns,
// then eight points along the rows.
TEXT ·fwdTxfm8x4NEON(SB), NOSPLIT, $0-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2

	VLD1 (R1), [V0.S4]
	ADD  $16, R1, R4
	VLD1 (R4), [V4.S4]
	ADD  R2, R1, R3
	VLD1 (R3), [V1.S4]
	ADD  $16, R3, R4
	VLD1 (R4), [V5.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V2.S4]
	ADD  $16, R3, R4
	VLD1 (R4), [V6.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V3.S4]
	ADD  $16, R3, R4
	VLD1 (R4), [V7.S4]

	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4

	DCT8C
	FDCT4N
	VMOV V0.B16, V8.B16
	VMOV V1.B16, V9.B16
	VMOV V2.B16, V10.B16
	VMOV V3.B16, V11.B16

	VMOV V4.B16, V0.B16
	VMOV V5.B16, V1.B16
	VMOV V6.B16, V2.B16
	VMOV V7.B16, V3.B16
	FDCT4N

	TT4(V0, V1, V2, V3, V4, V5, V6, V7, V20, V21, V22, V23)
	TT4(V8, V9, V10, V11, V0, V1, V2, V3, V20, V21, V22, V23)

	FDCT8N

	RECTC
	RECT1N(V0, 0)
	RECT1N(V1, 1)
	RECT1N(V2, 2)
	RECT1N(V3, 3)
	RECT1N(V4, 4)
	RECT1N(V5, 5)
	RECT1N(V6, 6)
	RECT1N(V7, 7)

	VST1 [V0.S4, V1.S4, V2.S4, V3.S4], (R0)
	ADD  $64, R0, R3
	VST1 [V4.S4, V5.S4, V6.S4, V7.S4], (R3)
	RET

#define DOWN1N(v, n) \
	VADD V20.S4, v.S4, v.S4; \
	SSHRN(1, n, n)

// fwdTxfm8x8NEON is the 8x8 DCT_DCT forward transform. A NEON register holds
// four lanes and the block is eight wide, so each pass runs twice over a
// scratch buffer rather than keeping sixteen rows live against FDCT8N's
// temporaries.
TEXT ·fwdTxfm8x8NEON(SB), NOSPLIT, $256-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2
	MOVD $buf-256(SP), R5

	MOVD R1, R3
	VLD1 (R3), [V0.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V1.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V2.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V3.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V4.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V5.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V6.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V7.S4]

	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4

	DCT8C
	FDCT8N

	MOVD R5, R6
	VST1 [V0.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V1.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V2.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V3.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V4.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V5.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V6.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V7.S4], (R6)

	ADD  $16, R1, R3
	VLD1 (R3), [V0.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V1.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V2.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V3.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V4.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V5.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V6.S4]
	ADD  R2, R3, R3
	VLD1 (R3), [V7.S4]

	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4

	FDCT8N

	ADD  $16, R5, R6
	VST1 [V0.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V1.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V2.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V3.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V4.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V5.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V6.S4], (R6)
	ADD  $32, R6, R6
	VST1 [V7.S4], (R6)

	MOVD R5, R6
	MOVD R0, R7
	MOVD $2, R8

rowpass8:
	VLD1 (R6), [V0.S4, V1.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V2.S4, V3.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V4.S4, V5.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V6.S4, V7.S4]
	ADD  $32, R6, R6

	TT4(V0, V2, V4, V6, V8, V9, V10, V11, V20, V21, V22, V23)
	TT4(V1, V3, V5, V7, V4, V5, V6, V7, V20, V21, V22, V23)
	VMOV V8.B16, V0.B16
	VMOV V9.B16, V1.B16
	VMOV V10.B16, V2.B16
	VMOV V11.B16, V3.B16

	FDCT8N

	MOVD $1, R4
	VDUP R4, V20.S4
	DOWN1N(V0, 0)
	DOWN1N(V1, 1)
	DOWN1N(V2, 2)
	DOWN1N(V3, 3)
	DOWN1N(V4, 4)
	DOWN1N(V5, 5)
	DOWN1N(V6, 6)
	DOWN1N(V7, 7)

	MOVD R7, R9
	VST1 [V0.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V7.S4], (R9)

	ADD  $16, R7, R7
	SUBS $1, R8, R8
	BNE  rowpass8
	RET

// The sixteen point transform takes its cosines as macro parameters: NEON has
// no room to keep fifteen of them in registers alongside sixteen live rows, so
// they are materialised per stage, and 16x16's row pass at cos_bit 12 is the
// same macro with a different table.
#define FDCT16N(sh, c4, c8, c12, c16, c20, c24, c28, c32, c36, c40, c44, c48, c52, c56, c60, rnd) \
	MOVD $c32, R4 \
	VDUP R4, V16.S4 \
	MOVD $c48, R4 \
	VDUP R4, V17.S4 \
	MOVD $c16, R4 \
	VDUP R4, V18.S4 \
	MOVD $rnd, R4 \
	VDUP R4, V19.S4 \
	VSUB V15.S4, V0.S4, V25.S4 \
	VADD V15.S4, V0.S4, V0.S4 \
	VMOV V25.B16, V15.B16 \
	VSUB V14.S4, V1.S4, V25.S4 \
	VADD V14.S4, V1.S4, V1.S4 \
	VMOV V25.B16, V14.B16 \
	VSUB V13.S4, V2.S4, V25.S4 \
	VADD V13.S4, V2.S4, V2.S4 \
	VMOV V25.B16, V13.B16 \
	VSUB V12.S4, V3.S4, V25.S4 \
	VADD V12.S4, V3.S4, V3.S4 \
	VMOV V25.B16, V12.B16 \
	VSUB V11.S4, V4.S4, V25.S4 \
	VADD V11.S4, V4.S4, V4.S4 \
	VMOV V25.B16, V11.B16 \
	VSUB V10.S4, V5.S4, V25.S4 \
	VADD V10.S4, V5.S4, V5.S4 \
	VMOV V25.B16, V10.B16 \
	VSUB V9.S4, V6.S4, V25.S4 \
	VADD V9.S4, V6.S4, V6.S4 \
	VMOV V25.B16, V9.B16 \
	VSUB V8.S4, V7.S4, V25.S4 \
	VADD V8.S4, V7.S4, V7.S4 \
	VMOV V25.B16, V8.B16 \
	VSUB V7.S4, V0.S4, V25.S4 \
	VADD V7.S4, V0.S4, V0.S4 \
	VMOV V25.B16, V7.B16 \
	VSUB V6.S4, V1.S4, V25.S4 \
	VADD V6.S4, V1.S4, V1.S4 \
	VMOV V25.B16, V6.B16 \
	VSUB V5.S4, V2.S4, V25.S4 \
	VADD V5.S4, V2.S4, V2.S4 \
	VMOV V25.B16, V5.B16 \
	VSUB V4.S4, V3.S4, V25.S4 \
	VADD V4.S4, V3.S4, V3.S4 \
	VMOV V25.B16, V4.B16 \
	VSUB V10.S4, V13.S4, V25.S4 \
	VADD V10.S4, V13.S4, V26.S4 \
	MUL4S(16, 25, 10) \
	VADD V19.S4, V10.S4, V10.S4 \
	SSHRN(sh, 10, 10) \
	MUL4S(16, 26, 13) \
	VADD V19.S4, V13.S4, V13.S4 \
	SSHRN(sh, 13, 13) \
	VSUB V11.S4, V12.S4, V25.S4 \
	VADD V11.S4, V12.S4, V26.S4 \
	MUL4S(16, 25, 11) \
	VADD V19.S4, V11.S4, V11.S4 \
	SSHRN(sh, 11, 11) \
	MUL4S(16, 26, 12) \
	VADD V19.S4, V12.S4, V12.S4 \
	SSHRN(sh, 12, 12) \
	VSUB V3.S4, V0.S4, V25.S4 \
	VADD V3.S4, V0.S4, V0.S4 \
	VMOV V25.B16, V3.B16 \
	VSUB V2.S4, V1.S4, V25.S4 \
	VADD V2.S4, V1.S4, V1.S4 \
	VMOV V25.B16, V2.B16 \
	VSUB V5.S4, V6.S4, V25.S4 \
	VADD V5.S4, V6.S4, V26.S4 \
	MUL4S(16, 25, 5) \
	VADD V19.S4, V5.S4, V5.S4 \
	SSHRN(sh, 5, 5) \
	MUL4S(16, 26, 6) \
	VADD V19.S4, V6.S4, V6.S4 \
	SSHRN(sh, 6, 6) \
	VSUB V11.S4, V8.S4, V25.S4 \
	VADD V11.S4, V8.S4, V8.S4 \
	VMOV V25.B16, V11.B16 \
	VSUB V10.S4, V9.S4, V25.S4 \
	VADD V10.S4, V9.S4, V9.S4 \
	VMOV V25.B16, V10.B16 \
	VSUB V12.S4, V15.S4, V25.S4 \
	VADD V12.S4, V15.S4, V15.S4 \
	VMOV V25.B16, V12.B16 \
	VSUB V13.S4, V14.S4, V25.S4 \
	VADD V13.S4, V14.S4, V14.S4 \
	VMOV V25.B16, V13.B16 \
	VSUB V1.S4, V0.S4, V25.S4 \
	VADD V1.S4, V0.S4, V26.S4 \
	MUL4S(16, 25, 1) \
	VADD V19.S4, V1.S4, V1.S4 \
	SSHRN(sh, 1, 1) \
	MUL4S(16, 26, 0) \
	VADD V19.S4, V0.S4, V0.S4 \
	SSHRN(sh, 0, 0) \
	MUL4S(17, 2, 25) \
	MUL4S(18, 3, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(17, 3, 26) \
	MUL4S(18, 2, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V2.B16 \
	VMOV V26.B16, V3.B16 \
	VSUB V5.S4, V4.S4, V25.S4 \
	VADD V5.S4, V4.S4, V4.S4 \
	VMOV V25.B16, V5.B16 \
	VSUB V6.S4, V7.S4, V25.S4 \
	VADD V6.S4, V7.S4, V7.S4 \
	VMOV V25.B16, V6.B16 \
	MUL4S(17, 14, 25) \
	MUL4S(18, 9, 27) \
	VSUB V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(18, 14, 26) \
	MUL4S(17, 9, 28) \
	VADD V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V9.B16 \
	VMOV V26.B16, V14.B16 \
	MUL4S(17, 13, 25) \
	MUL4S(18, 10, 27) \
	VSUB V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(17, 10, 26) \
	MUL4S(18, 13, 27) \
	VADD V27.S4, V26.S4, V26.S4 \
	VSUB V26.S4, V19.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V13.B16 \
	VMOV V26.B16, V10.B16 \
	MOVD $c56, R4 \
	VDUP R4, V20.S4 \
	MOVD $c8, R4 \
	VDUP R4, V21.S4 \
	MOVD $c24, R4 \
	VDUP R4, V22.S4 \
	MOVD $c40, R4 \
	VDUP R4, V23.S4 \
	MUL4S(20, 4, 25) \
	MUL4S(21, 7, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(20, 7, 26) \
	MUL4S(21, 4, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V4.B16 \
	VMOV V26.B16, V7.B16 \
	MUL4S(22, 5, 25) \
	MUL4S(23, 6, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(22, 6, 26) \
	MUL4S(23, 5, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V5.B16 \
	VMOV V26.B16, V6.B16 \
	VSUB V9.S4, V8.S4, V25.S4 \
	VADD V9.S4, V8.S4, V8.S4 \
	VMOV V25.B16, V9.B16 \
	VSUB V10.S4, V11.S4, V25.S4 \
	VADD V10.S4, V11.S4, V11.S4 \
	VMOV V25.B16, V10.B16 \
	VSUB V13.S4, V12.S4, V25.S4 \
	VADD V13.S4, V12.S4, V12.S4 \
	VMOV V25.B16, V13.B16 \
	VSUB V14.S4, V15.S4, V25.S4 \
	VADD V14.S4, V15.S4, V15.S4 \
	VMOV V25.B16, V14.B16 \
	MOVD $c60, R4 \
	VDUP R4, V16.S4 \
	MOVD $c4, R4 \
	VDUP R4, V17.S4 \
	MOVD $c28, R4 \
	VDUP R4, V18.S4 \
	MOVD $c36, R4 \
	VDUP R4, V20.S4 \
	MOVD $c44, R4 \
	VDUP R4, V21.S4 \
	MOVD $c20, R4 \
	VDUP R4, V22.S4 \
	MOVD $c12, R4 \
	VDUP R4, V23.S4 \
	MOVD $c52, R4 \
	VDUP R4, V24.S4 \
	MUL4S(16, 8, 25) \
	MUL4S(17, 15, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(16, 15, 26) \
	MUL4S(17, 8, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V8.B16 \
	VMOV V26.B16, V15.B16 \
	MUL4S(18, 9, 25) \
	MUL4S(20, 14, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(18, 14, 26) \
	MUL4S(20, 9, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V9.B16 \
	VMOV V26.B16, V14.B16 \
	MUL4S(21, 10, 25) \
	MUL4S(22, 13, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(21, 13, 26) \
	MUL4S(22, 10, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V10.B16 \
	VMOV V26.B16, V13.B16 \
	MUL4S(23, 11, 25) \
	MUL4S(24, 12, 27) \
	VADD V27.S4, V25.S4, V25.S4 \
	VADD V19.S4, V25.S4, V25.S4 \
	SSHRN(sh, 25, 25) \
	MUL4S(23, 12, 26) \
	MUL4S(24, 11, 28) \
	VSUB V28.S4, V26.S4, V26.S4 \
	VADD V19.S4, V26.S4, V26.S4 \
	SSHRN(sh, 26, 26) \
	VMOV V25.B16, V11.B16 \
	VMOV V26.B16, V12.B16

#define RECT2N(v, n) \
	VADD V13.S4, v.S4, v.S4; \
	SSHRN(2, n, n); \
	MUL4S(16, n, n); \
	VADD V14.S4, v.S4, v.S4; \
	SSHRN(12, n, n)

#define RECT2CN \
	MOVD $2, R4; \
	VDUP R4, V13.S4; \
	MOVD $2048, R4; \
	VDUP R4, V14.S4

// fwdTxfm8x16NEON is the 8x16 DCT_DCT forward transform: sixteen points down
// the columns, four of the eight at a time, then eight points along the rows,
// four of the sixteen at a time.
TEXT ·fwdTxfm8x16NEON(SB), NOSPLIT, $512-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2
	MOVD $buf-512(SP), R5

	MOVD R1, R3
	MOVD R5, R6
	MOVD $2, R8

colpass816:
	MOVD R3, R9
	VLD1 (R9), [V0.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V1.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V2.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V3.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V4.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V5.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V6.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V7.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V8.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V9.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V10.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V11.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V12.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V13.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V14.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V15.S4]
	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4
	VSHL $2, V8.S4, V8.S4
	VSHL $2, V9.S4, V9.S4
	VSHL $2, V10.S4, V10.S4
	VSHL $2, V11.S4, V11.S4
	VSHL $2, V12.S4, V12.S4
	VSHL $2, V13.S4, V13.S4
	VSHL $2, V14.S4, V14.S4
	VSHL $2, V15.S4, V15.S4
	FDCT16N(13, 8153, 8035, 7839, 7568, 7225, 6811, 6333, 5793, 5197, 4551, 3862, 3135, 2378, 1598, 803, 4096)
	MOVD R6, R9
	VST1 [V0.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V8.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V12.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V10.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V14.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V9.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V13.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V11.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V15.S4], (R9)
	ADD  $16, R3, R3
	ADD  $16, R6, R6
	SUBS $1, R8, R8
	BNE  colpass816

	MOVD R5, R6
	MOVD R0, R7
	MOVD $4, R8

rowpass816:
	VLD1 (R6), [V0.S4, V1.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V2.S4, V3.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V4.S4, V5.S4]
	ADD  $32, R6, R6
	VLD1 (R6), [V6.S4, V7.S4]
	ADD  $32, R6, R6
	TT4(V0, V2, V4, V6, V8, V9, V10, V11, V20, V21, V22, V23)
	TT4(V1, V3, V5, V7, V4, V5, V6, V7, V20, V21, V22, V23)
	VMOV V8.B16, V0.B16
	VMOV V9.B16, V1.B16
	VMOV V10.B16, V2.B16
	VMOV V11.B16, V3.B16
	DCT8C
	FDCT8N
	RECT2CN
	RECT2N(V0, 0)
	RECT2N(V1, 1)
	RECT2N(V2, 2)
	RECT2N(V3, 3)
	RECT2N(V4, 4)
	RECT2N(V5, 5)
	RECT2N(V6, 6)
	RECT2N(V7, 7)
	MOVD R7, R9
	VST1 [V0.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $16, R7, R7
	SUBS $1, R8, R8
	BNE  rowpass816
	RET

// RECT2W and DOWN2W are the 16 wide forms: the four lane versions keep their
// constants in V13 and V14, which are live rows once the transform is sixteen
// points, so these use V16..V18 and are reloaded after each butterfly.
#define RECT2W(v, n) \
	VADD V16.S4, v.S4, v.S4; \
	SSHRN(2, n, n); \
	MUL4S(17, n, n); \
	VADD V18.S4, v.S4, v.S4; \
	SSHRN(12, n, n)

#define RECT2CW \
	MOVD $2, R4; \
	VDUP R4, V16.S4; \
	MOVD $5793, R4; \
	VDUP R4, V17.S4; \
	MOVD $2048, R4; \
	VDUP R4, V18.S4

#define DOWN2W(v, n) \
	VADD V16.S4, v.S4, v.S4; \
	SSHRN(2, n, n)

// fwdTxfm16x8NEON is the 16x8 DCT_DCT forward transform: eight points down
// four columns at a time, then sixteen points along four rows at a time.
TEXT ·fwdTxfm16x8NEON(SB), NOSPLIT, $512-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2
	MOVD $buf-512(SP), R5

	MOVD R1, R3
	MOVD R5, R6
	MOVD $4, R8

colpass168:
	MOVD R3, R9
	VLD1 (R9), [V0.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V1.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V2.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V3.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V4.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V5.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V6.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V7.S4]
	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4
	DCT8C
	FDCT8N
	MOVD R6, R9
	VST1 [V0.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $16, R3, R3
	ADD  $16, R6, R6
	SUBS $1, R8, R8
	BNE  colpass168

	MOVD R5, R6
	MOVD R0, R7
	MOVD $2, R8

rowpass168:
	VLD1 (R6), [V0.S4, V1.S4, V2.S4, V3.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V4.S4, V5.S4, V6.S4, V7.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V8.S4, V9.S4, V10.S4, V11.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V12.S4, V13.S4, V14.S4, V15.S4]
	ADD  $64, R6, R6
	TT4(V0, V4, V8, V12, V16, V17, V18, V19, V28, V29, V30, V31)
	TT4(V1, V5, V9, V13, V20, V21, V22, V23, V28, V29, V30, V31)
	TT4(V2, V6, V10, V14, V24, V25, V26, V27, V28, V29, V30, V31)
	TT4(V3, V7, V11, V15, V28, V29, V30, V31, V0, V1, V2, V3)
	VMOV V16.B16, V0.B16
	VMOV V17.B16, V1.B16
	VMOV V18.B16, V2.B16
	VMOV V19.B16, V3.B16
	VMOV V20.B16, V4.B16
	VMOV V21.B16, V5.B16
	VMOV V22.B16, V6.B16
	VMOV V23.B16, V7.B16
	VMOV V24.B16, V8.B16
	VMOV V25.B16, V9.B16
	VMOV V26.B16, V10.B16
	VMOV V27.B16, V11.B16
	VMOV V28.B16, V12.B16
	VMOV V29.B16, V13.B16
	VMOV V30.B16, V14.B16
	VMOV V31.B16, V15.B16
	FDCT16N(13, 8153, 8035, 7839, 7568, 7225, 6811, 6333, 5793, 5197, 4551, 3862, 3135, 2378, 1598, 803, 4096)
	RECT2CW
	RECT2W(V0, 0)
	RECT2W(V1, 1)
	RECT2W(V2, 2)
	RECT2W(V3, 3)
	RECT2W(V4, 4)
	RECT2W(V5, 5)
	RECT2W(V6, 6)
	RECT2W(V7, 7)
	RECT2W(V8, 8)
	RECT2W(V9, 9)
	RECT2W(V10, 10)
	RECT2W(V11, 11)
	RECT2W(V12, 12)
	RECT2W(V13, 13)
	RECT2W(V14, 14)
	RECT2W(V15, 15)
	MOVD R7, R9
	VST1 [V0.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V8.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V12.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V10.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V14.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V9.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V13.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V11.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $32, R9, R9
	VST1 [V15.S4], (R9)
	ADD  $16, R7, R7
	SUBS $1, R8, R8
	BNE  rowpass168
	RET

// fwdTxfm16x16NEON is the 16x16 DCT_DCT forward transform. Both passes are
// sixteen points over four lanes, so both run four times; the row pass is
// cos_bit 12. The 1024 byte transpose buffer puts it over the nosplit limit.
TEXT ·fwdTxfm16x16NEON(SB), 0, $1024-24
	MOVD res+0(FP), R0
	MOVD src+8(FP), R1
	MOVD srcStride+16(FP), R2
	LSL  $2, R2, R2
	MOVD $buf-1024(SP), R5

	MOVD R1, R3
	MOVD R5, R6
	MOVD $4, R8

colpass1616:
	MOVD R3, R9
	VLD1 (R9), [V0.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V1.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V2.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V3.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V4.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V5.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V6.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V7.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V8.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V9.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V10.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V11.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V12.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V13.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V14.S4]
	ADD  R2, R9, R9
	VLD1 (R9), [V15.S4]
	VSHL $2, V0.S4, V0.S4
	VSHL $2, V1.S4, V1.S4
	VSHL $2, V2.S4, V2.S4
	VSHL $2, V3.S4, V3.S4
	VSHL $2, V4.S4, V4.S4
	VSHL $2, V5.S4, V5.S4
	VSHL $2, V6.S4, V6.S4
	VSHL $2, V7.S4, V7.S4
	VSHL $2, V8.S4, V8.S4
	VSHL $2, V9.S4, V9.S4
	VSHL $2, V10.S4, V10.S4
	VSHL $2, V11.S4, V11.S4
	VSHL $2, V12.S4, V12.S4
	VSHL $2, V13.S4, V13.S4
	VSHL $2, V14.S4, V14.S4
	VSHL $2, V15.S4, V15.S4
	FDCT16N(13, 8153, 8035, 7839, 7568, 7225, 6811, 6333, 5793, 5197, 4551, 3862, 3135, 2378, 1598, 803, 4096)
	MOVD R6, R9
	VST1 [V0.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V8.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V12.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V10.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V14.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V9.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V13.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V11.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V15.S4], (R9)
	ADD  $16, R3, R3
	ADD  $16, R6, R6
	SUBS $1, R8, R8
	BNE  colpass1616

	MOVD R5, R6
	MOVD R0, R7
	MOVD $4, R8

rowpass1616:
	VLD1 (R6), [V0.S4, V1.S4, V2.S4, V3.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V4.S4, V5.S4, V6.S4, V7.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V8.S4, V9.S4, V10.S4, V11.S4]
	ADD  $64, R6, R6
	VLD1 (R6), [V12.S4, V13.S4, V14.S4, V15.S4]
	ADD  $64, R6, R6
	TT4(V0, V4, V8, V12, V16, V17, V18, V19, V28, V29, V30, V31)
	TT4(V1, V5, V9, V13, V20, V21, V22, V23, V28, V29, V30, V31)
	TT4(V2, V6, V10, V14, V24, V25, V26, V27, V28, V29, V30, V31)
	TT4(V3, V7, V11, V15, V28, V29, V30, V31, V0, V1, V2, V3)
	VMOV V16.B16, V0.B16
	VMOV V17.B16, V1.B16
	VMOV V18.B16, V2.B16
	VMOV V19.B16, V3.B16
	VMOV V20.B16, V4.B16
	VMOV V21.B16, V5.B16
	VMOV V22.B16, V6.B16
	VMOV V23.B16, V7.B16
	VMOV V24.B16, V8.B16
	VMOV V25.B16, V9.B16
	VMOV V26.B16, V10.B16
	VMOV V27.B16, V11.B16
	VMOV V28.B16, V12.B16
	VMOV V29.B16, V13.B16
	VMOV V30.B16, V14.B16
	VMOV V31.B16, V15.B16
	FDCT16N(12, 4076, 4017, 3920, 3784, 3612, 3406, 3166, 2896, 2598, 2276, 1931, 1567, 1189, 799, 401, 2048)
	MOVD $2, R4
	VDUP R4, V16.S4
	DOWN2W(V0, 0)
	DOWN2W(V1, 1)
	DOWN2W(V2, 2)
	DOWN2W(V3, 3)
	DOWN2W(V4, 4)
	DOWN2W(V5, 5)
	DOWN2W(V6, 6)
	DOWN2W(V7, 7)
	DOWN2W(V8, 8)
	DOWN2W(V9, 9)
	DOWN2W(V10, 10)
	DOWN2W(V11, 11)
	DOWN2W(V12, 12)
	DOWN2W(V13, 13)
	DOWN2W(V14, 14)
	DOWN2W(V15, 15)
	MOVD R7, R9
	VST1 [V0.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V8.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V4.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V12.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V2.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V10.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V6.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V14.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V1.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V9.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V5.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V13.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V3.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V11.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V7.S4], (R9)
	ADD  $64, R9, R9
	VST1 [V15.S4], (R9)
	ADD  $16, R7, R7
	SUBS $1, R8, R8
	BNE  rowpass1616
	RET
