//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of these.
#define MULS(Vd, Vn, Vm)   WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLL(Vd, Vn)      WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHR9(Vd, Vn)      WORD $(0x4F000400 | ((64 - 9) << 16) | ((Vn) << 5) | (Vd))
#define SSHR8(Vd, Vn)      WORD $(0x4F000400 | ((64 - 8) << 16) | ((Vn) << 5) | (Vd))
#define XTNS(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))

// The a rows are int32 and the b rows int16, three taps apart in each. A pair
// of registers holds enough of a row that the taps come from VEXT rather than
// three loads.

// AROW loads one int32 row into V0:V1 and leaves its three taps in V0, V6, V7.
#define AROW(REG)                    \
	VLD1 (REG), [V0.S4, V1.S4];  \
	VEXT $4, V1.B16, V0.B16, V6.B16; \
	VEXT $8, V1.B16, V0.B16, V7.B16

// BROW loads one int16 row and widens its three taps into V20, V21, V22.
#define BROW(REG)                \
	VLD1 (REG), [V10.H8];    \
	SSHLL(20, 10);           \
	VEXT $2, V10.B16, V10.B16, V21.B16; \
	SSHLL(21, 21);           \
	VEXT $4, V10.B16, V10.B16, V22.B16; \
	SSHLL(22, 22)

// PIXEL widens four source samples into V30.
#define PIXEL                       \
	MOVWU  (R1), R11;           \
	VMOV   R11, V29.S[0];       \
	VUSHLL $0, V29.B8, V29.H8;  \
	VUSHLL $0, V29.H4, V30.S4

// SCALE43 turns the four and three weighted sums in ACC and T into ACC.
#define SCALE43(ACC, T)              \
	VSHL $2, ACC.S4, ACC.S4;     \
	VSHL $1, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

// SCALE65 does the same for the six and five weighted sums.
#define SCALE65(ACC, T)              \
	VSHL $2, ACC.S4, V19.S4;     \
	VSHL $1, ACC.S4, ACC.S4;     \
	VADD V19.S4, ACC.S4, ACC.S4; \
	VSHL $2, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

// STORE finishes one group of four: subtract, round, shift and narrow.
#define STORE(SHIFT)                 \
	MULS(31, 31, 30);            \
	VSUB V31.S4, V28.S4, V28.S4; \
	VADD V27.S4, V28.S4, V28.S4; \
	SHIFT(28, 28);               \
	XTNS(28, 28);                \
	VST1 [V28.D1], (R0);         \
	ADD  $8, R0, R0;             \
	ADD  $4, R1, R1;             \
	SUBS $4, R10, R10

// func sgrFinish1NEON(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)
TEXT ·sgrFinish1NEON(SB), NOSPLIT, $0-72
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a0+16(FP), R2
	MOVD a1+24(FP), R3
	MOVD a2+32(FP), R4
	MOVD b0+40(FP), R5
	MOVD b1+48(FP), R6
	MOVD b2+56(FP), R7
	MOVD n+64(FP), R10
	MOVD $256, R11
	VDUP R11, V27.S4

loop1:
	AROW(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW(R3)
	VADD V0.S4, V28.S4, V28.S4
	VADD V6.S4, V28.S4, V28.S4
	VADD V7.S4, V28.S4, V28.S4
	AROW(R4)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE43(V28, V18)

	BROW(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW(R6)
	VADD V20.S4, V31.S4, V31.S4
	VADD V21.S4, V31.S4, V31.S4
	VADD V22.S4, V31.S4, V31.S4
	BROW(R7)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE43(V31, V26)

	PIXEL
	STORE(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $16, R4, R4
	ADD $8, R5, R5
	ADD $8, R6, R6
	ADD $8, R7, R7
	BNE loop1

	RET

// func sgrFinish2RowNEON(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)
TEXT ·sgrFinish2RowNEON(SB), NOSPLIT, $0-56
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a0+16(FP), R2
	MOVD a1+24(FP), R3
	MOVD b0+32(FP), R5
	MOVD b1+40(FP), R6
	MOVD n+48(FP), R10
	MOVD $256, R11
	VDUP R11, V27.S4

loop2:
	AROW(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW(R3)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE65(V28, V18)

	BROW(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW(R6)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL
	STORE(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $8, R5, R5
	ADD $8, R6, R6
	BNE loop2

	RET

// func sgrFinish2MidNEON(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)
TEXT ·sgrFinish2MidNEON(SB), NOSPLIT, $0-40
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a1+16(FP), R3
	MOVD b1+24(FP), R6
	MOVD n+32(FP), R10
	MOVD $128, R11
	VDUP R11, V27.S4

loop3:
	AROW(R3)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	SCALE65(V28, V18)

	BROW(R6)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL
	STORE(SSHR8)

	ADD $16, R3, R3
	ADD $8, R6, R6
	BNE loop3

	RET
