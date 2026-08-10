//go:build amd64 && !noasm

#include "textflag.h"

DATA wienerPerm<>+0(SB)/8, $0x0b0a030209080100
DATA wienerPerm<>+8(SB)/8, $0x0f0e07060d0c0504
GLOBL wienerPerm<>(SB), RODATA|NOPTR, $16

// HSETUP loads the four tap pairs, the rounding term and the clip bounds. The
// pairs are laid out so that the last one carries the seventh tap in its high
// half, which keeps every load inside the row.
#define HSETUP                     \
	MOVQ         c+24(FP), BX;     \
	VPBROADCASTD (BX), Y8;         \
	VPBROADCASTD 4(BX), Y9;        \
	VPBROADCASTD 8(BX), Y10;       \
	VPBROADCASTD 12(BX), Y11;      \
	MOVL         rnd+32(FP), AX;   \
	VMOVD        AX, X12;          \
	VPBROADCASTD X12, Y12;         \
	MOVQ         shift+40(FP), AX; \
	VMOVD        AX, X15;          \
	MOVL         limit+48(FP), AX; \
	VMOVD        AX, X13;          \
	VPBROADCASTD X13, Y13;         \
	VPXOR        Y14, Y14, Y14;    \
	VBROADCASTI128 wienerPerm<>(SB), Y7

// HTAIL rounds, shifts and clips the two halves, then interleaves them back
// into ascending order and stores sixteen samples.
#define HTAIL                 \
	VPADDD    Y12, Y0, Y0;    \
	VPADDD    Y12, Y2, Y2;    \
	VPSRAD    X15, Y0, Y0;    \
	VPSRAD    X15, Y2, Y2;    \
	VPMAXSD   Y14, Y0, Y0;    \
	VPMAXSD   Y14, Y2, Y2;    \
	VPMINSD   Y13, Y0, Y0;    \
	VPMINSD   Y13, Y2, Y2;    \
	VPACKUSDW Y2, Y0, Y0;     \
	VPSHUFB   Y7, Y0, Y0;     \
	VMOVDQU   Y0, (DI);       \
	ADDQ      $32, DI

// VSETUP resolves the seven row pointers and the tap pairs. DX walks all seven
// rows at once, so the loop advances a single index.
#define VSETUP                     \
	MOVQ         hor+8(FP), SI;    \
	MOVQ         rows+16(FP), R8;  \
	MOVQ         (R8), AX;         \
	LEAQ         (SI)(AX*2), R9;   \
	MOVQ         8(R8), AX;        \
	LEAQ         (SI)(AX*2), R10;  \
	MOVQ         16(R8), AX;       \
	LEAQ         (SI)(AX*2), R11;  \
	MOVQ         24(R8), AX;       \
	LEAQ         (SI)(AX*2), R12;  \
	MOVQ         32(R8), AX;       \
	LEAQ         (SI)(AX*2), R13;  \
	MOVQ         40(R8), AX;       \
	LEAQ         (SI)(AX*2), R14;  \
	MOVQ         48(R8), AX;       \
	LEAQ         (SI)(AX*2), R15;  \
	MOVQ         c+24(FP), BX;     \
	VPBROADCASTD (BX), Y8;         \
	VPBROADCASTD 4(BX), Y9;        \
	VPBROADCASTD 8(BX), Y10;       \
	VPBROADCASTD 12(BX), Y11;      \
	MOVQ         n+32(FP), CX;     \
	MOVL         rnd+40(FP), AX;   \
	VMOVD        AX, X12;          \
	VPBROADCASTD X12, Y12;         \
	MOVQ         shift+48(FP), AX; \
	VMOVD        AX, X15;          \
	MOVL         limit+56(FP), AX; \
	VMOVD        AX, X13;          \
	VPBROADCASTD X13, Y13;         \
	VPXOR        Y14, Y14, Y14;    \
	XORQ         DX, DX

// VPAIR folds two adjacent rows into the two accumulators.
#define VPAIR(RA, RB, COEF)          \
	VMOVDQU    (RA)(DX*1), Y0;       \
	VMOVDQU    (RB)(DX*1), Y1;       \
	VPUNPCKLWD Y1, Y0, Y2;           \
	VPUNPCKHWD Y1, Y0, Y3;           \
	VPMADDWD   COEF, Y2, Y2;         \
	VPMADDWD   COEF, Y3, Y3;         \
	VPADDD     Y2, Y4, Y4;           \
	VPADDD     Y3, Y5, Y5

#define VPAIRN(RA, RB, COEF)         \
	VMOVDQU    (RA)(DX*1), Y0;       \
	VMOVDQU    (RB)(DX*1), Y1;       \
	VPUNPCKLWD Y1, Y0, Y2;           \
	VPUNPCKHWD Y1, Y0, Y3;           \
	VPDPWSSD   COEF, Y2, Y4;         \
	VPDPWSSD   COEF, Y3, Y5

