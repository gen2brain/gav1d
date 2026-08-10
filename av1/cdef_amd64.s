//go:build amd64 && !noasm

#include "textflag.h"

DATA cdefRound<>+0(SB)/8, $0x0008000800080008
DATA cdefRound<>+8(SB)/8, $0x0008000800080008
DATA cdefRound<>+16(SB)/8, $0x0008000800080008
DATA cdefRound<>+24(SB)/8, $0x0008000800080008
GLOBL cdefRound<>(SB), RODATA|NOPTR, $32

// V0 to V15 and LOADTAP carry the lane count, so the same taps serve the one
// row and the two row kernels.
#define LOADTAP(OFF) VMOVDQU (SI)(OFF*2), V4
#define V0 X0
#define V1 X1
#define V2 X2
#define V3 X3
#define V4 X4
#define V5 X5
#define V6 X6
#define V7 X7
#define V8 X8
#define V9 X9
#define V12 X12
#define V13 X13
#define V14 X14
#define V15 X15

// TAP accumulates one neighbour. V12 is the out-of-frame fill, which the mask
// in V5 forces to contribute nothing.
#define TAP(OFF, THR, SH, MUL)     \
	LOADTAP(OFF);              \
	VPCMPEQW V12, V4, V5;      \
	VPSUBW   V0, V4, V6;       \
	VPABSW   V6, V7;           \
	VPSRLW   SH, V7, V14;      \
	VPSUBUSW V14, THR, V14;    \
	VPMINSW  V7, V14, V14;     \
	VPANDN   V14, V5, V14;     \
	VPSIGNW  V6, V14, V14;     \
	VPMULLW  MUL, V14, V14;    \
	VPADDW   V14, V1, V1

// TAPC also tracks the range the result is clipped to; the fill loses both
// comparisons on its own.
#define TAPC(OFF, THR, SH, MUL)    \
	TAP(OFF, THR, SH, MUL);    \
	VPMAXSW  V4, V2, V2;       \
	VPMINUW  V4, V3, V3

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
	VPBROADCASTW 24(BX), V8;       \
	VPBROADCASTW 28(BX), V9;       \
	VMOVD   32(BX), X10;           \
	VMOVD   36(BX), X11;           \
	VPCMPEQW V12, V12, V12;        \
	VPSLLW  $15, V12, V12;         \
	MOVLQSX 56(BX), AX;            \
	MOVLQSX 60(BX), CX

#define ROUND                          \
	VPSRAW  $15, V1, V4;           \
	VPADDW  V4, V1, V1;            \
	VPADDW  cdefRound<>(SB), V1, V1; \
	VPSRAW  $4, V1, V1;            \
	VPADDW  V0, V1, V1

// TAPS runs the four primary and eight secondary neighbours through T.
#define TAPS(T)                    \
	VPBROADCASTW 40(BX), V13;  \
	T(R8, V8, X10, V13);       \
	NEGQ R8;                   \
	T(R8, V8, X10, V13);       \
	NEGQ R8;                   \
	VPBROADCASTW 44(BX), V13;  \
	T(R9, V8, X10, V13);       \
	NEGQ R9;                   \
	T(R9, V8, X10, V13);       \
	NEGQ R9;                   \
	VPBROADCASTW 48(BX), V13;  \
	T(R10, V9, X11, V13);      \
	NEGQ R10;                  \
	T(R10, V9, X11, V13);      \
	NEGQ R10;                  \
	T(R12, V9, X11, V13);      \
	NEGQ R12;                  \
	T(R12, V9, X11, V13);      \
	NEGQ R12;                  \
	VPBROADCASTW 52(BX), V13;  \
	T(R11, V9, X11, V13);      \
	NEGQ R11;                  \
	T(R11, V9, X11, V13);      \
	NEGQ R11;                  \
	T(R13, V9, X11, V13);      \
	NEGQ R13;                  \
	T(R13, V9, X11, V13);      \
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

// The two row kernels take a pair of rows into each lane half. The scratch
// rows are twenty four bytes apart, so a tap is one load and one insert and
// the taps themselves do not change.
#undef LOADTAP
#define LOADTAP(OFF)                        \
	VMOVDQU     (SI)(OFF*2), X4;        \
	VINSERTI128 $1, 24(SI)(OFF*2), Y4, Y4
