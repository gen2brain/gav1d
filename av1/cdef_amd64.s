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
	VMOVDQU8 cdefLut4<>(SB), Z5
	VPERMI2B Z1, Z0, Z5
	MOVLQSX  ZEDGE, BX
	CALL     cdefquad<>(SB)
	VZEROUPPER
	RET

// cdefquad filters one four by four quadrant. Z5 is its lut, BX its edges.
TEXT cdefquad<>(SB), NOSPLIT, $0
	VMOVDQU8     cdefPxIdx<>(SB), Z3
	VPBROADCASTD cdefBias<>(SB), Z0
	VPXORD       Z7, Z7, Z7
	VPERMB       Z5, Z3, Z6

	CMPQ BX, $15
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
	VPBROADCASTQ (R14)(BX*8), Z8
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
	RET

DATA cdefLut8<>+0(SB)/8, $0x2726252423222120
DATA cdefLut8<>+8(SB)/8, $0x3736353433323130
DATA cdefLut8<>+16(SB)/8, $0x2b2a292827262524
DATA cdefLut8<>+24(SB)/8, $0x3b3a393837363534
DATA cdefLut8<>+32(SB)/8, $0x0504030201000d0c
DATA cdefLut8<>+40(SB)/8, $0x1514131211100f0e
DATA cdefLut8<>+48(SB)/8, $0x0908070605040302
DATA cdefLut8<>+56(SB)/8, $0x1918171615141312
DATA cdefLut8<>+64(SB)/8, $0x2524232221201d1c
DATA cdefLut8<>+72(SB)/8, $0x3534333231301f1e
DATA cdefLut8<>+80(SB)/8, $0x2928272625242322
DATA cdefLut8<>+88(SB)/8, $0x3938373635343332
GLOBL cdefLut8<>(SB), RODATA|NOPTR, $96

// func cdef8x8AVX512(dst *uint8, stride int, left *uint8, top *uint8, bot *uint8, p *cdefZParams)
TEXT ·cdef8x8AVX512(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), DX
	MOVQ left+16(FP), R8
	MOVQ top+24(FP), R9
	MOVQ bot+32(FP), R10
	MOVQ p+40(FP), R11
	LEAQ (DX)(DX*2), R12
	LEAQ (DI)(DX*4), R15

	VMOVDQU32    (DI), X13
	VPINSRD      $3, (R8), X13, X13
	VINSERTI128  $1, (DI)(DX*1), Y13, Y13
	VMOVDQU32    (DI)(DX*2), X14
	VINSERTI32X4 $2, -2(R9), Z13, Z13
	VPINSRD      $3, 4(R8), X14, X14
	VINSERTI32X4 $3, -2(R9)(DX*1), Z13, Z13
	VINSERTI128  $1, (DI)(R12*1), Y14, Y14
	VPBROADCASTD 8(R8), Y15
	VPBLENDD     $0x80, Y15, Y14, Y14
	VMOVDQU32    (R15)(DX*2), X15
	VINSERTI32X4 $2, (R15), Z14, Z14
	VPINSRD      $3, 12(R8), X15, X15
	VINSERTI32X4 $3, (R15)(DX*1), Z14, Z14
	VINSERTI128  $1, (R15)(R12*1), Y15, Y15
	MOVLQSX      ZEDGE, BX
	TESTQ        $8, BX
	JZ           z8main
	VINSERTI32X4 $2, -2(R10), Z15, Z15
	VINSERTI32X4 $3, -2(R10)(DX*1), Z15, Z15

