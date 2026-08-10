//go:build amd64 && !noasm

#include "textflag.h"

// CFLBODY scales one vector of AC by alpha and folds it into the DC. The
// magnitude rounds before the sign goes back on, which is what keeps the
// result symmetric about zero.
#define CFLBODY(V)            \
	VPMULLD Y8, V, V;     \
	VPABSD  V, Y1;        \
	VPADDD  Y11, Y1, Y1;  \
	VPSRAD  $6, Y1, Y1;   \
	VPSIGND V, Y1, Y1;    \
	VPADDD  Y9, Y1, Y1;   \
	VPMAXSD Y12, Y1, Y1;  \
	VPMINSD Y10, Y1, Y1

#define CFLSETUP                  \
	MOVQ dst+0(FP), DI;       \
	MOVQ stride+8(FP), R8;    \
	MOVQ w+16(FP), BX;        \
	MOVQ h+24(FP), CX;        \
	MOVL dc+32(FP), AX;       \
	VMOVD AX, X9;             \
	VPBROADCASTD X9, Y9;      \
	MOVQ ac+40(FP), SI;       \
	MOVL alpha+48(FP), AX;    \
	VMOVD AX, X8;             \
	VPBROADCASTD X8, Y8;      \
	MOVL max+52(FP), AX;      \
	VMOVD AX, X10;            \
	VPBROADCASTD X10, Y10;    \
	MOVL $32, AX;             \
	VMOVD AX, X11;            \
	VPBROADCASTD X11, Y11;    \
	VPXOR Y12, Y12, Y12

// func cflPredict8AVX2(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict8AVX2(SB), NOSPLIT, $0-56
	CFLSETUP
	CMPQ BX, $4
	JEQ  narrow

row:
	MOVQ DI, DX
	XORQ R10, R10

col:
	VPMOVSXWD (SI)(R10*2), Y0
	CFLBODY(Y0)
	VPACKUSDW Y1, Y1, Y1
	VPERMQ    $0x08, Y1, Y1
	VPACKUSWB X1, X1, X1
	VMOVQ     X1, (DX)(R10*1)
	ADDQ      $8, R10
	CMPQ      R10, BX
	JLT       col

	ADDQ R8, DI
	LEAQ (SI)(BX*2), SI
	DECQ CX
	JNZ  row

	VZEROUPPER
	RET

// The AC scratch is a fixed 32x32, so a four wide block can be read eight at a
// time and only its four samples stored.
narrow:
	VPMOVSXWD (SI), Y0
	CFLBODY(Y0)
	VPACKUSDW Y1, Y1, Y1
	VPERMQ    $0x08, Y1, Y1
	VPACKUSWB X1, X1, X1
	VMOVD     X1, (DI)
	ADDQ      R8, DI
	ADDQ      $8, SI
	DECQ      CX
	JNZ       narrow

	VZEROUPPER
	RET

// func cflPredict16AVX2(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict16AVX2(SB), NOSPLIT, $0-56
	CFLSETUP
	SHLQ $1, R8
	CMPQ BX, $4
	JEQ  narrow16

row16:
	MOVQ DI, DX
	XORQ R10, R10

col16:
	VPMOVSXWD (SI)(R10*2), Y0
	CFLBODY(Y0)
	VPACKUSDW Y1, Y1, Y1
	VPERMQ    $0x08, Y1, Y1
	VMOVDQU   X1, (DX)(R10*2)
	ADDQ      $8, R10
	CMPQ      R10, BX
	JLT       col16

	ADDQ R8, DI
	LEAQ (SI)(BX*2), SI
	DECQ CX
	JNZ  row16

	VZEROUPPER
	RET

narrow16:
	VPMOVSXWD (SI), Y0
	CFLBODY(Y0)
	VPACKUSDW Y1, Y1, Y1
	VPERMQ    $0x08, Y1, Y1
	VMOVQ     X1, (DI)
	ADDQ      R8, DI
	ADDQ      $8, SI
	DECQ      CX
	JNZ       narrow16

	VZEROUPPER
	RET

// func cflAcNormAVX2(ac *int16, n int, log2sz int)
TEXT ·cflAcNormAVX2(SB), NOSPLIT, $0-24
	MOVQ ac+0(FP), DI
	MOVQ n+8(FP), R8
	MOVQ log2sz+16(FP), CX

	VPCMPEQW Y15, Y15, Y15
	VPABSW   Y15, Y15
	VPXOR    Y0, Y0, Y0
	MOVQ     DI, SI
	MOVQ     R8, R11

sumloop:
	VMOVDQU  (SI), Y1
	VPMADDWD Y15, Y1, Y1
	VPADDD   Y1, Y0, Y0
	ADDQ     $32, SI
	SUBQ     $16, R11
	JNZ      sumloop

	VEXTRACTI128 $1, Y0, X1
	VPADDD       X1, X0, X0
	VPSHUFD      $0x4E, X0, X1
	VPADDD       X1, X0, X0
	VPSHUFD      $0xB1, X0, X1
	VPADDD       X1, X0, X0
	VMOVD        X0, AX

	MOVQ $1, BX
	SHLQ CX, BX
	SHRQ $1, BX
	ADDL BX, AX
	SARL CX, AX

	VMOVD        AX, X2
	VPBROADCASTW X2, Y2

subloop:
	VMOVDQU (DI), Y1
	VPSUBW  Y2, Y1, Y1
	VMOVDQU Y1, (DI)
	ADDQ    $32, DI
	SUBQ    $16, R8
	JNZ     subloop

	VZEROUPPER
	RET