#undef V0
#define V0 Y0
#undef V1
#define V1 Y1
#undef V2
#define V2 Y2
#undef V3
#define V3 Y3
#undef V4
#define V4 Y4
#undef V5
#define V5 Y5
#undef V6
#define V6 Y6
#undef V7
#define V7 Y7
#undef V8
#define V8 Y8
#undef V9
#define V9 Y9
#undef V12
#define V12 Y12
#undef V13
#define V13 Y13
#undef V14
#define V14 Y14
#undef V15
#define V15 Y15

// func cdefFilter2AVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter2AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHRQ $1, CX

loop2:
	VPMOVZXBW   (DI), X0
	VPMOVZXBW   (DI)(DX*1), X6
	VINSERTI128 $1, X6, Y0, Y0
	VPXOR       V1, V1, V1

	TAPS(TAP)

	ROUND
	VPACKUSWB    V1, V1, V1
	VEXTRACTI128 $1, Y1, X6
	CMPQ         AX, $8
	JNE          store42
	VMOVQ        X1, (DI)
	VMOVQ        X6, (DI)(DX*1)
	JMP          stored2

store42:
	VMOVD X1, (DI)
	VMOVD X6, (DI)(DX*1)

stored2:
	LEAQ (DI)(DX*2), DI
	ADDQ $48, SI
	DECQ CX
	JNZ  loop2

	VZEROUPPER
	RET

// func cdefFilterClip2AVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip2AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHRQ $1, CX

loopc2:
	VPMOVZXBW   (DI), X0
	VPMOVZXBW   (DI)(DX*1), X6
	VINSERTI128 $1, X6, Y0, Y0
	VPXOR       V1, V1, V1
	VMOVDQU     V0, V2
	VMOVDQU     V0, V3

	TAPS(TAPC)

	ROUND
	VPMAXSW      V3, V1, V1
	VPMINSW      V2, V1, V1
	VPACKUSWB    V1, V1, V1
	VEXTRACTI128 $1, Y1, X6
	CMPQ         AX, $8
	JNE          store4c2
	VMOVQ        X1, (DI)
	VMOVQ        X6, (DI)(DX*1)
	JMP          storedc2

store4c2:
	VMOVD X1, (DI)
	VMOVD X6, (DI)(DX*1)

storedc2:
	LEAQ (DI)(DX*2), DI
	ADDQ $48, SI
	DECQ CX
	JNZ  loopc2

	VZEROUPPER
	RET

// func cdefFilter2_16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter2_16AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHLQ         $1, DX
	SHRQ         $1, CX
	VPBROADCASTW 64(BX), V15
	VPXOR        V2, V2, V2

loop216:
	VMOVDQU     (DI), X0
	VINSERTI128 $1, (DI)(DX*1), Y0, Y0
	VPXOR       V1, V1, V1

	TAPS(TAP)

	ROUND
	VPMAXSW      V2, V1, V1
	VPMINSW      V15, V1, V1
	VEXTRACTI128 $1, Y1, X6
	CMPQ         AX, $8
	JNE          store4216
	VMOVDQU      X1, (DI)
	VMOVDQU      X6, (DI)(DX*1)
	JMP          stored216

store4216:
	VMOVQ X1, (DI)
	VMOVQ X6, (DI)(DX*1)

stored216:
	LEAQ (DI)(DX*2), DI
	ADDQ $48, SI
	DECQ CX
	JNZ  loop216

	VZEROUPPER
	RET

// func cdefFilterClip2_16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip2_16AVX2(SB), NOSPLIT, $0-32
	SETUP
	SHLQ $1, DX
	SHRQ $1, CX

loopc216:
	VMOVDQU     (DI), X0
	VINSERTI128 $1, (DI)(DX*1), Y0, Y0
	VPXOR       V1, V1, V1
	VMOVDQU     V0, V2
	VMOVDQU     V0, V3

	TAPS(TAPC)

	ROUND
	VPMAXSW      V3, V1, V1
	VPMINSW      V2, V1, V1
	VEXTRACTI128 $1, Y1, X6
	CMPQ         AX, $8
	JNE          store4c216
	VMOVDQU      X1, (DI)
	VMOVDQU      X6, (DI)(DX*1)
	JMP          storedc216

store4c216:
	VMOVQ X1, (DI)
	VMOVQ X6, (DI)(DX*1)

storedc216:
	LEAQ (DI)(DX*2), DI
	ADDQ $48, SI
	DECQ CX
	JNZ  loopc216

	VZEROUPPER
	RET