// VBODY accumulates all seven rows and lands the clipped result in Y4 as
// sixteen words in ascending order.
#define VBODY                        \
	VMOVDQU    (R9)(DX*1), Y0;       \
	VMOVDQU    (R10)(DX*1), Y1;      \
	VPUNPCKLWD Y1, Y0, Y2;           \
	VPUNPCKHWD Y1, Y0, Y3;           \
	VPMADDWD   Y8, Y2, Y4;           \
	VPMADDWD   Y8, Y3, Y5;           \
	VPAIR(R11, R12, Y9);             \
	VPAIR(R13, R14, Y10);            \
	VMOVDQU    (R15)(DX*1), Y0;      \
	VPUNPCKLWD Y0, Y14, Y2;          \
	VPUNPCKHWD Y0, Y14, Y3;          \
	VPMADDWD   Y11, Y2, Y2;          \
	VPMADDWD   Y11, Y3, Y3;          \
	VPADDD     Y2, Y4, Y4;           \
	VPADDD     Y3, Y5, Y5;           \
	VPADDD     Y12, Y4, Y4;          \
	VPADDD     Y12, Y5, Y5;          \
	VPSRAD     X15, Y4, Y4;          \
	VPSRAD     X15, Y5, Y5;          \
	VPMAXSD    Y14, Y4, Y4;          \
	VPMAXSD    Y14, Y5, Y5;          \
	VPMINSD    Y13, Y4, Y4;          \
	VPMINSD    Y13, Y5, Y5;          \
	VPACKUSDW  Y5, Y4, Y4

#define VBODYN                       \
	VMOVDQU    (R9)(DX*1), Y0;       \
	VMOVDQU    (R10)(DX*1), Y1;      \
	VPUNPCKLWD Y1, Y0, Y2;           \
	VPUNPCKHWD Y1, Y0, Y3;           \
	VPMADDWD   Y8, Y2, Y4;           \
	VPMADDWD   Y8, Y3, Y5;           \
	VPAIRN(R11, R12, Y9);            \
	VPAIRN(R13, R14, Y10);           \
	VMOVDQU    (R15)(DX*1), Y0;      \
	VPUNPCKLWD Y0, Y14, Y2;          \
	VPUNPCKHWD Y0, Y14, Y3;          \
	VPDPWSSD   Y11, Y2, Y4;          \
	VPDPWSSD   Y11, Y3, Y5;          \
	VPADDD     Y12, Y4, Y4;          \
	VPADDD     Y12, Y5, Y5;          \
	VPSRAD     X15, Y4, Y4;          \
	VPSRAD     X15, Y5, Y5;          \
	VPMAXSD    Y14, Y4, Y4;          \
	VPMAXSD    Y14, Y5, Y5;          \
	VPMINSD    Y13, Y4, Y4;          \
	VPMINSD    Y13, Y5, Y5;          \
	VPACKUSDW  Y5, Y4, Y4

#define HBODY8                   \
	VPMOVZXBW (SI), Y0;      \
	VPMADDWD  Y8, Y0, Y0;    \
	VPMOVZXBW 2(SI), Y1;     \
	VPMADDWD  Y9, Y1, Y1;    \
	VPADDD    Y1, Y0, Y0;    \
	VPMOVZXBW 4(SI), Y1;     \
	VPMADDWD  Y10, Y1, Y1;   \
	VPADDD    Y1, Y0, Y0;    \
	VPMOVZXBW 5(SI), Y1;     \
	VPMADDWD  Y11, Y1, Y1;   \
	VPADDD    Y1, Y0, Y0;    \
	VPMOVZXBW 1(SI), Y2;     \
	VPMADDWD  Y8, Y2, Y2;    \
	VPMOVZXBW 3(SI), Y1;     \
	VPMADDWD  Y9, Y1, Y1;    \
	VPADDD    Y1, Y2, Y2;    \
	VPMOVZXBW 5(SI), Y1;     \
	VPMADDWD  Y10, Y1, Y1;   \
	VPADDD    Y1, Y2, Y2;    \
	VPMOVZXBW 6(SI), Y1;     \
	VPMADDWD  Y11, Y1, Y1;   \
	VPADDD    Y1, Y2, Y2

#define HBODY8N                  \
	VPMOVZXBW (SI), Y0;      \
	VPMADDWD  Y8, Y0, Y0;    \
	VPMOVZXBW 2(SI), Y1;     \
	VPDPWSSD  Y9, Y1, Y0;    \
	VPMOVZXBW 4(SI), Y1;     \
	VPDPWSSD  Y10, Y1, Y0;   \
	VPMOVZXBW 5(SI), Y1;     \
	VPDPWSSD  Y11, Y1, Y0;   \
	VPMOVZXBW 1(SI), Y2;     \
	VPMADDWD  Y8, Y2, Y2;    \
	VPMOVZXBW 3(SI), Y1;     \
	VPDPWSSD  Y9, Y1, Y2;    \
	VPMOVZXBW 5(SI), Y1;     \
	VPDPWSSD  Y10, Y1, Y2;   \
	VPMOVZXBW 6(SI), Y1;     \
	VPDPWSSD  Y11, Y1, Y2