z8main:
	VMOVDQU8   cdefLut8<>(SB), Z22
	VMOVDQU8   cdefLut8<>+32(SB), Z23
	VPERMB     Z13, Z22, Z13
	VPERMB     Z14, Z23, Z14
	VPERMB     Z15, Z22, Z15
	VSHUFI32X4 $0x88, Z14, Z13, Z28
	VSHUFI32X4 $0xDD, Z14, Z13, Z29
	VSHUFI32X4 $0x28, Z15, Z14, Z30
	VSHUFI32X4 $0x7D, Z15, Z14, Z31

	MOVLQSX ZEDGE, R15
	VMOVDQU8 Z28, Z5
	MOVQ    R15, BX
	ORQ     $0xA, BX
	CALL    cdefquad<>(SB)

	ADDQ    $4, DI
	VMOVDQU8 Z29, Z5
	MOVQ    R15, BX
	ORQ     $0x9, BX
	CALL    cdefquad<>(SB)

	LEAQ    (DI)(DX*4), DI
	SUBQ    $4, DI
	VMOVDQU8 Z30, Z5
	MOVQ    R15, BX
	ORQ     $0x6, BX
	CALL    cdefquad<>(SB)

	ADDQ    $4, DI
	VMOVDQU8 Z31, Z5
	MOVQ    R15, BX
	ORQ     $0x5, BX
	CALL    cdefquad<>(SB)

	VZEROUPPER
	RET

// The sixteen bit neighbourhood of a four by four block is 64 words, so it
// takes two registers and every tap is a VPERMI2W across the pair. VPSRLVW
// shifts words directly, so nothing here needs GFNI.
DATA cdefPerm16<>+0(SB)/8, $0x1300131812101202
DATA cdefPerm16<>+8(SB)/8, $0x1502151a14011419
DATA cdefPerm16<>+16(SB)/8, $0x1b041b1c1a031a03
DATA cdefPerm16<>+24(SB)/8, $0x1dff1d1e1cff1c1d
DATA cdefPerm16<>+32(SB)/8, $0x2308231022112200
DATA cdefPerm16<>+40(SB)/8, $0x250a251224092411
DATA cdefPerm16<>+48(SB)/8, $0x2b0c2b142a0b2a01
DATA cdefPerm16<>+56(SB)/8, $0x2dff2d162cff2c15
GLOBL cdefPerm16<>(SB), RODATA|NOPTR, $64

DATA cdefEndPerm4<>+0(SB)/8, $0x0e0d0a0906050201
DATA cdefEndPerm4<>+8(SB)/8, $0x1e1d1a1916151211
DATA cdefEndPerm4<>+16(SB)/8, $0x2e2d2a2926252221
DATA cdefEndPerm4<>+24(SB)/8, $0x3e3d3a3936353231
GLOBL cdefEndPerm4<>(SB), RODATA|NOPTR, $32

DATA cdefEdgeMask16<>+0(SB)/8, $0xff00ff11ff88ff99
DATA cdefEdgeMask16<>+8(SB)/8, $0x00ff11ff88ff99ff
DATA cdefEdgeMask16<>+16(SB)/8, $0x0000111188889999
GLOBL cdefEdgeMask16<>(SB), RODATA|NOPTR, $24

DATA cdefPriTap16<>+0(SB)/8, $0x0030003000200040
GLOBL cdefPriTap16<>(SB), RODATA|NOPTR, $8

DATA cdefDirs16<>+0(SB)/8, $0x000f000800100008
DATA cdefDirs16<>+8(SB)/8, $0xfffa0001fff2fff9
DATA cdefDirs16<>+16(SB)/8, $0x000a000100020001
DATA cdefDirs16<>+24(SB)/8, $0x0011000800120009
DATA cdefDirs16<>+32(SB)/8, $0x000f000800100008
DATA cdefDirs16<>+40(SB)/8, $0xfffa0001fff2fff9
GLOBL cdefDirs16<>(SB), RODATA|NOPTR, $48

DATA cdefSecTap16<>+0(SB)/8, $0x0010002000100020
GLOBL cdefSecTap16<>(SB), RODATA|NOPTR, $8

DATA cdefBias16<>+0(SB)/4, $268435568
GLOBL cdefBias16<>(SB), RODATA|NOPTR, $4

