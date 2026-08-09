//go:build amd64 && !noasm

#include "textflag.h"

DATA cdefRound<>+0(SB)/8, $0x0008000800080008
DATA cdefRound<>+8(SB)/8, $0x0008000800080008
GLOBL cdefRound<>(SB), RODATA|NOPTR, $16

// TAP accumulates one neighbour into the running sum. X0 is the pixel, X1 the
// sum, X12 the fill value that padding writes outside the frame; a tap that
// reads it must contribute nothing, which the mask in X5 forces.
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

// TAPC also tracks the neighbourhood range the result is clipped to. The fill
// value loses both comparisons on its own: it is the smallest signed word, and
// unsigned it is larger than any pixel.
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

// ROUND turns the accumulated sum into the filtered pixel, still as words.
#define ROUND                          \
	VPSRAW  $15, X1, X4;           \
	VPADDW  X4, X1, X1;            \
	VPADDW  cdefRound<>(SB), X1, X1; \
	VPSRAW  $4, X1, X1;            \
	VPADDW  X0, X1, X1

// func cdefFilterAVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterAVX2(SB), NOSPLIT, $0-32
	SETUP

loop:
	VPMOVZXBW (DI), X0
	VPXOR     X1, X1, X1

	VPBROADCASTW 40(BX), X13
	TAP(R8, X8, X10, X13)
	NEGQ R8
	TAP(R8, X8, X10, X13)
	NEGQ R8

	VPBROADCASTW 44(BX), X13
	TAP(R9, X8, X10, X13)
	NEGQ R9
	TAP(R9, X8, X10, X13)
	NEGQ R9

	VPBROADCASTW 48(BX), X13
	TAP(R10, X9, X11, X13)
	NEGQ R10
	TAP(R10, X9, X11, X13)
	NEGQ R10
	TAP(R12, X9, X11, X13)
	NEGQ R12
	TAP(R12, X9, X11, X13)
	NEGQ R12

	VPBROADCASTW 52(BX), X13
	TAP(R11, X9, X11, X13)
	NEGQ R11
	TAP(R11, X9, X11, X13)
	NEGQ R11
	TAP(R13, X9, X11, X13)
	NEGQ R13
	TAP(R13, X9, X11, X13)
	NEGQ R13

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

	VPBROADCASTW 40(BX), X13
	TAPC(R8, X8, X10, X13)
	NEGQ R8
	TAPC(R8, X8, X10, X13)
	NEGQ R8

	VPBROADCASTW 44(BX), X13
	TAPC(R9, X8, X10, X13)
	NEGQ R9
	TAPC(R9, X8, X10, X13)
	NEGQ R9

	VPBROADCASTW 48(BX), X13
	TAPC(R10, X9, X11, X13)
	NEGQ R10
	TAPC(R10, X9, X11, X13)
	NEGQ R10
	TAPC(R12, X9, X11, X13)
	NEGQ R12
	TAPC(R12, X9, X11, X13)
	NEGQ R12

	VPBROADCASTW 52(BX), X13
	TAPC(R11, X9, X11, X13)
	NEGQ R11
	TAPC(R11, X9, X11, X13)
	NEGQ R11
	TAPC(R13, X9, X11, X13)
	NEGQ R13
	TAPC(R13, X9, X11, X13)
	NEGQ R13

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
