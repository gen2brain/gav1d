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

DATA cdefLut4<>+0(SB)/8, $0x4746454443424140
DATA cdefLut4<>+8(SB)/8, $0x4f4e4d4c4b4a4948
DATA cdefLut4<>+16(SB)/8, $0x0504030201001110
DATA cdefLut4<>+24(SB)/8, $0x0d0c0b0a09081312
DATA cdefLut4<>+32(SB)/8, $0x5554535251501514
DATA cdefLut4<>+40(SB)/8, $0x2524232221201716
DATA cdefLut4<>+48(SB)/8, $0x6968676665646362
DATA cdefLut4<>+56(SB)/8, $0x3938373635343332
GLOBL cdefLut4<>(SB), RODATA|NOPTR, $64

DATA cdefPxIdx<>+0(SB)/8, $0x1313131312121212
DATA cdefPxIdx<>+8(SB)/8, $0x1515151514141414
DATA cdefPxIdx<>+16(SB)/8, $0x1b1b1b1b1a1a1a1a
DATA cdefPxIdx<>+24(SB)/8, $0x1d1d1d1d1c1c1c1c
DATA cdefPxIdx<>+32(SB)/8, $0x2323232322222222
DATA cdefPxIdx<>+40(SB)/8, $0x2525252524242424
DATA cdefPxIdx<>+48(SB)/8, $0x2b2b2b2b2a2a2a2a
DATA cdefPxIdx<>+56(SB)/8, $0x2d2d2d2d2c2c2c2c
GLOBL cdefPxIdx<>(SB), RODATA|NOPTR, $64

DATA cdefDirs<>+0(SB)/8, $0x310f380830103808
DATA cdefDirs<>+8(SB)/8, $0x063a3f010e320739
DATA cdefDirs<>+16(SB)/8, $0x360a3f013e023f01
DATA cdefDirs<>+24(SB)/8, $0x2f1138082e123709
DATA cdefDirs<>+32(SB)/8, $0x310f380830103808
DATA cdefDirs<>+40(SB)/8, $0x063a3f010e320739
GLOBL cdefDirs<>(SB), RODATA|NOPTR, $48

DATA cdefGfShr<>+0(SB)/8, $0x0102040810204080
DATA cdefGfShr<>+8(SB)/8, $0x0204081020408000
DATA cdefGfShr<>+16(SB)/8, $0x0408102040800000
DATA cdefGfShr<>+24(SB)/8, $0x0810204080000000
DATA cdefGfShr<>+32(SB)/8, $0x1020408000000000
DATA cdefGfShr<>+40(SB)/8, $0x2040800000000000
DATA cdefGfShr<>+48(SB)/8, $0x4080000000000000
DATA cdefGfShr<>+56(SB)/8, $0x8000000000000000
GLOBL cdefGfShr<>(SB), RODATA|NOPTR, $64

DATA cdefEdgeMask<>+0(SB)/8, $0x00003c3c3c3c0000
DATA cdefEdgeMask<>+8(SB)/8, $0x00003f3f3f3f0000
DATA cdefEdgeMask<>+16(SB)/8, $0x0000fcfcfcfc0000
DATA cdefEdgeMask<>+24(SB)/8, $0x0000ffffffff0000
DATA cdefEdgeMask<>+32(SB)/8, $0x00003c3c3c3c3c3c
DATA cdefEdgeMask<>+40(SB)/8, $0x00003f3f3f3f3f3f
DATA cdefEdgeMask<>+48(SB)/8, $0x0000fcfcfcfcfcfc
DATA cdefEdgeMask<>+56(SB)/8, $0x0000ffffffffffff
DATA cdefEdgeMask<>+64(SB)/8, $0x3c3c3c3c3c3c0000
DATA cdefEdgeMask<>+72(SB)/8, $0x3f3f3f3f3f3f0000
DATA cdefEdgeMask<>+80(SB)/8, $0xfcfcfcfcfcfc0000
DATA cdefEdgeMask<>+88(SB)/8, $0xffffffffffff0000
DATA cdefEdgeMask<>+96(SB)/8, $0x3c3c3c3c3c3c3c3c
DATA cdefEdgeMask<>+104(SB)/8, $0x3f3f3f3f3f3f3f3f
DATA cdefEdgeMask<>+112(SB)/8, $0xfcfcfcfcfcfcfcfc
DATA cdefEdgeMask<>+120(SB)/8, $0xffffffffffffffff
GLOBL cdefEdgeMask<>(SB), RODATA|NOPTR, $128

DATA cdefPriTap<>+0(SB)/8, $0x3030303020204040
GLOBL cdefPriTap<>(SB), RODATA|NOPTR, $8

DATA cdefSecTap<>+0(SB)/8, $0x1010202010102020
GLOBL cdefSecTap<>(SB), RODATA|NOPTR, $8