DATA cdefEdgeFill16<>+0(SB)/4, $0xc000c000
GLOBL cdefEdgeFill16<>(SB), RODATA|NOPTR, $4

#define CONSTRAIN16(P)              \
	VPSUBW   Z3, P, Z10;        \
	VPABSW   Z10, Z10;          \
	VPCMPGTW P, Z3, K1;         \
	VPSRLVW  Z14, Z10, Z11;     \
	VPSUBUSW Z11, Z13, Z11;     \
	VPMINSW  Z11, Z10, Z10;     \
	VPSUBW   Z10, Z12, K1, Z10; \
	VPDPWSSD Z15, Z10, Z0

#define TAPPAIR16            \
	VPADDW   Z9, Z5, Z8; \
	VPERMI2W Z2, Z1, Z8; \
	VPSUBW   Z9, Z5, Z9; \
	VPERMI2W Z2, Z1, Z9; \
	CONSTRAIN16(Z8);     \
	CONSTRAIN16(Z9)

#define CLIPTAPS16           \
	VPMINUW Z8, Z6, Z6;  \
	VPMAXSW Z8, Z7, Z7;  \
	VPMINUW Z9, Z6, Z6;  \
	VPMAXSW Z9, Z7, Z7

#define SECPARAMS16                       \
	VPBROADCASTW R13, Z13;            \
	MOVLQSX      ZSECSHR, R13;        \
	VPBROADCASTW R13, Z14;            \
	VPBROADCASTD cdefSecTap16<>(SB), Z15; \
	LEAQ         cdefDirs16<>(SB), R14

// func cdef4x4_16AVX512(dst *uint16, stride int, left *uint16, top *uint16, bot *uint16, p *cdefZParams)
TEXT ·cdef4x4_16AVX512(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), DX
	MOVQ left+16(FP), R8
	MOVQ top+24(FP), R9
	MOVQ bot+32(FP), R10
	MOVQ p+40(FP), R11
	SHLQ $1, DX

	VMOVDQU      (DI), X3
	VINSERTI32X4 $1, (DI)(DX*1), Y3, Y3
	VMOVDQU      (R8), X2
	LEAQ         (DI)(DX*2), BX
	VINSERTI32X4 $2, (BX), Z3, Z3
	VMOVDQU32    cdefPerm16<>(SB), Z5
	VINSERTI32X4 $3, (BX)(DX*1), Z3, Z3
	VPERMT2D     Z3, Z5, Z2
	VINSERTI32X4 $0, -4(R9), Z2, Z1
	VINSERTI32X4 $1, -4(R9)(DX*1), Z1, Z1
	MOVLQSX      ZEDGE, AX
	VPUNPCKLWD   Z3, Z3, Z3
	VPSRLW       $8, Z5, Z5
	VPBROADCASTD cdefBias16<>(SB), Z0
	VPXORD       Z12, Z12, Z12
	CMPQ         AX, $15
	JNE          zw4maskedges
	VINSERTI32X4 $2, -4(R10), Z2, Z2
	VINSERTI32X4 $3, -4(R10)(DX*1), Z2, Z2

zw4main:
	MOVLQSX ZPRI, R13
	TESTQ   R13, R13
	JZ      zw4seconly
	VPBROADCASTW R13, Z13
	MOVLQSX      ZPRISHR, R13
	VPBROADCASTW R13, Z14
	MOVLQSX      ZPRITAP, R13
	LEAQ         cdefPriTap16<>(SB), R14
	VPBROADCASTD (R14)(R13*1), Z15
	MOVLQSX      ZDIRPRI, R13
	LEAQ         cdefDirs16<>(SB), R14
	VPBROADCASTD (R14)(R13*1), Z9
	TAPPAIR16
	MOVLQSX ZSEC, R13
	TESTQ   R13, R13
	JZ      zw4endnoclip
	VPMINUW Z8, Z3, Z6
	VPMAXSW Z8, Z3, Z7
	VPMINUW Z9, Z6, Z6
	VPMAXSW Z9, Z7, Z7
	SECPARAMS16
	MOVLQSX      ZDIRSEC0, R13
	VPBROADCASTD (R14)(R13*1), Z9
	TAPPAIR16
	CLIPTAPS16
	MOVLQSX      ZDIRSEC1, R13
	LEAQ         cdefDirs16<>(SB), R14
	VPBROADCASTD (R14)(R13*1), Z9
	TAPPAIR16
	CLIPTAPS16
	JMP     zw4endclip

