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
	VZIP1 i1.S4, i0.S4, V20.S4 \
	VZIP2 i1.S4, i0.S4, V21.S4 \
	VZIP1 i3.S4, i2.S4, V22.S4 \
	VZIP2 i3.S4, i2.S4, V23.S4 \
	VZIP1 V22.D2, V20.D2, o0.D2 \
	VZIP2 V22.D2, V20.D2, o1.D2 \
	VZIP1 V23.D2, V21.D2, o2.D2 \
	VZIP2 V23.D2, V21.D2, o3.D2

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