DATA cdefEndPerm<>+0(SB)/8, $0x1d1915110d090501
DATA cdefEndPerm<>+8(SB)/8, $0x3d3935312d292521
GLOBL cdefEndPerm<>(SB), RODATA|NOPTR, $16

DATA cdefBias<>+0(SB)/8, $0x0000000010000070
GLOBL cdefBias<>(SB), RODATA|NOPTR, $8


// The whole eight by eight neighbourhood of a four by four block is 64 bytes,
// so it lives in one register and every tap is a permute of it rather than a
// load. px repeats each pixel four times so one VPDPBUSD takes four taps.
#define ZPRI       0(R11)
#define ZSEC       4(R11)
#define ZPRISHR    8(R11)
#define ZSECSHR   12(R11)
#define ZPRITAP   16(R11)
#define ZDIRPRI   20(R11)
#define ZDIRSEC0  24(R11)
#define ZDIRSEC1  28(R11)
#define ZEDGE     32(R11)

#define ZSECTAPS                          \
	MOVLQSX ZDIRSEC0, R13;            \
	LEAQ    cdefDirs<>(SB), R14;      \
	VPADDD.BCST (R14)(R13*1), Z3, Z2; \
	MOVLQSX ZDIRSEC1, R13;            \
	VPADDD.BCST (R14)(R13*1), Z3, Z3

#define ZMASKSEC                          \
	MOVLQSX ZDIRSEC0, R13;            \
	LEAQ    cdefDirs<>(SB), R14;      \
	VPADDD.BCST (R14)(R13*1), Z3, Z4; \
	MOVLQSX ZDIRSEC1, R13;            \
	VPADDD.BCST (R14)(R13*1), Z3, Z9; \
	VPSHUFBITQMB Z4, Z8, K1;          \
	VMOVDQU8 Z6, Z2;                  \
	VPERMB   Z5, Z4, K1, Z2;          \
	VPSHUFBITQMB Z9, Z8, K1;          \
	VMOVDQU8 Z6, Z3;                  \
	VPERMB   Z5, Z9, K1, Z3

#define ZSECMAIN                            \
	VPBROADCASTD   cdefSecTap<>(SB), Z8; \
	VPCMPUB        $6, Z2, Z6, K1;      \
	VPSUBB         Z6, Z2, Z4;          \
	MOVLQSX        ZSEC, R13;           \
	VPBROADCASTB   R13, Z12;            \
	MOVLQSX        ZSECSHR, R13;        \
	LEAQ           cdefGfShr<>(SB), R14; \
	VPBROADCASTQ   (R14)(R13*8), Z11;   \
	VPSUBB         Z2, Z6, K1, Z4;      \
	VPCMPUB        $6, Z3, Z6, K2;      \
	VGF2P8AFFINEQB $0, Z11, Z4, Z10;    \
	VPSUBB         Z6, Z3, Z5;          \
	VMOVDQU8       Z8, Z9;              \
	VPSUBB         Z8, Z7, K1, Z8;      \
	VPSUBUSB       Z10, Z12, Z10;       \
	VPSUBB         Z3, Z6, K2, Z5;      \
	VPMINUB        Z10, Z4, Z4;         \
	VPDPBUSD       Z8, Z4, Z0;          \
	VGF2P8AFFINEQB $0, Z11, Z5, Z11;    \
	VPSUBB         Z9, Z7, K2, Z9;      \
	VPSUBUSB       Z11, Z12, Z12;       \
	VPMINUB        Z12, Z5, Z5;         \
	VPDPBUSD       Z9, Z5, Z0

#define ZPRIMAIN                            \
	VPCMPUB        $6, Z1, Z6, K1;      \
	VPSUBB         Z6, Z1, Z2;          \
	MOVLQSX        ZPRISHR, R13;        \
	LEAQ           cdefGfShr<>(SB), R14; \
	VPSUBB         Z1, Z6, K1, Z2;      \
	VPBROADCASTQ   (R14)(R13*8), Z11;   \
	MOVLQSX        ZPRI, R13;           \
	VPBROADCASTB   R13, Z4;             \
	VGF2P8AFFINEQB $0, Z11, Z2, Z9;     \
	MOVLQSX        ZPRITAP, R13;        \
	LEAQ           cdefPriTap<>(SB), R14; \
	VPBROADCASTD   (R14)(R13*1), Z10;   \
	VPSUBB         Z10, Z7, K1, Z10;    \
	VPSUBUSB       Z9, Z4, Z4;          \
	VPMINUB        Z4, Z2, Z2;          \
	VPDPBUSD       Z10, Z2, Z0