zw4seconly:
	MOVLQSX ZSEC, R13
	SECPARAMS16
	MOVLQSX      ZDIRSEC0, R13
	VPBROADCASTD (R14)(R13*1), Z9
	TAPPAIR16
	MOVLQSX      ZDIRSEC1, R13
	LEAQ         cdefDirs16<>(SB), R14
	VPBROADCASTD (R14)(R13*1), Z9
	TAPPAIR16
	JMP     zw4endnoclip

zw4maskedges:
	VPBROADCASTD cdefEdgeFill16<>(SB), Z6
	LEAQ         cdefEdgeMask16<>(SB), R14
	TESTQ        $8, AX
	JZ           zw4nobottom
	VINSERTI32X4 $2, -4(R10), Z2, Z2
	VINSERTI32X4 $3, -4(R10)(DX*1), Z2, Z2
	KMOVW        -8(R14)(AX*2), K1
	JMP          zw4maskmain

zw4nobottom:
	KMOVW 8(R14)(AX*2), K1

zw4maskmain:
	ORQ       $4, AX
	VMOVDQU32 Z6, K1, Z1
	KMOVW     -8(R14)(AX*2), K1
	VMOVDQU32 Z6, K1, Z2
	JMP       zw4main

zw4endclip:
	VPSRLDQ $2, Z6, Z8
	VPSHLDD $8, Z0, Z3, Z3
	VPSRLDQ $2, Z7, Z9
	VPADDD  Z3, Z0, Z0
	VPMINUW Z8, Z6, Z6
	VPSRLDQ $1, Z0, Z0
	VPMAXSW Z9, Z7, Z7
	VPMAXSW Z6, Z0, Z0
	VPMINSW Z7, Z0, Z0
	VPMOVDW Z0, Y0
	JMP     zw4end

zw4endnoclip:
	VMOVDQU8 cdefEndPerm4<>(SB), Y1
	VPSHLDD  $8, Z0, Z3, Z3
	VPADDD   Z3, Z0, Z0
	VPERMB   Z0, Z1, Z0

zw4end:
	VMOVQ         X0, (DI)
	VPEXTRQ       $1, X0, (DI)(DX*1)
	VEXTRACTI32X4 $1, Y0, X0
	VMOVQ         X0, (BX)
	VPEXTRQ       $1, X0, (BX)(DX*1)
	VZEROUPPER
	RET

DATA cdefDirs8_16<>+0(SB)/8, $0xe402c4e23e204020
DATA cdefDirs8_16<>+8(SB)/8, $0x4220442224020402
DATA cdefDirs8_16<>+16(SB)/8, $0xe402c4e23e204020
GLOBL cdefDirs8_16<>(SB), RODATA|NOPTR, $24

DATA cdefPriTap8_16<>+0(SB)/8, $0x0002000200040004
DATA cdefPriTap8_16<>+8(SB)/8, $0x0003000300030003
GLOBL cdefPriTap8_16<>(SB), RODATA|NOPTR, $16

DATA cdefEdgeMask8_16<>+0(SB)/8, $0x0000010120202121
GLOBL cdefEdgeMask8_16<>(SB), RODATA|NOPTR, $8

DATA cdefRound16<>+0(SB)/4, $0x08000800
GLOBL cdefRound16<>(SB), RODATA|NOPTR, $4

