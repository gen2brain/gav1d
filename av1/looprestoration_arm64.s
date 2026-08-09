//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of these.
#define SMULL(Vd, Vn, Vm)  WORD $(0x0E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMULL2(Vd, Vn, Vm) WORD $(0x4E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMLAL(Vd, Vn, Vm)  WORD $(0x0E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMLAL2(Vd, Vn, Vm) WORD $(0x4E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHL(Vd, Vn, Vm)   WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMIN(Vd, Vn, Vm)   WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SQXTUN(Vd, Vn)     WORD $(0x2E612800 | ((Vn) << 5) | (Vd))
#define SQXTUN2(Vd, Vn)    WORD $(0x6E612800 | ((Vn) << 5) | (Vd))
#define XTN(Vd, Vn)        WORD $(0x0E212800 | ((Vn) << 5) | (Vd))

// TAPS spreads the seven taps over one register each, which is what the
// non-element form of SMLAL wants.
#define TAPS                        \
	MOVD c+24(FP), R10;         \
	MOVH 0(R10), R9;            \
	VDUP R9, V16.H8;            \
	MOVH 2(R10), R9;            \
	VDUP R9, V17.H8;            \
	MOVH 4(R10), R9;            \
	VDUP R9, V18.H8;            \
	MOVH 6(R10), R9;            \
	VDUP R9, V19.H8;            \
	MOVH 8(R10), R9;            \
	VDUP R9, V20.H8;            \
	MOVH 10(R10), R9;           \
	VDUP R9, V21.H8;            \
	MOVH 12(R10), R9;           \
	VDUP R9, V22.H8

// ROUND holds the rounding term, the upper clip and the negated shift, since
// SSHL right-shifts on a negative count.
#define ROUND(RNDOFF, SHIFTOFF, LIMOFF) \
	MOVW RNDOFF(FP), R9;        \
	VDUP R9, V23.S4;            \
	MOVW LIMOFF(FP), R9;        \
	VDUP R9, V24.S4;            \
	MOVD SHIFTOFF(FP), R9;      \
	NEG  R9, R9;                \
	VDUP R9, V25.S4

// HTAP slides the window by one sample and folds in the next tap.
#define HTAP(IMM, TAPR)                  \
	VEXT $IMM, V2.B16, V1.B16, V3.B16; \
	SMLAL(26, 3, TAPR);              \
	SMLAL2(27, 3, TAPR)

// HBODY runs the seven taps over the window pair in V1:V2 and leaves eight
// clipped samples in V28.
#define HBODY                        \
	SMULL(26, 1, 16);            \
	SMULL2(27, 1, 16);           \
	HTAP(2, 17);                 \
	HTAP(4, 18);                 \
	HTAP(6, 19);                 \
	HTAP(8, 20);                 \
	HTAP(10, 21);                \
	HTAP(12, 22);                \
	VADD V23.S4, V26.S4, V26.S4; \
	VADD V23.S4, V27.S4, V27.S4; \
	SSHL(26, 26, 25);            \
	SSHL(27, 27, 25);            \
	SMIN(26, 26, 24);            \
	SMIN(27, 27, 24);            \
	SQXTUN(28, 26);              \
	SQXTUN2(28, 27)

// VBODY accumulates the seven rows already loaded into V0..V6.
#define VBODY                        \
	SMULL(26, 0, 16);            \
	SMULL2(27, 0, 16);           \
	SMLAL(26, 1, 17);            \
	SMLAL2(27, 1, 17);           \
	SMLAL(26, 2, 18);            \
	SMLAL2(27, 2, 18);           \
	SMLAL(26, 3, 19);            \
	SMLAL2(27, 3, 19);           \
	SMLAL(26, 4, 20);            \
	SMLAL2(27, 4, 20);           \
	SMLAL(26, 5, 21);            \
	SMLAL2(27, 5, 21);           \
	SMLAL(26, 6, 22);            \
	SMLAL2(27, 6, 22);           \
	VADD V23.S4, V26.S4, V26.S4; \
	VADD V23.S4, V27.S4, V27.S4; \
	SSHL(26, 26, 25);            \
	SSHL(27, 27, 25);            \
	SMIN(26, 26, 24);            \
	SMIN(27, 27, 24);            \
	SQXTUN(28, 26);              \
	SQXTUN2(28, 27)

#define VROWS                       \
	MOVD hor+8(FP), R9;         \
	MOVD rows+16(FP), R11;      \
	MOVD 0(R11), R1;            \
	ADD  R1<<1, R9, R1;         \
	MOVD 8(R11), R2;            \
	ADD  R2<<1, R9, R2;         \
	MOVD 16(R11), R3;           \
	ADD  R3<<1, R9, R3;         \
	MOVD 24(R11), R4;           \
	ADD  R4<<1, R9, R4;         \
	MOVD 32(R11), R5;           \
	ADD  R5<<1, R9, R5;         \
	MOVD 40(R11), R6;           \
	ADD  R6<<1, R9, R6;         \
	MOVD 48(R11), R7;           \
	ADD  R7<<1, R9, R7;         \
	MOVD n+32(FP), R8

#define VLOAD                       \
	VLD1.P 16(R1), [V0.H8];     \
	VLD1.P 16(R2), [V1.H8];     \
	VLD1.P 16(R3), [V2.H8];     \
	VLD1.P 16(R4), [V3.H8];     \
	VLD1.P 16(R5), [V4.H8];     \
	VLD1.P 16(R6), [V5.H8];     \
	VLD1.P 16(R7), [V6.H8]

// func wienerH8NEON(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH8NEON(SB), NOSPLIT, $0-52
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD n+16(FP), R8
	TAPS
	ROUND(rnd+32, shift+40, limit+48)

loop8:
	VLD1   (R1), [V0.B16]
	VUXTL  V0.B8, V1.H8
	VUXTL2 V0.B16, V2.H8
	HBODY
	VST1   [V28.H8], (R0)
	ADD    $16, R0
	ADD    $8, R1
	SUBS   $8, R8
	BNE    loop8

	RET

// func wienerH16NEON(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH16NEON(SB), NOSPLIT, $0-52
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD n+16(FP), R8
	TAPS
	ROUND(rnd+32, shift+40, limit+48)

loop16:
	VLD1 (R1), [V1.H8, V2.H8]
	HBODY
	VST1 [V28.H8], (R0)
	ADD  $16, R0
	ADD  $16, R1
	SUBS $8, R8
	BNE  loop16

	RET

// func wienerV8NEON(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV8NEON(SB), NOSPLIT, $0-60
	MOVD p+0(FP), R0
	VROWS
	TAPS
	ROUND(rnd+40, shift+48, limit+56)

vloop8:
	VLOAD
	VBODY
	XTN(28, 28)
	VST1.P [V28.B8], 8(R0)
	SUBS   $8, R8
	BNE    vloop8

	RET

// func wienerV16NEON(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV16NEON(SB), NOSPLIT, $0-60
	MOVD p+0(FP), R0
	VROWS
	TAPS
	ROUND(rnd+40, shift+48, limit+56)

vloop16:
	VLOAD
	VBODY
	VST1.P [V28.H8], 16(R0)
	SUBS   $8, R8
	BNE    vloop16

	RET
