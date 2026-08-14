#include "textflag.h"

#define SXTL(n, d)   WORD $(0x0f10a400 | ((n) << 5) | (d))
#define SXTL2(n, d)  WORD $(0x4f10a400 | ((n) << 5) | (d))
#define SMLAL(m, n, d)  WORD $(0x0e608000 | ((m) << 16) | ((n) << 5) | (d))
#define SMLAL2(m, n, d) WORD $(0x4e608000 | ((m) << 16) | ((n) << 5) | (d))
#define ADDV4S(n, d) WORD $(0x4eb1b800 | ((n) << 5) | (d))
#define ABS4S(n, d)  WORD $(0x4ea0b800 | ((n) << 5) | (d))

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
