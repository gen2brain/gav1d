//go:build amd64 && !noasm

#include "textflag.h"

// At high bit depth the sums are int32 like the squares, so both rows move in
// dwords and the pack down to words that the eight bit kernels do disappears.

#define LOADB16(OFF, REG, VD) \
	VMOVDQU OFF(REG), VD

#define TAIL16(SHIFT)         \
	VPMOVZXWD (SI), Y4;   \
	VPMULLD   Y4, Y3, Y3; \
	VPSUBD    Y3, Y0, Y0; \
	VPADDD    Y6, Y0, Y0; \
	VPSRAD    SHIFT, Y0, Y0; \
	VMOVDQU   Y0, (DI)

// func sgrFinish1_16AVX2(tmp *int32, src *uint16, a0, a1, a2, b0, b1, b2 *int32, n int)
TEXT ·sgrFinish1_16AVX2(SB), NOSPLIT, $0-72
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

loop1_16:
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

	LOADB16(4, R12, Y3)
	LOADB16(0, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB16(8, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB16(4, R11, Y4)
	VPADDD Y4, Y3, Y3
	LOADB16(4, R13, Y4)
	VPADDD Y4, Y3, Y3
	LOADB16(0, R11, Y5)
	LOADB16(0, R13, Y4)
	VPADDD Y4, Y5, Y5
	LOADB16(8, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB16(8, R13, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y3
	VPSLLD $1, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL16($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $32, R10
	ADDQ $32, R11
	ADDQ $32, R12
	ADDQ $32, R13
	ADDQ $16, SI
	ADDQ $32, DI
	SUBQ $8, CX
	JNZ  loop1_16

	VZEROUPPER
	RET

// func sgrFinish2Row_16AVX2(tmp *int32, src *uint16, a0, a1, b0, b1 *int32, n int)
TEXT ·sgrFinish2Row_16AVX2(SB), NOSPLIT, $0-56
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

loop2_16:
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

	LOADB16(4, R11, Y3)
	LOADB16(4, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB16(0, R11, Y5)
	LOADB16(0, R12, Y4)
	VPADDD Y4, Y5, Y5
	LOADB16(8, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB16(8, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL16($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $32, R11
	ADDQ $32, R12
	ADDQ $16, SI
	ADDQ $32, DI
	SUBQ $8, CX
	JNZ  loop2_16

	VZEROUPPER
	RET

// func sgrFinish2Mid_16AVX2(tmp *int32, src *uint16, a1, b1 *int32, n int)
TEXT ·sgrFinish2Mid_16AVX2(SB), NOSPLIT, $0-40
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a1+16(FP), R9
	MOVQ         b1+24(FP), R12
	MOVQ         n+32(FP), CX
	MOVL         $128, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop3_16:
	VMOVDQU 4(R9), Y0
	VMOVDQU (R9), Y1
	VPADDD  8(R9), Y1, Y1
	VPSLLD  $2, Y0, Y2
	VPSLLD  $1, Y0, Y0
	VPADDD  Y2, Y0, Y0
	VPSLLD  $2, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB16(4, R12, Y3)
	LOADB16(0, R12, Y5)
	LOADB16(8, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL16($8)

	ADDQ $32, R9
	ADDQ $32, R12
	ADDQ $16, SI
	ADDQ $32, DI
	SUBQ $8, CX
	JNZ  loop3_16

	VZEROUPPER
	RET

#define BOXV16(RSQ, RSUM)         \
	VMOVDQU (RSQ)(DX*4), Y1;  \
	VPADDD  Y1, Y0, Y0;       \
	VMOVDQU (RSUM)(DX*4), Y3; \
	VPADDD  Y3, Y2, Y2

// func sgrBoxV3_16AVX2(sq0, sq1, sq2, s0, s1, s2, sqOut, sOut *int32, n int)
TEXT ·sgrBoxV3_16AVX2(SB), NOSPLIT, $0-72
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

boxv3_16:
	VMOVDQU (R8)(DX*4), Y0
	VMOVDQU (R11)(DX*4), Y2
	BOXV16(R9, R12)
	BOXV16(R10, R13)
	VMOVDQU Y0, (DI)(DX*4)
	VMOVDQU Y2, (SI)(DX*4)
	ADDQ    $8, DX
	SUBQ    $8, CX
	JNZ     boxv3_16

	VZEROUPPER
	RET

// func sgrBoxV5_16AVX2(sq0, sq1, sq2, sq3, sq4, s0, s1, s2, s3, s4, sqOut, sOut *int32, n int)
TEXT ·sgrBoxV5_16AVX2(SB), NOSPLIT, $0-104
	MOVQ sq0+0(FP), R8
	MOVQ sq1+8(FP), R9
	MOVQ sq2+16(FP), R10
	MOVQ sq3+24(FP), R11
	MOVQ sq4+32(FP), R12
	MOVQ sqOut+80(FP), DI
	MOVQ sOut+88(FP), SI
	MOVQ n+96(FP), CX
	XORQ DX, DX

boxv5_16:
	VMOVDQU (R8)(DX*4), Y0
	VMOVDQU (R9)(DX*4), Y1
	VPADDD  Y1, Y0, Y0
	VMOVDQU (R10)(DX*4), Y1
	VPADDD  Y1, Y0, Y0
	VMOVDQU (R11)(DX*4), Y1
	VPADDD  Y1, Y0, Y0
	VMOVDQU (R12)(DX*4), Y1
	VPADDD  Y1, Y0, Y0
	VMOVDQU Y0, (DI)(DX*4)

	MOVQ    s0+40(FP), R13
	VMOVDQU (R13)(DX*4), Y2
	MOVQ    s1+48(FP), R13
	VMOVDQU (R13)(DX*4), Y3
	VPADDD  Y3, Y2, Y2
	MOVQ    s2+56(FP), R13
	VMOVDQU (R13)(DX*4), Y3
	VPADDD  Y3, Y2, Y2
	MOVQ    s3+64(FP), R13
	VMOVDQU (R13)(DX*4), Y3
	VPADDD  Y3, Y2, Y2
	MOVQ    s4+72(FP), R13
	VMOVDQU (R13)(DX*4), Y3
	VPADDD  Y3, Y2, Y2
	VMOVDQU Y2, (SI)(DX*4)

	ADDQ $8, DX
	SUBQ $8, CX
	JNZ  boxv5_16

	VZEROUPPER
	RET

#define BOXH16(OFF)            \
	VPMOVZXWD OFF(SI), Y1; \
	VPADDD    Y1, Y0, Y0;  \
	VPMULLD   Y1, Y1, Y1;  \
	VPADDD    Y1, Y2, Y2

#define PACKH16           \
	VMOVDQU Y0, (BX); \
	VMOVDQU Y2, (DI); \
	ADDQ    $16, SI;  \
	ADDQ    $32, BX;  \
	ADDQ    $32, DI;  \
	SUBQ    $8, CX

// func sgrBoxH3_16AVX2(sumsq, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH3_16AVX2(SB), NOSPLIT, $0-32
	MOVQ sumsq+0(FP), DI
	MOVQ sum+8(FP), BX
	MOVQ src+16(FP), SI
	MOVQ n+24(FP), CX

boxh3_16:
	VPMOVZXWD -2(SI), Y0
	VPMULLD   Y0, Y0, Y2
	BOXH16(0)
	BOXH16(2)
	PACKH16
	JNZ       boxh3_16

	VZEROUPPER
	RET

// func sgrBoxH5_16AVX2(sumsq, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH5_16AVX2(SB), NOSPLIT, $0-32
	MOVQ sumsq+0(FP), DI
	MOVQ sum+8(FP), BX
	MOVQ src+16(FP), SI
	MOVQ n+24(FP), CX

boxh5_16:
	VPMOVZXWD -4(SI), Y0
	VPMULLD   Y0, Y0, Y2
	BOXH16(-2)
	BOXH16(0)
	BOXH16(2)
	BOXH16(4)
	PACKH16
	JNZ       boxh5_16

	VZEROUPPER
	RET

// func sgrCalcAB_16AVX2(aa, bb *int32, tab *[256]uint32, n int, s, mul, oneByX uint32, rndA, shA, rndB, shB int32)
TEXT ·sgrCalcAB_16AVX2(SB), NOSPLIT, $0-60
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
	MOVL         rndA+44(FP), AX
	VMOVD        AX, X7
	VPBROADCASTD X7, Y7
	MOVL         rndB+52(FP), AX
	VMOVD        AX, X8
	VPBROADCASTD X8, Y8
	MOVL         shA+48(FP), AX
	VMOVD        AX, X5
	MOVL         shB+56(FP), AX
	VMOVD        AX, X6
	VPXOR        Y9, Y9, Y9

calcab_16:
	VMOVDQU (DI), Y0
	VMOVDQU (BX), Y1
	VPADDD  Y7, Y0, Y2
	VPSRAD  X5, Y2, Y2
	VPADDD  Y8, Y1, Y3
	VPSRAD  X6, Y3, Y3

	VPMULLD Y11, Y2, Y2
	VPMULLD Y3, Y3, Y3
	VPSUBD  Y3, Y2, Y2
	VPMAXSD Y9, Y2, Y2
	VPMULLD Y10, Y2, Y2
	VPADDD  Y13, Y2, Y2
	VPSRLD  $20, Y2, Y2
	VPMINUD Y14, Y2, Y2

	VPCMPEQD   Y4, Y4, Y4
	VPGATHERDD Y4, (R8)(Y2*4), Y3

	VPMULLD Y3, Y1, Y0
	VPMULLD Y12, Y0, Y0
	VPADDD  Y15, Y0, Y0
	VPSRLD  $12, Y0, Y0
	VMOVDQU Y0, (DI)
	VMOVDQU Y3, (BX)

	ADDQ $32, DI
	ADDQ $32, BX
	SUBQ $8, CX
	JNZ  calcab_16

	VZEROUPPER
	RET

#define WEND16                 \
	VPADDD    Y8, Y0, Y0;  \
	VPSRAD    $11, Y0, Y0; \
	VPMOVZXWD (DI), Y1;    \
	VPADDD    Y1, Y0, Y0;  \
	VPMAXSD   Y9, Y0, Y0;  \
	VPMINSD   Y10, Y0, Y0; \
	VPACKUSDW Y0, Y0, Y0;  \
	VPERMQ    $0x08, Y0, Y0; \
	VMOVDQU   X0, (DI);    \
	ADDQ      $16, DI;     \
	ADDQ      $32, SI;     \
	SUBQ      $8, CX

// func sgrWeight1_16AVX2(dst *uint16, t1 *int32, n int, w1, bitdepthMax int32)
TEXT ·sgrWeight1_16AVX2(SB), NOSPLIT, $0-32
	MOVQ         dst+0(FP), DI
	MOVQ         t1+8(FP), SI
	MOVQ         n+16(FP), CX
	MOVL         w1+24(FP), AX
	VMOVD        AX, X11
	VPBROADCASTD X11, Y11
	MOVL         $1024, AX
	VMOVD        AX, X8
	VPBROADCASTD X8, Y8
	VPXOR        Y9, Y9, Y9
	MOVL         bitdepthMax+28(FP), AX
	VMOVD        AX, X10
	VPBROADCASTD X10, Y10

weight1_16:
	VMOVDQU (SI), Y0
	VPMULLD Y11, Y0, Y0
	WEND16
	JNZ     weight1_16

	VZEROUPPER
	RET

// func sgrWeight2_16AVX2(dst *uint16, t1, t2 *int32, n int, w0, w1, bitdepthMax int32)
TEXT ·sgrWeight2_16AVX2(SB), NOSPLIT, $0-44
	MOVQ         dst+0(FP), DI
	MOVQ         t1+8(FP), SI
	MOVQ         t2+16(FP), BX
	MOVQ         n+24(FP), CX
	MOVL         w0+32(FP), AX
	VMOVD        AX, X11
	VPBROADCASTD X11, Y11
	MOVL         w1+36(FP), AX
	VMOVD        AX, X12
	VPBROADCASTD X12, Y12
	MOVL         $1024, AX
	VMOVD        AX, X8
	VPBROADCASTD X8, Y8
	VPXOR        Y9, Y9, Y9
	MOVL         bitdepthMax+40(FP), AX
	VMOVD        AX, X10
	VPBROADCASTD X10, Y10

weight2_16:
	VMOVDQU (SI), Y0
	VPMULLD Y11, Y0, Y0
	VMOVDQU (BX), Y2
	VPMULLD Y12, Y2, Y2
	VPADDD  Y2, Y0, Y0
	ADDQ    $32, BX
	WEND16
	JNZ     weight2_16

	VZEROUPPER
	RET