DATA z2pw32<>+0(SB)/4, $0x00200020
GLOBL z2pw32<>(SB), RODATA|NOPTR, $4

// func z2Top8AVX2(dst *uint8, edge *uint8, n int, frac int32)
TEXT ·z2Top8AVX2(SB), NOSPLIT, $0-28
	MOVQ dst+0(FP), DI
	MOVQ edge+8(FP), BX
	MOVQ n+16(FP), CX
	MOVL frac+24(FP), AX

	MOVL $64, DX
	SUBL AX, DX
	SHLL $8, AX
	ORL  DX, AX

	VMOVD        AX, X5
	VPBROADCASTW X5, Y5
	VPBROADCASTD z2pw32<>(SB), Y7
	XORQ         R14, R14

z2t:
	VMOVDQU     (BX)(R14*1), X0
	VMOVDQU     1(BX)(R14*1), X1
	VPUNPCKLBW  X1, X0, X2
	VPUNPCKHBW  X1, X0, X3
	VINSERTI128 $1, X3, Y2, Y2
	VPMADDUBSW  Y5, Y2, Y2
	VPADDW      Y7, Y2, Y2
	VPSRLW      $6, Y2, Y2
	VPACKUSWB   Y2, Y2, Y2
	VPERMQ      $0xD8, Y2, Y2
	VMOVDQU     X2, (DI)(R14*1)
	ADDQ        $16, R14
	CMPQ        R14, CX
	JLT         z2t

	VZEROUPPER
	RET

// func z2Top16AVX2(dst *uint16, edge *uint16, n int, frac int32)
TEXT ·z2Top16AVX2(SB), NOSPLIT, $0-28
	MOVQ dst+0(FP), DI
	MOVQ edge+8(FP), BX
	MOVQ n+16(FP), CX
	MOVL frac+24(FP), AX

	MOVL         $64, DX
	SUBL         AX, DX
	VMOVD        AX, X5
	VPBROADCASTD X5, Y5
	VMOVD        DX, X6
	VPBROADCASTD X6, Y6
	VPBROADCASTD z2pw32<>(SB), Y7
	VPSRLD       $16, Y7, Y7
	XORQ         R14, R14

z2t16:
	VPMOVZXWD (BX)(R14*2), Y0
	VPMOVZXWD 2(BX)(R14*2), Y1
	VPMULLD   Y6, Y0, Y0
	VPMULLD   Y5, Y1, Y1
	VPADDD    Y1, Y0, Y0
	VPADDD    Y7, Y0, Y0
	VPSRLD    $6, Y0, Y0
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (DI)(R14*2)
	ADDQ      $8, R14
	CMPQ      R14, CX
	JLT       z2t16

	VZEROUPPER
	RET

DATA cflOnes<>+0(SB)/8, $0x0101010101010101
GLOBL cflOnes<>(SB), RODATA|NOPTR, $8

// The subsampled luma sum is a pairwise add of adjacent bytes, which is what
// VPMADDUBSW against ones does, and the shift that follows is the one the
// layout implies: three for full chroma, two for a horizontal pair, one when
// both directions pair.
// func cflAcMain8AVX2(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)
TEXT ·cflAcMain8AVX2(SB), NOSPLIT, $0-64
	MOVQ ac+0(FP), DI
	MOVQ acStride+8(FP), R8
	MOVQ ypx+16(FP), SI
	MOVQ stride+24(FP), R9
	MOVQ n+32(FP), CX
	MOVQ rows+40(FP), DX
	MOVQ ssHor+48(FP), R10
	MOVQ ssVer+56(FP), R11

	SHLQ         $1, R8
	VPBROADCASTQ cflOnes<>(SB), Y15
	TESTQ        R10, R10
	JZ           ac444row
	TESTQ        R11, R11
	JZ           ac422row

ac420row:
	XORQ BX, BX

ac420col:
	VMOVDQU    (SI)(BX*2), Y0
	VPMADDUBSW Y15, Y0, Y0
	LEAQ       (SI)(R9*1), AX
	VMOVDQU    (AX)(BX*2), Y1
	VPMADDUBSW Y15, Y1, Y1
	VPADDW     Y1, Y0, Y0
	VPSLLW     $1, Y0, Y0
	VMOVDQU    Y0, (DI)(BX*2)
	ADDQ       $8, BX
	CMPQ       BX, CX
	JLT        ac420col

	ADDQ R8, DI
	LEAQ (SI)(R9*2), SI
	DECQ DX
	JNZ  ac420row

	VZEROUPPER
	RET

ac422row:
	XORQ BX, BX

ac422col:
	VMOVDQU    (SI)(BX*2), Y0
	VPMADDUBSW Y15, Y0, Y0
	VPSLLW     $2, Y0, Y0
	VMOVDQU    Y0, (DI)(BX*2)
	ADDQ       $8, BX
	CMPQ       BX, CX
	JLT        ac422col

	ADDQ R8, DI
	ADDQ R9, SI
	DECQ DX
	JNZ  ac422row

	VZEROUPPER
	RET

ac444row:
	XORQ BX, BX

ac444col:
	VPMOVZXBW (SI)(BX*1), Y0
	VPSLLW    $3, Y0, Y0
	VMOVDQU   Y0, (DI)(BX*2)
	ADDQ      $8, BX
	CMPQ      BX, CX
	JLT       ac444col

	ADDQ R8, DI
	ADDQ R9, SI
	DECQ DX
	JNZ  ac444row

	VZEROUPPER
	RET