#define CONSTRAINW(D, P, PX)    \
	VPSUBW   PX, P, D;      \
	VPABSW   D, D;          \
	VPCMPGTW P, PX, K1;     \
	VPSRLVW  Z14, D, Z15;   \
	VPSUBUSW Z15, Z13, Z15; \
	VPMINSW  Z15, D, D;     \
	VPSUBW   D, Z12, K1, D

// TAPS8 gathers one tap and its opposite for all eight rows, constrains both
// and leaves the sum in Z8 and Z9. R13 is the byte offset of the tap.
#define TAPS8                                     \
	VMOVDQU32  (R12)(R13*1), Z4;              \
	VSHUFI32X4 $0x88, 64(R12)(R13*1), Z4, Z4; \
	VMOVDQU32  128(R12)(R13*1), Z5;           \
	VSHUFI32X4 $0x88, 192(R12)(R13*1), Z5, Z5; \
	NEGQ       R13;                           \
	VMOVDQU32  (R12)(R13*1), Z6;              \
	VSHUFI32X4 $0x88, 64(R12)(R13*1), Z6, Z6; \
	VMOVDQU32  128(R12)(R13*1), Z7;           \
	VSHUFI32X4 $0x88, 192(R12)(R13*1), Z7, Z7; \
	CONSTRAINW(Z8, Z4, Z0);                   \
	CONSTRAINW(Z9, Z5, Z1);                   \
	CONSTRAINW(Z10, Z6, Z0);                  \
	CONSTRAINW(Z11, Z7, Z1);                  \
	VPADDW Z10, Z8, Z8;                       \
	VPADDW Z11, Z9, Z9

#define TAPOFF(FIELD, ADD)          \
	MOVLQSX FIELD, R13;         \
	SHRQ    $1, R13;            \
	LEAQ    cdefDirs8_16<>(SB), R14; \
	MOVBQSX ADD(R14)(R13*1), R13

#define MINMAX45            \
	VPMINUW Z4, Z18, Z18; \
	VPMAXSW Z4, Z19, Z19; \
	VPMINUW Z5, Z20, Z20; \
	VPMAXSW Z5, Z21, Z21

#define MINMAX67            \
	VPMINUW Z6, Z18, Z18; \
	VPMAXSW Z6, Z19, Z19; \
	VPMINUW Z7, Z20, Z20; \
	VPMAXSW Z7, Z21, Z21

#define SECPARAMSW                 \
	MOVLQSX      ZSEC, R13;    \
	VPBROADCASTW R13, Z13;     \
	MOVLQSX      ZSECSHR, R13; \
	VPBROADCASTW R13, Z14

// func cdef8x8_16AVX512(dst *uint16, stride int, left *uint16, top *uint16, bot *uint16, p *cdefZParams)
TEXT ·cdef8x8_16AVX512(SB), NOSPLIT, $448-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), DX
	MOVQ left+16(FP), R8
	MOVQ top+24(FP), R9
	MOVQ bot+32(FP), R10
	MOVQ p+40(FP), R11
	SHLQ $1, DX
	LEAQ (DX)(DX*2), BX

	VMOVDQU32    (DI), Y17
	VINSERTI32X8 $1, (DI)(DX*1), Z17, Z17
	VMOVQ        0(R8), X4
	VMOVQ        8(R8), X5
	VPSRLD       $16, cdefPerm16<>(SB), Z2
	VMOVQ        16(R8), X6
	VMOVQ        24(R8), X7
	LEAQ         (DI)(DX*4), SI
	VMOVDQU32    (DI)(DX*2), Y18
	VINSERTI32X8 $1, (DI)(BX*1), Z18, Z18
	VMOVDQU32    (SI), Y19
	VINSERTI32X8 $1, (SI)(DX*1), Z19, Z19
	VMOVDQU32    (SI)(DX*2), Y20
	VINSERTI32X8 $1, (SI)(BX*1), Z20, Z20
	VMOVDQU32    -4(R9), Y16
	VINSERTI32X8 $1, -4(R9)(DX*1), Z16, Z16
	VSHUFI32X4   $0x88, Z18, Z17, Z0
	MOVLQSX      ZEDGE, AX
	VSHUFI32X4   $0x88, Z20, Z19, Z1
	VPERMT2D     Z4, Z2, Z17
	VPERMT2D     Z5, Z2, Z18
	VPXORD       Z12, Z12, Z12
	VPERMT2D     Z6, Z2, Z19
	VPERMT2D     Z7, Z2, Z20
	CMPQ         AX, $15
	JNE          zw8maskedges
	VMOVDQU32    -4(R10), Y21
	VINSERTI32X8 $1, -4(R10)(DX*1), Z21, Z21