// func cdef4x4AVX512(dst *uint8, stride int, left *uint8, top *uint8, bot *uint8, p *cdefZParams)
TEXT ·cdef4x4AVX512(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), DX
	MOVQ left+16(FP), R8
	MOVQ top+24(FP), R9
	MOVQ bot+32(FP), R10
	MOVQ p+40(FP), R11
	LEAQ (DX)(DX*2), R12

	VMOVQ        (DI), X0
	VMOVHPD      (DI)(DX*1), X0, X0
	VMOVQ        -2(R9), X1
	VMOVHPD      -2(R9)(DX*1), X1, X1
	VINSERTI32X4 $1, (R8), Y0, Y0
	VINSERTI32X4 $1, (DI)(DX*2), Y1, Y1
	VINSERTI32X4 $2, (DI)(R12*1), Z0, Z0
	MOVLQSX      ZEDGE, R13
	TESTQ        $8, R13
	JZ           zmain
	VINSERTI32X4 $2, -4(R10), Z1, Z1
	VINSERTI32X4 $3, -4(R10)(DX*1), Z0, Z0

zmain:
	VMOVDQU8     cdefLut4<>(SB), Z5
	VMOVDQU8     cdefPxIdx<>(SB), Z3
	VPERMI2B     Z1, Z0, Z5
	VPBROADCASTD cdefBias<>(SB), Z0
	VPXORD       Z7, Z7, Z7
	VPERMB       Z5, Z3, Z6

	CMPQ R13, $15
	JNE  zmaskedges

	MOVLQSX ZPRI, R14
	TESTQ   R14, R14
	JZ      zseconly
	MOVLQSX ZDIRPRI, R13
	LEAQ    cdefDirs<>(SB), R14
	VPADDD.BCST (R14)(R13*1), Z3, Z1
	VPERMB  Z5, Z1, Z1
	ZPRIMAIN
	MOVLQSX ZSEC, R13
	TESTQ   R13, R13
	JZ      zendnoclip
	ZSECTAPS
	VPERMB  Z5, Z2, Z2
	VPERMB  Z5, Z3, Z3
	ZSECMAIN
	JMP     zendclip

zseconly:
	ZSECTAPS
	VPERMB  Z5, Z2, Z2
	VPERMB  Z5, Z3, Z3
	ZSECMAIN
	JMP     zendnoclip

zmaskedges:
	LEAQ         cdefEdgeMask<>(SB), R14
	VPBROADCASTQ (R14)(R13*8), Z8
	MOVLQSX      ZPRI, R14
	TESTQ        R14, R14
	JZ           zmaskseconly
	MOVLQSX      ZDIRPRI, R13
	LEAQ         cdefDirs<>(SB), R14
	VPADDD.BCST  (R14)(R13*1), Z3, Z2
	VPSHUFBITQMB Z2, Z8, K1
	VMOVDQU8     Z6, Z1
	VPERMB       Z5, Z2, K1, Z1
	ZPRIMAIN
	MOVLQSX      ZSEC, R13
	TESTQ        R13, R13
	JZ           zendnoclip
	ZMASKSEC
	ZSECMAIN
	JMP          zendclip

zmaskseconly:
	ZMASKSEC
	ZSECMAIN
	JMP  zendnoclip

zendclip:
	VPMINUB  Z1, Z6, Z4
	VPMAXUB  Z6, Z1, Z1
	VPMINUB  Z3, Z2, Z5
	VPMAXUB  Z3, Z2, Z2
	VPMINUB  Z5, Z4, Z4
	VPMAXUB  Z1, Z2, Z2
	VPSRLDQ  $2, Z4, Z1
	VPSRLDQ  $2, Z2, Z3
	VPMINUB  Z4, Z1, Z1
	VPCMPW   $1, Z7, Z0, K1
	VPSHLDD  $8, Z0, Z6, Z6
	VPMAXUB  Z3, Z2, Z2
	VPSLLDQ  $1, Z1, Z3
	VPSUBW   Z0, Z7, Z7
	VPADDUSW Z6, Z0, Z0
	VPSUBUSW Z7, Z6, K1, Z0
	VPSLLDQ  $1, Z2, Z4
	VPMINUB  Z3, Z1, Z1
	VPMAXUB  Z4, Z2, Z2
	VPMAXUB  Z1, Z0, Z0
	VPMINUB  Z2, Z0, Z0
	JMP      zend

zendnoclip:
	VPSHLDD $8, Z0, Z6, Z6
	VPADDW  Z6, Z0, Z0

zend:
	VMOVDQU8 cdefEndPerm<>(SB), X1
	VPERMB   Z0, Z1, Z0
	VMOVD    X0, (DI)
	VPEXTRD  $1, X0, (DI)(DX*1)
	VPEXTRD  $2, X0, (DI)(DX*2)
	VPEXTRD  $3, X0, (DI)(R12*1)
	VZEROUPPER
	RET