#define HBODY16                 \
	VMOVDQU  (SI), Y0;      \
	VPMADDWD Y8, Y0, Y0;    \
	VMOVDQU  4(SI), Y1;     \
	VPMADDWD Y9, Y1, Y1;    \
	VPADDD   Y1, Y0, Y0;    \
	VMOVDQU  8(SI), Y1;     \
	VPMADDWD Y10, Y1, Y1;   \
	VPADDD   Y1, Y0, Y0;    \
	VMOVDQU  10(SI), Y1;    \
	VPMADDWD Y11, Y1, Y1;   \
	VPADDD   Y1, Y0, Y0;    \
	VMOVDQU  2(SI), Y2;     \
	VPMADDWD Y8, Y2, Y2;    \
	VMOVDQU  6(SI), Y1;     \
	VPMADDWD Y9, Y1, Y1;    \
	VPADDD   Y1, Y2, Y2;    \
	VMOVDQU  10(SI), Y1;    \
	VPMADDWD Y10, Y1, Y1;   \
	VPADDD   Y1, Y2, Y2;    \
	VMOVDQU  12(SI), Y1;    \
	VPMADDWD Y11, Y1, Y1;   \
	VPADDD   Y1, Y2, Y2

#define HBODY16N                \
	VMOVDQU  (SI), Y0;      \
	VPMADDWD Y8, Y0, Y0;    \
	VMOVDQU  4(SI), Y1;     \
	VPDPWSSD Y9, Y1, Y0;    \
	VMOVDQU  8(SI), Y1;     \
	VPDPWSSD Y10, Y1, Y0;   \
	VMOVDQU  10(SI), Y1;    \
	VPDPWSSD Y11, Y1, Y0;   \
	VMOVDQU  2(SI), Y2;     \
	VPMADDWD Y8, Y2, Y2;    \
	VMOVDQU  6(SI), Y1;     \
	VPDPWSSD Y9, Y1, Y2;    \
	VMOVDQU  10(SI), Y1;    \
	VPDPWSSD Y10, Y1, Y2;   \
	VMOVDQU  12(SI), Y1;    \
	VPDPWSSD Y11, Y1, Y2

// WIENERH and WIENERV wrap one filter body in its loop, so the two tap forms
// share the setup and the tail.
#define WIENERH8(BODY)       \
	MOVQ dst+0(FP), DI;  \
	MOVQ src+8(FP), SI;  \
	MOVQ n+16(FP), CX;   \
	HSETUP;              \
	loop8:;              \
	BODY;                \
	HTAIL;               \
	ADDQ $16, SI;        \
	SUBQ $16, CX;        \
	JNZ  loop8;          \
	VZEROUPPER

#define WIENERH16(BODY)      \
	MOVQ dst+0(FP), DI;  \
	MOVQ src+8(FP), SI;  \
	MOVQ n+16(FP), CX;   \
	HSETUP;              \
	loop16:;             \
	BODY;                \
	HTAIL;               \
	ADDQ $32, SI;        \
	SUBQ $16, CX;        \
	JNZ  loop16;         \
	VZEROUPPER

#define WIENERV8(BODY)         \
	MOVQ p+0(FP), DI;      \
	VSETUP;                \
	vloop8:;               \
	BODY;                  \
	VPACKUSWB Y4, Y4, Y4;  \
	VPERMQ    $0x08, Y4, Y4; \
	VMOVDQU   X4, (DI);    \
	ADDQ      $16, DI;     \
	ADDQ      $32, DX;     \
	SUBQ      $16, CX;     \
	JNZ       vloop8;      \
	VZEROUPPER

#define WIENERV16(BODY)      \
	MOVQ p+0(FP), DI;    \
	VSETUP;              \
	vloop16:;            \
	BODY;                \
	VMOVDQU Y4, (DI);    \
	ADDQ    $32, DI;     \
	ADDQ    $32, DX;     \
	SUBQ    $16, CX;     \
	JNZ     vloop16;     \
	VZEROUPPER

// func wienerH8AVX2(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH8AVX2(SB), NOSPLIT, $0-52
	WIENERH8(HBODY8)
	RET

// func wienerH8AVX512(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH8AVX512(SB), NOSPLIT, $0-52
	WIENERH8(HBODY8N)
	RET

// func wienerH16AVX2(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH16AVX2(SB), NOSPLIT, $0-52
	WIENERH16(HBODY16)
	RET

// func wienerH16AVX512(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)
TEXT ·wienerH16AVX512(SB), NOSPLIT, $0-52
	WIENERH16(HBODY16N)
	RET

// func wienerV8AVX2(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV8AVX2(SB), NOSPLIT, $0-60
	WIENERV8(VBODY)
	RET

// func wienerV8AVX512(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV8AVX512(SB), NOSPLIT, $0-60
	WIENERV8(VBODYN)
	RET

// func wienerV16AVX2(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV16AVX2(SB), NOSPLIT, $0-60
	WIENERV16(VBODY)
	RET

// func wienerV16AVX512(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32, shift int, limit int32)
TEXT ·wienerV16AVX512(SB), NOSPLIT, $0-60
	WIENERV16(VBODYN)
	RET