zw8main:
	VMOVDQU32 Z16, 0(SP)
	VMOVDQU32 Z17, 64(SP)
	VMOVDQU32 Z18, 128(SP)
	VMOVDQU32 Z19, 192(SP)
	VMOVDQU32 Z20, 256(SP)
	VMOVDQU32 Z21, 320(SP)
	LEAQ      68(SP), R12

	MOVLQSX ZPRI, R13
	TESTQ   R13, R13
	JZ      zw8seconly
	VPBROADCASTW R13, Z13
	MOVLQSX      ZPRISHR, R13
	VPBROADCASTW R13, Z14
	MOVLQSX      ZPRITAP, R13
	LEAQ         cdefPriTap8_16<>(SB), R14
	VPBROADCASTD (R14)(R13*2), Z2
	VPBROADCASTD 4(R14)(R13*2), Z3
	TAPOFF(ZDIRPRI, 0)
	TAPS8
	VPMULLW Z2, Z8, Z16
	VPMULLW Z2, Z9, Z17
	MOVLQSX ZSEC, R13
	TESTQ   R13, R13
	JNZ     zw8prisec
	TAPOFF(ZDIRPRI, 1)
	TAPS8
	VPMULLW Z3, Z8, Z8
	VPMULLW Z3, Z9, Z9
	JMP     zw8endnoclip

zw8prisec:
	VPMINUW Z4, Z0, Z18
	VPMAXSW Z4, Z0, Z19
	VPMINUW Z5, Z1, Z20
	VPMAXSW Z5, Z1, Z21
	MINMAX67
	TAPOFF(ZDIRPRI, 1)
	TAPS8
	VPMULLW Z3, Z8, Z8
	VPMULLW Z3, Z9, Z9
	VPADDW  Z8, Z16, Z16
	VPADDW  Z9, Z17, Z17
	SECPARAMSW
	MINMAX45
	MINMAX67
	TAPOFF(ZDIRSEC1, 0)
	TAPS8
	VMOVDQU32 Z8, Z2
	VMOVDQU32 Z9, Z3
	MINMAX45
	MINMAX67
	TAPOFF(ZDIRSEC0, 0)
	TAPS8
	VPADDW Z8, Z2, Z2
	VPADDW Z9, Z3, Z3
	MINMAX45
	MINMAX67
	TAPOFF(ZDIRSEC1, 1)
	TAPS8
	VPADDW Z2, Z2, Z2
	VPADDW Z3, Z3, Z3
	VPADDW Z8, Z16, Z16
	VPADDW Z9, Z17, Z17
	MINMAX45
	MINMAX67
	TAPOFF(ZDIRSEC0, 1)
	TAPS8
	VPBROADCASTD cdefRound16<>(SB), Z10
	VPADDW  Z2, Z16, Z16
	VPADDW  Z3, Z17, Z17
	VPADDW  Z8, Z16, Z16
	VPADDW  Z9, Z17, Z17
	VPSRAW  $15, Z16, Z8
	VPSRAW  $15, Z17, Z9
	VPADDW  Z8, Z16, Z16
	VPADDW  Z9, Z17, Z17
	VPMULHRSW Z10, Z16, Z16
	VPMULHRSW Z10, Z17, Z17
	VPMINUW Z4, Z18, Z18
	VPMAXSW Z4, Z19, Z19
	VPMINUW Z5, Z20, Z20
	VPMAXSW Z5, Z21, Z21
	VPMINUW Z6, Z18, Z18
	VPMAXSW Z6, Z19, Z19
	VPMINUW Z7, Z20, Z20
	VPMAXSW Z7, Z21, Z21
	VPADDW  Z0, Z16, Z16
	VPADDW  Z1, Z17, Z17
	VPMAXSW Z18, Z16, Z16
	VPMAXSW Z20, Z17, Z17
	VPMINSW Z19, Z16, Z16
	VPMINSW Z21, Z17, Z17
	JMP     zw8end

