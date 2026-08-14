#include "textflag.h"

#define SXTL(n, d)   WORD $(0x0f10a400 | ((n) << 5) | (d))
#define SXTL2(n, d)  WORD $(0x4f10a400 | ((n) << 5) | (d))
#define SMLAL(m, n, d)  WORD $(0x0e608000 | ((m) << 16) | ((n) << 5) | (d))
#define SMLAL2(m, n, d) WORD $(0x4e608000 | ((m) << 16) | ((n) << 5) | (d))
#define ADDV4S(n, d) WORD $(0x4eb1b800 | ((n) << 5) | (d))

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
