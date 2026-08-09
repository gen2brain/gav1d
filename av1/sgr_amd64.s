//go:build amd64 && !noasm

#include "textflag.h"

// The a rows are int32 and the b rows int16, three taps apart in each, so the
// taps are fixed displacements off one pointer per row.

// SUM3 folds the three taps of one int32 row, scaled, into ACC.
#define LOADB(OFF, REG, VD) \
	VPMOVSXWD OFF(REG), VD

// TAIL turns the two accumulators into eight samples and stores them.
#define TAIL(SHIFT)           \
	VPMOVZXBD (SI), Y4;   \
	VPMULLD   Y4, Y3, Y3; \
	VPSUBD    Y3, Y0, Y0; \
	VPADDD    Y6, Y0, Y0; \
	VPSRAD    SHIFT, Y0, Y0; \
	VPACKSSDW Y0, Y0, Y0; \
	VPERMQ    $0x08, Y0, Y0; \
	VMOVDQU   X0, (DI)

// func sgrFinish1AVX2(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)
TEXT ·sgrFinish1AVX2(SB), NOSPLIT, $0-72
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a0+16(FP), R8
	MOVQ         a1+24(FP), R9
	MOVQ         a2+32(FP), R10
	MOVQ         b0+40(FP), R11
	MOVQ         b1+48(FP), R12
	MOVQ         b2+56(FP), R13
	MOVQ         n+64(FP), CX
	MOVL         $256, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop1:
	VMOVDQU 4(R9), Y0
	VPADDD  (R9), Y0, Y0
	VPADDD  8(R9), Y0, Y0
	VPADDD  4(R8), Y0, Y0
	VPADDD  4(R10), Y0, Y0
	VMOVDQU (R8), Y1
	VPADDD  (R10), Y1, Y1
	VPADDD  8(R8), Y1, Y1
	VPADDD  8(R10), Y1, Y1
	VPSLLD  $2, Y0, Y0
	VPSLLD  $1, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R12, Y3)
	LOADB(0, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(4, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(2, R11, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(2, R13, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(0, R11, Y5)
	LOADB(0, R13, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R13, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y3
	VPSLLD $1, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $32, R10
	ADDQ $16, R11
	ADDQ $16, R12
	ADDQ $16, R13
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop1

	VZEROUPPER
	RET

// func sgrFinish2RowAVX2(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)
TEXT ·sgrFinish2RowAVX2(SB), NOSPLIT, $0-56
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a0+16(FP), R8
	MOVQ         a1+24(FP), R9
	MOVQ         b0+32(FP), R11
	MOVQ         b1+40(FP), R12
	MOVQ         n+48(FP), CX
	MOVL         $256, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop2:
	VMOVDQU 4(R8), Y0
	VPADDD  4(R9), Y0, Y0
	VMOVDQU (R8), Y1
	VPADDD  (R9), Y1, Y1
	VPADDD  8(R8), Y1, Y1
	VPADDD  8(R9), Y1, Y1
	VPSLLD  $2, Y0, Y2
	VPSLLD  $1, Y0, Y0
	VPADDD  Y2, Y0, Y0
	VPSLLD  $2, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R11, Y3)
	LOADB(2, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(0, R11, Y5)
	LOADB(0, R12, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $16, R11
	ADDQ $16, R12
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop2

	VZEROUPPER
	RET

// func sgrFinish2MidAVX2(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)
TEXT ·sgrFinish2MidAVX2(SB), NOSPLIT, $0-40
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a1+16(FP), R9
	MOVQ         b1+24(FP), R12
	MOVQ         n+32(FP), CX
	MOVL         $128, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop3:
	VMOVDQU 4(R9), Y0
	VMOVDQU (R9), Y1
	VPADDD  8(R9), Y1, Y1
	VPSLLD  $2, Y0, Y2
	VPSLLD  $1, Y0, Y0
	VPADDD  Y2, Y0, Y0
	VPSLLD  $2, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R12, Y3)
	LOADB(0, R12, Y5)
	LOADB(4, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($8)

	ADDQ $32, R9
	ADDQ $16, R12
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop3

	VZEROUPPER
	RET

// BOXV folds one more row into the running sums.
#define BOXV(RSQ, RSUM)         \
	VMOVDQU (RSQ)(DX*4), Y1; \
	VPADDD  Y1, Y0, Y0;     \
	VMOVDQU (RSUM)(DX*2), X3; \
	VPADDW  X3, X2, X2

// func sgrBoxV3AVX2(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV3AVX2(SB), NOSPLIT, $0-72
	MOVQ sq0+0(FP), R8
	MOVQ sq1+8(FP), R9
	MOVQ sq2+16(FP), R10
	MOVQ s0+24(FP), R11
	MOVQ s1+32(FP), R12
	MOVQ s2+40(FP), R13
	MOVQ sqOut+48(FP), DI
	MOVQ sOut+56(FP), SI
	MOVQ n+64(FP), CX
	XORQ DX, DX

boxv3:
	VMOVDQU (R8)(DX*4), Y0
	VMOVDQU (R11)(DX*2), X2
	BOXV(R9, R12)
	BOXV(R10, R13)
	VMOVDQU Y0, (DI)(DX*4)
	VMOVDQU X2, (SI)(DX*2)
	ADDQ    $8, DX
	SUBQ    $8, CX
	JNZ     boxv3

	VZEROUPPER
	RET

// func sgrBoxV5AVX2(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV5AVX2(SB), NOSPLIT, $0-104
	MOVQ sq0+0(FP), R8
	MOVQ sq1+8(FP), R9
	MOVQ sq2+16(FP), R10
	MOVQ s0+40(FP), R11
	MOVQ s1+48(FP), R12
	MOVQ s2+56(FP), R13
	MOVQ sqOut+80(FP), DI
	MOVQ sOut+88(FP), SI
	MOVQ n+96(FP), CX
	XORQ DX, DX

boxv5:
	VMOVDQU (R8)(DX*4), Y0
	VMOVDQU (R11)(DX*2), X2
	BOXV(R9, R12)
	BOXV(R10, R13)
	MOVQ    sq3+24(FP), AX
	MOVQ    s3+64(FP), BX
	BOXV(AX, BX)
	MOVQ    sq4+32(FP), AX
	MOVQ    s4+72(FP), BX
	BOXV(AX, BX)
	VMOVDQU Y0, (DI)(DX*4)
	VMOVDQU X2, (SI)(DX*2)
	ADDQ    $8, DX
	SUBQ    $8, CX
	JNZ     boxv5

	VZEROUPPER
	RET

// BOXH folds one more sample of the horizontal window into the sum and the
// sum of squares.
#define BOXH(OFF)              \
	VPMOVZXBD OFF(SI), Y1; \
	VPADDD    Y1, Y0, Y0;  \
	VPMULLD   Y1, Y1, Y1;  \
	VPADDD    Y1, Y2, Y2

// PACKH narrows the running sum to sixteen bits and stores both rows.
#define PACKH                  \
	VPACKSSDW Y0, Y0, Y0;  \
	VPERMQ    $0x08, Y0, Y0; \
	VMOVDQU   X0, (BX);    \
	VMOVDQU   Y2, (DI);    \
	ADDQ      $8, SI;      \
	ADDQ      $16, BX;     \
	ADDQ      $32, DI;     \
	SUBQ      $8, CX

// func sgrBoxH3AVX2(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH3AVX2(SB), NOSPLIT, $0-32
	MOVQ sumsq+0(FP), DI
	MOVQ sum+8(FP), BX
	MOVQ src+16(FP), SI
	MOVQ n+24(FP), CX

boxh3:
	VPMOVZXBD -1(SI), Y0
	VPMULLD   Y0, Y0, Y2
	BOXH(0)
	BOXH(1)
	PACKH
	JNZ       boxh3

	VZEROUPPER
	RET

// func sgrBoxH5AVX2(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH5AVX2(SB), NOSPLIT, $0-32
	MOVQ sumsq+0(FP), DI
	MOVQ sum+8(FP), BX
	MOVQ src+16(FP), SI
	MOVQ n+24(FP), CX

boxh5:
	VPMOVZXBD -2(SI), Y0
	VPMULLD   Y0, Y0, Y2
	BOXH(-1)
	BOXH(0)
	BOXH(1)
	BOXH(2)
	PACKH
	JNZ       boxh5

	VZEROUPPER
	RET

// func sgrCalcABAVX2(aa *int32, bb *int16, tab *[256]uint32, n int, s, mul, oneByX uint32)
TEXT ·sgrCalcABAVX2(SB), NOSPLIT, $0-44
	MOVQ         aa+0(FP), DI
	MOVQ         bb+8(FP), BX
	MOVQ         tab+16(FP), R8
	MOVQ         n+24(FP), CX
	MOVL         s+32(FP), AX
	VMOVD        AX, X10
	VPBROADCASTD X10, Y10
	MOVL         mul+36(FP), AX
	VMOVD        AX, X11
	VPBROADCASTD X11, Y11
	MOVL         oneByX+40(FP), AX
	VMOVD        AX, X12
	VPBROADCASTD X12, Y12
	MOVL         $(1 << 19), AX
	VMOVD        AX, X13
	VPBROADCASTD X13, Y13
	MOVL         $255, AX
	VMOVD        AX, X14
	VPBROADCASTD X14, Y14
	MOVL         $(1 << 11), AX
	VMOVD        AX, X15
	VPBROADCASTD X15, Y15
	VPXOR        Y9, Y9, Y9

calcab:
	VMOVDQU   (DI), Y0
	VPMOVSXWD (BX), Y1
	VPMULLD   Y11, Y0, Y2
	VPMULLD   Y1, Y1, Y3
	VPSUBD    Y3, Y2, Y2
	VPMAXSD   Y9, Y2, Y2
	VPMULLD   Y10, Y2, Y2
	VPADDD    Y13, Y2, Y2
	VPSRLD    $20, Y2, Y2
	VPMINUD   Y14, Y2, Y2

	VPCMPEQD  Y4, Y4, Y4
	VPGATHERDD Y4, (R8)(Y2*4), Y5

	VPMULLD   Y5, Y1, Y6
	VPMULLD   Y12, Y6, Y6
	VPADDD    Y15, Y6, Y6
	VPSRLD    $12, Y6, Y6
	VMOVDQU   Y6, (DI)
	VPACKSSDW Y5, Y5, Y5
	VPERMQ    $0x08, Y5, Y5
	VMOVDQU   X5, (BX)

	ADDQ $32, DI
	ADDQ $16, BX
	SUBQ $8, CX
	JNZ  calcab

	VZEROUPPER
	RET

// WEND rounds the weighted sum, folds it into the destination and stores it.
#define WEND                   \
	VPADDD    Y8, Y0, Y0;  \
	VPSRAD    $11, Y0, Y0; \
	VPMOVZXBD (DI), Y1;    \
	VPADDD    Y1, Y0, Y0;  \
	VPMAXSD   Y9, Y0, Y0;  \
	VPMINSD   Y10, Y0, Y0; \
	VPACKUSDW Y0, Y0, Y0;  \
	VPERMQ    $0x08, Y0, Y0; \
	VPACKUSWB X0, X0, X0;  \
	VMOVQ     X0, (DI);    \
	ADDQ      $8, DI;      \
	ADDQ      $16, SI;     \
	SUBQ      $8, CX

// func sgrWeight1AVX2(dst *uint8, t1 *int16, n int, w1 int32)
TEXT ·sgrWeight1AVX2(SB), NOSPLIT, $0-28
	MOVQ dst+0(FP), DI
	MOVQ t1+8(FP), SI
	MOVQ n+16(FP), CX
	MOVL w1+24(FP), AX
	VMOVD AX, X11
	VPBROADCASTD X11, Y11
	MOVL $1024, AX
	VMOVD AX, X8
	VPBROADCASTD X8, Y8
	VPXOR Y9, Y9, Y9
	MOVL $255, AX
	VMOVD AX, X10
	VPBROADCASTD X10, Y10

weight1:
	VPMOVSXWD (SI), Y0
	VPMULLD   Y11, Y0, Y0
	WEND
	JNZ       weight1

	VZEROUPPER
	RET

// func sgrWeight2AVX2(dst *uint8, t1, t2 *int16, n int, w0, w1 int32)
TEXT ·sgrWeight2AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ t1+8(FP), SI
	MOVQ t2+16(FP), DX
	MOVQ n+24(FP), CX
	MOVL         w0+32(FP), AX
	VMOVD        AX, X12
	VPBROADCASTD X12, Y12
	MOVL         w1+36(FP), AX
	VMOVD        AX, X11
	VPBROADCASTD X11, Y11
	MOVL         $1024, AX
	VMOVD        AX, X8
	VPBROADCASTD X8, Y8
	VPXOR        Y9, Y9, Y9
	MOVL         $255, AX
	VMOVD        AX, X10
	VPBROADCASTD X10, Y10

weight2:
	VPMOVSXWD (SI), Y0
	VPMULLD   Y12, Y0, Y0
	VPMOVSXWD (DX), Y2
	VPMULLD   Y11, Y2, Y2
	VPADDD    Y2, Y0, Y0
	ADDQ      $16, DX
	WEND
	JNZ       weight2

	VZEROUPPER
	RET