zw8seconly:
	SECPARAMSW
	TAPOFF(ZDIRSEC1, 0)
	TAPS8
	VMOVDQU32 Z8, Z16
	VMOVDQU32 Z9, Z17
	TAPOFF(ZDIRSEC0, 0)
	TAPS8
	VPADDW Z8, Z16, Z16
	VPADDW Z9, Z17, Z17
	TAPOFF(ZDIRSEC1, 1)
	TAPS8
	VPADDW Z16, Z16, Z16
	VPADDW Z17, Z17, Z17
	VPADDW Z8, Z16, Z16
	VPADDW Z9, Z17, Z17
	TAPOFF(ZDIRSEC0, 1)
	TAPS8

zw8endnoclip:
	VPBROADCASTD cdefRound16<>(SB), Z10
	VPADDW    Z8, Z16, Z16
	VPADDW    Z9, Z17, Z17
	VPSRAW    $15, Z16, Z8
	VPSRAW    $15, Z17, Z9
	VPADDW    Z8, Z16, Z16
	VPADDW    Z9, Z17, Z17
	VPMULHRSW Z10, Z16, Z16
	VPMULHRSW Z10, Z17, Z17
	VPADDW    Z0, Z16, Z16
	VPADDW    Z1, Z17, Z17
	JMP       zw8end

zw8maskedges:
	VPBROADCASTD cdefEdgeFill16<>(SB), Z2
	TESTQ        $8, AX
	JZ           zw8nobottom
	VMOVDQU32    -4(R10), Y21
	VINSERTI32X8 $1, -4(R10)(DX*1), Z21, Z21
	JMP          zw8masktop

zw8nobottom:
	VMOVDQU32 Z2, Z21

zw8masktop:
	TESTQ     $4, AX
	JNZ       zw8maskmain
	VMOVDQU32 Z2, Z16

zw8maskmain:
	ANDQ  $3, AX
	CMPQ  AX, $3
	JEQ   zw8main
	LEAQ  cdefEdgeMask8_16<>(SB), R14
	KMOVW (R14)(AX*2), K1
	VMOVDQU32 Z2, K1, Z16
	VMOVDQU32 Z2, K1, Z17
	VMOVDQU32 Z2, K1, Z18
	VMOVDQU32 Z2, K1, Z19
	VMOVDQU32 Z2, K1, Z20
	VMOVDQU32 Z2, K1, Z21
	JMP       zw8main

zw8end:
	VMOVDQU32     X16, (DI)
	VEXTRACTI32X4 $1, Z16, (DI)(DX*1)
	VEXTRACTI32X4 $2, Z16, (DI)(DX*2)
	VEXTRACTI32X4 $3, Z16, (DI)(BX*1)
	VMOVDQU32     X17, (SI)
	VEXTRACTI32X4 $1, Z17, (SI)(DX*1)
	VEXTRACTI32X4 $2, Z17, (SI)(DX*2)
	VEXTRACTI32X4 $3, Z17, (SI)(BX*1)
	VZEROUPPER
	RET
