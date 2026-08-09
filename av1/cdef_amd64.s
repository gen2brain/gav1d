//go:build amd64 && !noasm

#include "textflag.h"

DATA cdefRound<>+0(SB)/8, $0x0008000800080008
DATA cdefRound<>+8(SB)/8, $0x0008000800080008
GLOBL cdefRound<>(SB), RODATA|NOPTR, $16

// TAP accumulates one neighbour. X12 is the out-of-frame fill, which the mask
// in X5 forces to contribute nothing.
#define TAP(OFF, THR, SH, MUL)     \
	VMOVDQU  (SI)(OFF*2), X4;  \
	VPCMPEQW X12, X4, X5;      \
	VPSUBW   X0, X4, X6;       \
	VPABSW   X6, X7;           \
	VPSRLW   SH, X7, X14;      \
	VPSUBUSW X14, THR, X14;    \
	VPMINSW  X7, X14, X14;     \
	VPANDN   X14, X5, X14;     \
	VPSIGNW  X6, X14, X14;     \
	VPMULLW  MUL, X14, X14;    \
	VPADDW   X14, X1, X1

// TAPC also tracks the range the result is clipped to; the fill loses both
// comparisons on its own.
#define TAPC(OFF, THR, SH, MUL)    \
	TAP(OFF, THR, SH, MUL);    \
	VPMAXSW  X4, X2, X2;       \
	VPMINUW  X4, X3, X3

#define SETUP                          \
	MOVQ    dst+0(FP), DI;         \
	MOVQ    dstStride+8(FP), DX;   \
	MOVQ    tmp+16(FP), SI;        \
	MOVQ    p+24(FP), BX;          \
	MOVLQSX 0(BX), R8;             \
	MOVLQSX 4(BX), R9;             \
	MOVLQSX 8(BX), R10;            \
	MOVLQSX 12(BX), R11;           \
	MOVLQSX 16(BX), R12;           \
	MOVLQSX 20(BX), R13;           \
	VPBROADCASTW 24(BX), X8;       \
	VPBROADCASTW 28(BX), X9;       \
	VMOVD   32(BX), X10;           \
	VMOVD   36(BX), X11;           \
	VPCMPEQW X12, X12, X12;        \
	VPSLLW  $15, X12, X12;         \
	MOVLQSX 56(BX), AX;            \
	MOVLQSX 60(BX), CX

#define ROUND                          \
	VPSRAW  $15, X1, X4;           \
	VPADDW  X4, X1, X1;            \
	VPADDW  cdefRound<>(SB), X1, X1; \
	VPSRAW  $4, X1, X1;            \
	VPADDW  X0, X1, X1

// TAPS runs the four primary and eight secondary neighbours through T.
#define TAPS(T)                    \
	VPBROADCASTW 40(BX), X13;  \
	T(R8, X8, X10, X13);       \
	NEGQ R8;                   \
	T(R8, X8, X10, X13);       \
	NEGQ R8;                   \
	VPBROADCASTW 44(BX), X13;  \
	T(R9, X8, X10, X13);       \
	NEGQ R9;                   \
	T(R9, X8, X10, X13);       \
	NEGQ R9;                   \
	VPBROADCASTW 48(BX), X13;  \
	T(R10, X9, X11, X13);      \
	NEGQ R10;                  \
	T(R10, X9, X11, X13);      \
	NEGQ R10;                  \
	T(R12, X9, X11, X13);      \
	NEGQ R12;                  \
	T(R12, X9, X11, X13);      \
	NEGQ R12;                  \
	VPBROADCASTW 52(BX), X13;  \
	T(R11, X9, X11, X13);      \
	NEGQ R11;                  \
	T(R11, X9, X11, X13);      \
	NEGQ R11;                  \
	T(R13, X9, X11, X13);      \
	NEGQ R13;                  \
	T(R13, X9, X11, X13);      \
	NEGQ R13

// func cdefFilterAVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterAVX2(SB), NOSPLIT, $0-32
	SETUP

loop:
	VPMOVZXBW (DI), X0
	VPXOR     X1, X1, X1

	TAPS(TAP)

	ROUND
	VPACKUSWB X1, X1, X1
	CMPQ      AX, $8
	JNE       store4
	VMOVQ     X1, (DI)
	JMP       stored

store4:
	VMOVD X1, (DI)

stored:

	ADDQ DX, DI
	ADDQ $24, SI
	DECQ CX
	JNZ  loop

	VZEROUPPER
	RET

// func cdefFilterClipAVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClipAVX2(SB), NOSPLIT, $0-32
	SETUP

loopc:
	VPMOVZXBW (DI), X0
	VPXOR     X1, X1, X1
	VMOVDQU   X0, X2
	VMOVDQU   X0, X3

	TAPS(TAPC)

	ROUND
	VPMAXSW X3, X1, X1
	VPMINSW   X2, X1, X1
	VPACKUSWB X1, X1, X1
	CMPQ      AX, $8
	JNE       store4c
	VMOVQ     X1, (DI)
	JMP       storedc

store4c:
	VMOVD X1, (DI)

storedc:

	ADDQ DX, DI
	ADDQ $24, SI
	DECQ CX
	JNZ  loopc

	VZEROUPPER
	RET

// The 16-bit pixel needs no widening, and the store clips explicitly where
// VPACKUSWB did it for free at eight bits.
// func cdefFilter16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter16AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHLQ         $1, DX
	VPBROADCASTW 64(BX), X15
	VPXOR        X2, X2, X2

loop16:
	VMOVDQU (DI), X0
	VPXOR   X1, X1, X1
	TAPS(TAP)
	ROUND
	VPMAXSW X2, X1, X1
	VPMINSW X15, X1, X1

	CMPQ    AX, $8
	JNE     store416
	VMOVDQU X1, (DI)
	JMP     stored16

store416:
	VMOVQ X1, (DI)

stored16:
	ADDQ DX, DI
	ADDQ $24, SI
	DECQ CX
	JNZ  loop16

	VZEROUPPER
	RET

// func cdefFilterClip16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip16AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHLQ $1, DX

loopc16:
	VMOVDQU (DI), X0
	VPXOR   X1, X1, X1
	VMOVDQU X0, X2
	VMOVDQU X0, X3
	TAPS(TAPC)
	ROUND
	VPMAXSW X3, X1, X1
	VPMINSW X2, X1, X1

	CMPQ    AX, $8
	JNE     store4c16
	VMOVDQU X1, (DI)
	JMP     storedc16

store4c16:
	VMOVQ X1, (DI)

storedc16:
	ADDQ DX, DI
	ADDQ $24, SI
	DECQ CX
	JNZ  loopc16

	VZEROUPPER
	RET
