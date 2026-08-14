//go:build amd64 && !noasm

#include "textflag.h"

// residualAVX2 subtracts the prediction from the source, stores the difference
// and returns its sum of squares. The narrow shapes pack whole rows into one
// register: four rows at 4 px, two at 8 px.
TEXT ·residualAVX2(SB), NOSPLIT, $0-64
	MOVQ res+0(FP), DI
	MOVQ dst+8(FP), SI
	MOVQ stride+16(FP), R8
	MOVQ src+24(FP), DX
	MOVQ sstride+32(FP), R9
	MOVQ w+40(FP), CX
	MOVQ h+48(FP), BX
	VPXOR Y7, Y7, Y7

	CMPQ CX, $4
	JE   w4loop
	CMPQ CX, $8
	JE   w8loop
	JMP  rowloop

w4loop:
	LEAQ         (DX)(R9*2), R10
	VMOVD        (DX), X0
	VPINSRD      $1, (DX)(R9*1), X0, X0
	VPINSRD      $2, (R10), X0, X0
	VPINSRD      $3, (R10)(R9*1), X0, X0
	LEAQ         (SI)(R8*2), R11
	VMOVD        (SI), X1
	VPINSRD      $1, (SI)(R8*1), X1, X1
	VPINSRD      $2, (R11), X1, X1
	VPINSRD      $3, (R11)(R8*1), X1, X1
	VPMOVZXBW    X0, Y0
	VPMOVZXBW    X1, Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	LEAQ         (DX)(R9*4), DX
	LEAQ         (SI)(R8*4), SI
	SUBQ         $4, BX
	JNZ          w4loop
	JMP          done

w8loop:
	VMOVQ        (DX), X0
	VPINSRQ      $1, (DX)(R9*1), X0, X0
	VMOVQ        (SI), X1
	VPINSRQ      $1, (SI)(R8*1), X1, X1
	VPMOVZXBW    X0, Y0
	VPMOVZXBW    X1, Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	LEAQ         (DX)(R9*2), DX
	LEAQ         (SI)(R8*2), SI
	SUBQ         $2, BX
	JNZ          w8loop
	JMP          done

rowloop:
	MOVQ CX, R11
	MOVQ DX, R12
	MOVQ SI, R13

colloop:
	VPMOVZXBW    (R12), Y0
	VPMOVZXBW    (R13), Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	ADDQ         $16, R12
	ADDQ         $16, R13
	SUBQ         $16, R11
	JNZ          colloop

	ADDQ R9, DX
	ADDQ R8, SI
	DECQ BX
	JNZ  rowloop

done:
	VEXTRACTI128 $1, Y7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x0e, X7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x01, X7, X0
	VPADDD       X0, X7, X7
	VMOVD        X7, AX
	MOVLQZX      AX, AX
	MOVQ         AX, ret+56(FP)
	VZEROUPPER
	RET

#define HAD8(i0, i1, i2, i3, i4, i5, i6, i7, o0, o1, o2, o3, o4, o5, o6, o7) \
	VPADDD i4, i0, o0 \
	VPADDD i5, i1, o1 \
	VPADDD i6, i2, o2 \
	VPADDD i7, i3, o3 \
	VPSUBD i4, i0, o4 \
	VPSUBD i5, i1, o5 \
	VPSUBD i6, i2, o6 \
	VPSUBD i7, i3, o7 \
	VPADDD o2, o0, i0 \
	VPADDD o3, o1, i1 \
	VPSUBD o2, o0, i2 \
	VPSUBD o3, o1, i3 \
	VPADDD o6, o4, i4 \
	VPADDD o7, o5, i5 \
	VPSUBD o6, o4, i6 \
	VPSUBD o7, o5, i7 \
	VPADDD i1, i0, o0 \
	VPSUBD i1, i0, o1 \
	VPADDD i3, i2, o2 \
	VPSUBD i3, i2, o3 \
	VPADDD i5, i4, o4 \
	VPSUBD i5, i4, o5 \
	VPADDD i7, i6, o6 \
	VPSUBD i7, i6, o7

// satd8AVX2 is the sum of absolute 8x8 Hadamard coefficients. The transform is
// separable and the sum does not care about transposition, so both passes run
// down the columns with a transpose between them.
TEXT ·satd8AVX2(SB), NOSPLIT, $0-20
	MOVQ res+0(FP), SI
	MOVQ stride+8(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), Y0
	VMOVDQU (SI)(DX*1), Y1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), Y2
	VMOVDQU (AX)(DX*1), Y3
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y4
	VMOVDQU (AX)(DX*1), Y5
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y6
	VMOVDQU (AX)(DX*1), Y7

	HAD8(Y0, Y1, Y2, Y3, Y4, Y5, Y6, Y7, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15)

	VPUNPCKLDQ  Y9, Y8, Y0
	VPUNPCKHDQ  Y9, Y8, Y1
	VPUNPCKLDQ  Y11, Y10, Y2
	VPUNPCKHDQ  Y11, Y10, Y3
	VPUNPCKLDQ  Y13, Y12, Y4
	VPUNPCKHDQ  Y13, Y12, Y5
	VPUNPCKLDQ  Y15, Y14, Y6
	VPUNPCKHDQ  Y15, Y14, Y7
	VPUNPCKLQDQ Y2, Y0, Y8
	VPUNPCKHQDQ Y2, Y0, Y9
	VPUNPCKLQDQ Y3, Y1, Y10
	VPUNPCKHQDQ Y3, Y1, Y11
	VPUNPCKLQDQ Y6, Y4, Y12
	VPUNPCKHQDQ Y6, Y4, Y13
	VPUNPCKLQDQ Y7, Y5, Y14
	VPUNPCKHQDQ Y7, Y5, Y15
	VPERM2I128  $0x20, Y12, Y8, Y0
	VPERM2I128  $0x20, Y13, Y9, Y1
	VPERM2I128  $0x20, Y14, Y10, Y2
	VPERM2I128  $0x20, Y15, Y11, Y3
	VPERM2I128  $0x31, Y12, Y8, Y4
	VPERM2I128  $0x31, Y13, Y9, Y5
	VPERM2I128  $0x31, Y14, Y10, Y6
	VPERM2I128  $0x31, Y15, Y11, Y7

	HAD8(Y0, Y1, Y2, Y3, Y4, Y5, Y6, Y7, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15)

	VPABSD Y8, Y8
	VPABSD Y9, Y9
	VPABSD Y10, Y10
	VPABSD Y11, Y11
	VPABSD Y12, Y12
	VPABSD Y13, Y13
	VPABSD Y14, Y14
	VPABSD Y15, Y15
	VPADDD Y9, Y8, Y8
	VPADDD Y11, Y10, Y10
	VPADDD Y13, Y12, Y12
	VPADDD Y15, Y14, Y14
	VPADDD Y10, Y8, Y8
	VPADDD Y14, Y12, Y12
	VPADDD Y12, Y8, Y8

	VEXTRACTI128 $1, Y8, X0
	VPADDD       X0, X8, X8
	VPSHUFD      $0x0e, X8, X0
	VPADDD       X0, X8, X8
	VPSHUFD      $0x01, X8, X0
	VPADDD       X0, X8, X8
	VMOVD        X8, AX
	ADDL         $4, AX
	SARL         $3, AX
	MOVL         AX, ret+16(FP)
	VZEROUPPER
	RET

DATA fwd4Const<>+0(SB)/4, $5793
DATA fwd4Const<>+4(SB)/4, $3135
DATA fwd4Const<>+8(SB)/4, $7568
DATA fwd4Const<>+12(SB)/4, $4096
DATA fwd4Const<>+16(SB)/4, $2642
DATA fwd4Const<>+20(SB)/4, $6689
DATA fwd4Const<>+24(SB)/4, $4964
DATA fwd4Const<>+28(SB)/4, $7606
DATA fwd4Const<>+32(SB)/4, $8035
DATA fwd4Const<>+36(SB)/4, $6811
DATA fwd4Const<>+40(SB)/4, $4551
DATA fwd4Const<>+44(SB)/4, $1598
GLOBL fwd4Const<>(SB), RODATA|NOPTR, $48

#define RS(v) \
	VPADDD X11, v, v \
	VPSRAD $13, v, v

#define FDCT4(i0, i1, i2, i3, o0, o1, o2, o3) \
	VPADDD  i3, i0, X4 \
	VPADDD  i2, i1, X5 \
	VPSUBD  i2, i1, X6 \
	VPSUBD  i3, i0, X7 \
	VPADDD  X5, X4, X13 \
	VPSUBD  X5, X4, X14 \
	VPMULLD X8, X13, X13 \
	RS(X13) \
	VPMULLD X8, X14, X14 \
	RS(X14) \
	VPMULLD X9, X6, X15 \
	VPMULLD X10, X7, X4 \
	VPADDD  X4, X15, X15 \
	RS(X15) \
	VPMULLD X9, X7, X5 \
	VPMULLD X10, X6, X6 \
	VPSUBD  X6, X5, X5 \
	RS(X5) \
	VMOVDQU X13, o0 \
	VMOVDQU X15, o1 \
	VMOVDQU X14, o2 \
	VMOVDQU X5, o3

#define FADST4(i0, i1, i2, i3, o0, o1, o2, o3) \
	VPMULLD X8, i0, X4 \
	VPMULLD X10, i1, X5 \
	VPADDD  X5, X4, X4 \
	VPMULLD X12, i3, X5 \
	VPADDD  X5, X4, X4 \
	VPMULLD X12, i0, X5 \
	VPMULLD X8, i1, X6 \
	VPSUBD  X6, X5, X5 \
	VPMULLD X10, i3, X6 \
	VPADDD  X6, X5, X5 \
	VPMULLD X9, i2, X6 \
	VPADDD  i1, i0, X7 \
	VPSUBD  i3, X7, X7 \
	VPMULLD X9, X7, X7 \
	VPADDD  X6, X4, X13 \
	RS(X13) \
	RS(X7) \
	VPSUBD  X6, X5, X14 \
	RS(X14) \
	VPSUBD  X4, X5, X15 \
	VPADDD  X6, X15, X15 \
	RS(X15) \
	VMOVDQU X13, o0 \
	VMOVDQU X7, o1 \
	VMOVDQU X14, o2 \
	VMOVDQU X15, o3

#define T4(i0, i1, i2, i3, o0, o1, o2, o3) \
	VPUNPCKLDQ  i1, i0, X4 \
	VPUNPCKHDQ  i1, i0, X5 \
	VPUNPCKLDQ  i3, i2, X6 \
	VPUNPCKHDQ  i3, i2, X7 \
	VPUNPCKLQDQ X6, X4, o0 \
	VPUNPCKHQDQ X6, X4, o1 \
	VPUNPCKLQDQ X7, X5, o2 \
	VPUNPCKHQDQ X7, X5, o3

#define LOADDCT \
	VPBROADCASTD fwd4Const<>+0(SB), X8 \
	VPBROADCASTD fwd4Const<>+4(SB), X9 \
	VPBROADCASTD fwd4Const<>+8(SB), X10

#define LOADADST \
	VPBROADCASTD fwd4Const<>+16(SB), X8 \
	VPBROADCASTD fwd4Const<>+20(SB), X9 \
	VPBROADCASTD fwd4Const<>+24(SB), X10 \
	VPBROADCASTD fwd4Const<>+28(SB), X12

// fwdTxfm4x4AVX2 is the 4x4 forward transform for the four non-flipping
// DCT/ADST combinations. cos_bit is 13 at this size, so the constants fold in,
// and every intermediate fits int32 for 8-bit input.
TEXT ·fwdTxfm4x4AVX2(SB), NOSPLIT, $0-40
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), X0
	VMOVDQU (SI)(DX*1), X1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), X2
	VMOVDQU (AX)(DX*1), X3
	VPSLLD  $2, X0, X0
	VPSLLD  $2, X1, X1
	VPSLLD  $2, X2, X2
	VPSLLD  $2, X3, X3

	VPBROADCASTD fwd4Const<>+12(SB), X11

	MOVQ  colAdst+24(FP), AX
	TESTQ AX, AX
	JNZ   coladst
	LOADDCT
	FDCT4(X0, X1, X2, X3, X0, X1, X2, X3)
	JMP   colfin

coladst:
	LOADADST
	FADST4(X0, X1, X2, X3, X0, X1, X2, X3)

colfin:
	T4(X0, X1, X2, X3, X0, X1, X2, X3)

	MOVQ  rowAdst+32(FP), AX
	TESTQ AX, AX
	JNZ   rowadst
	LOADDCT
	FDCT4(X0, X1, X2, X3, X0, X1, X2, X3)
	JMP   store

rowadst:
	LOADADST
	FADST4(X0, X1, X2, X3, X0, X1, X2, X3)

store:
	VMOVDQU X0, (DI)
	VMOVDQU X1, 16(DI)
	VMOVDQU X2, 32(DI)
	VMOVDQU X3, 48(DI)
	VZEROUPPER
	RET

// sseAVX2 is the sum of squared differences, packing whole rows into one
// register at the narrow shapes the way residualAVX2 does.
TEXT ·sseAVX2(SB), NOSPLIT, $0-56
	MOVQ  dst+0(FP), SI
	MOVQ  stride+8(FP), R8
	MOVQ  src+16(FP), DX
	MOVQ  sstride+24(FP), R9
	MOVQ  w+32(FP), CX
	MOVQ  h+40(FP), BX
	VPXOR Y7, Y7, Y7

	CMPQ CX, $4
	JE   s4loop
	CMPQ CX, $8
	JE   s8loop
	JMP  srowloop

s4loop:
	LEAQ      (DX)(R9*2), R10
	VMOVD     (DX), X0
	VPINSRD   $1, (DX)(R9*1), X0, X0
	VPINSRD   $2, (R10), X0, X0
	VPINSRD   $3, (R10)(R9*1), X0, X0
	LEAQ      (SI)(R8*2), R11
	VMOVD     (SI), X1
	VPINSRD   $1, (SI)(R8*1), X1, X1
	VPINSRD   $2, (R11), X1, X1
	VPINSRD   $3, (R11)(R8*1), X1, X1
	VPMOVZXBW X0, Y0
	VPMOVZXBW X1, Y1
	VPSUBW    Y1, Y0, Y2
	VPMADDWD  Y2, Y2, Y6
	VPADDD    Y6, Y7, Y7
	LEAQ      (DX)(R9*4), DX
	LEAQ      (SI)(R8*4), SI
	SUBQ      $4, BX
	JNZ       s4loop
	JMP       sdone

s8loop:
	VMOVQ     (DX), X0
	VPINSRQ   $1, (DX)(R9*1), X0, X0
	VMOVQ     (SI), X1
	VPINSRQ   $1, (SI)(R8*1), X1, X1
	VPMOVZXBW X0, Y0
	VPMOVZXBW X1, Y1
	VPSUBW    Y1, Y0, Y2
	VPMADDWD  Y2, Y2, Y6
	VPADDD    Y6, Y7, Y7
	LEAQ      (DX)(R9*2), DX
	LEAQ      (SI)(R8*2), SI
	SUBQ      $2, BX
	JNZ       s8loop
	JMP       sdone

srowloop:
	MOVQ CX, R11
	MOVQ DX, R12
	MOVQ SI, R13

scolloop:
	VPMOVZXBW (R12), Y0
	VPMOVZXBW (R13), Y1
	VPSUBW    Y1, Y0, Y2
	VPMADDWD  Y2, Y2, Y6
	VPADDD    Y6, Y7, Y7
	ADDQ      $16, R12
	ADDQ      $16, R13
	SUBQ      $16, R11
	JNZ       scolloop

	ADDQ R9, DX
	ADDQ R8, SI
	DECQ BX
	JNZ  srowloop

sdone:
	VEXTRACTI128 $1, Y7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x0e, X7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x01, X7, X0
	VPADDD       X0, X7, X7
	VMOVD        X7, AX
	MOVLQZX      AX, AX
	MOVQ         AX, ret+48(FP)
	VZEROUPPER
	RET

#define HAD4(i0, i1, i2, i3, o0, o1, o2, o3) \
	VPADDD i2, i0, X4 \
	VPADDD i3, i1, X5 \
	VPSUBD i2, i0, X6 \
	VPSUBD i3, i1, X7 \
	VPADDD X5, X4, o0 \
	VPSUBD X5, X4, o1 \
	VPADDD X7, X6, o2 \
	VPSUBD X7, X6, o3

// satd4AVX2 is the sum of absolute 4x4 Hadamard coefficients.
TEXT ·satd4AVX2(SB), NOSPLIT, $0-20
	MOVQ res+0(FP), SI
	MOVQ stride+8(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), X0
	VMOVDQU (SI)(DX*1), X1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), X2
	VMOVDQU (AX)(DX*1), X3

	HAD4(X0, X1, X2, X3, X8, X9, X10, X11)
	T4(X8, X9, X10, X11, X0, X1, X2, X3)
	HAD4(X0, X1, X2, X3, X8, X9, X10, X11)

	VPABSD  X8, X8
	VPABSD  X9, X9
	VPABSD  X10, X10
	VPABSD  X11, X11
	VPADDD  X9, X8, X8
	VPADDD  X11, X10, X10
	VPADDD  X10, X8, X8
	VPSHUFD $0x0e, X8, X0
	VPADDD  X0, X8, X8
	VPSHUFD $0x01, X8, X0
	VPADDD  X0, X8, X8
	VMOVD   X8, AX
	ADDL    $2, AX
	SARL    $2, AX
	MOVL    AX, ret+16(FP)
	VZEROUPPER
	RET

#define RS8(v) \
	VPADDD Y15, v, v \
	VPSRAD $13, v, v

#define BTF8(wa, a, wb, b, t, dst) \
	VPMULLD wa, a, dst \
	VPMULLD wb, b, t \
	VPADDD  t, dst, dst \
	RS8(dst)

#define BTFN8(wa, a, wb, b, t, dst) \
	VPMULLD wa, a, dst \
	VPMULLD wb, b, t \
	VPSUBD  t, dst, dst \
	RS8(dst)

// FDCT8 runs an 8 point DCT down the columns of Y0..Y7 in place. Constants:
// Y8 cospi32, Y9 cospi48, Y10 cospi16, Y11 cospi56, Y12 cospi24, Y13 cospi40,
// Y14 cospi8, Y15 the rounding term. Temporaries are Y16 upward, which is why
// this one kernel needs AVX-512 rather than AVX2.
#define FDCT8 \
	VPADDD Y7, Y0, Y16 \
	VPADDD Y6, Y1, Y17 \
	VPADDD Y5, Y2, Y18 \
	VPADDD Y4, Y3, Y19 \
	VPSUBD Y4, Y3, Y20 \
	VPSUBD Y5, Y2, Y21 \
	VPSUBD Y6, Y1, Y22 \
	VPSUBD Y7, Y0, Y23 \
	VPADDD Y19, Y16, Y24 \
	VPADDD Y18, Y17, Y25 \
	VPSUBD Y18, Y17, Y26 \
	VPSUBD Y19, Y16, Y27 \
	BTFN8(Y8, Y22, Y8, Y21, Y29, Y28) \
	BTF8(Y8, Y22, Y8, Y21, Y30, Y29) \
	BTF8(Y8, Y24, Y8, Y25, Y31, Y0) \
	BTFN8(Y8, Y24, Y8, Y25, Y31, Y4) \
	BTF8(Y9, Y26, Y10, Y27, Y31, Y2) \
	BTFN8(Y9, Y27, Y10, Y26, Y31, Y6) \
	VPADDD Y28, Y20, Y24 \
	VPSUBD Y28, Y20, Y25 \
	VPSUBD Y29, Y23, Y26 \
	VPADDD Y29, Y23, Y27 \
	BTF8(Y11, Y24, Y14, Y27, Y31, Y1) \
	BTF8(Y12, Y25, Y13, Y26, Y31, Y5) \
	BTFN8(Y12, Y26, Y13, Y25, Y31, Y3) \
	BTFN8(Y11, Y27, Y14, Y24, Y31, Y7)

#define LOADC8 \
	VPBROADCASTD fwd4Const<>+0(SB), Y8 \
	VPBROADCASTD fwd4Const<>+4(SB), Y9 \
	VPBROADCASTD fwd4Const<>+8(SB), Y10 \
	VPBROADCASTD fwd4Const<>+44(SB), Y11 \
	VPBROADCASTD fwd4Const<>+36(SB), Y12 \
	VPBROADCASTD fwd4Const<>+40(SB), Y13 \
	VPBROADCASTD fwd4Const<>+32(SB), Y14 \
	VPBROADCASTD fwd4Const<>+12(SB), Y15

// fwdTxfm8x8AVX512 is the 8x8 DCT_DCT forward transform.
TEXT ·fwdTxfm8x8AVX512(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), Y0
	VMOVDQU (SI)(DX*1), Y1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), Y2
	VMOVDQU (AX)(DX*1), Y3
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y4
	VMOVDQU (AX)(DX*1), Y5
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y6
	VMOVDQU (AX)(DX*1), Y7
	VPSLLD  $2, Y0, Y0
	VPSLLD  $2, Y1, Y1
	VPSLLD  $2, Y2, Y2
	VPSLLD  $2, Y3, Y3
	VPSLLD  $2, Y4, Y4
	VPSLLD  $2, Y5, Y5
	VPSLLD  $2, Y6, Y6
	VPSLLD  $2, Y7, Y7

	LOADC8
	FDCT8

	VPUNPCKLDQ  Y1, Y0, Y16
	VPUNPCKHDQ  Y1, Y0, Y17
	VPUNPCKLDQ  Y3, Y2, Y18
	VPUNPCKHDQ  Y3, Y2, Y19
	VPUNPCKLDQ  Y5, Y4, Y20
	VPUNPCKHDQ  Y5, Y4, Y21
	VPUNPCKLDQ  Y7, Y6, Y22
	VPUNPCKHDQ  Y7, Y6, Y23
	VPUNPCKLQDQ Y18, Y16, Y24
	VPUNPCKHQDQ Y18, Y16, Y25
	VPUNPCKLQDQ Y19, Y17, Y26
	VPUNPCKHQDQ Y19, Y17, Y27
	VPUNPCKLQDQ Y22, Y20, Y28
	VPUNPCKHQDQ Y22, Y20, Y29
	VPUNPCKLQDQ Y23, Y21, Y30
	VPUNPCKHQDQ Y23, Y21, Y31
	VSHUFI32X4  $0x00, Y28, Y24, Y0
	VSHUFI32X4  $0x00, Y29, Y25, Y1
	VSHUFI32X4  $0x00, Y30, Y26, Y2
	VSHUFI32X4  $0x00, Y31, Y27, Y3
	VSHUFI32X4  $0x03, Y28, Y24, Y4
	VSHUFI32X4  $0x03, Y29, Y25, Y5
	VSHUFI32X4  $0x03, Y30, Y26, Y6
	VSHUFI32X4  $0x03, Y31, Y27, Y7

	FDCT8

	MOVL    $1, AX
	VMOVD   AX, X16
	VPBROADCASTD X16, Y16
	VPADDD  Y16, Y0, Y0
	VPADDD  Y16, Y1, Y1
	VPADDD  Y16, Y2, Y2
	VPADDD  Y16, Y3, Y3
	VPADDD  Y16, Y4, Y4
	VPADDD  Y16, Y5, Y5
	VPADDD  Y16, Y6, Y6
	VPADDD  Y16, Y7, Y7
	VPSRAD  $1, Y0, Y0
	VPSRAD  $1, Y1, Y1
	VPSRAD  $1, Y2, Y2
	VPSRAD  $1, Y3, Y3
	VPSRAD  $1, Y4, Y4
	VPSRAD  $1, Y5, Y5
	VPSRAD  $1, Y6, Y6
	VPSRAD  $1, Y7, Y7
	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	VMOVDQU Y2, 64(DI)
	VMOVDQU Y3, 96(DI)
	VMOVDQU Y4, 128(DI)
	VMOVDQU Y5, 160(DI)
	VMOVDQU Y6, 192(DI)
	VMOVDQU Y7, 224(DI)
	VZEROUPPER
	RET

#define T4R(i0, i1, i2, i3, t0, t1, t2, t3, o0, o1, o2, o3) \
	VPUNPCKLDQ  i1, i0, t0 \
	VPUNPCKHDQ  i1, i0, t1 \
	VPUNPCKLDQ  i3, i2, t2 \
	VPUNPCKHDQ  i3, i2, t3 \
	VPUNPCKLQDQ t2, t0, o0 \
	VPUNPCKHQDQ t2, t0, o1 \
	VPUNPCKLQDQ t3, t1, o2 \
	VPUNPCKHQDQ t3, t1, o3

// FDCT4R runs a 4 point DCT across four registers, taking its temporaries as
// parameters so the caller can keep the inputs live. Constants are the same
// ones LOADC8 sets: Y8 cospi32, Y9 cospi48, Y10 cospi16, Y15 the rounding term.
#define FDCT4R(i0, i1, i2, i3, a0, a1, a2, a3, t0, t1, o0, o1, o2, o3) \
	VPADDD    i3, i0, a0 \
	VPADDD    i2, i1, a1 \
	VPSUBD    i2, i1, a2 \
	VPSUBD    i3, i0, a3 \
	VPADDD    a1, a0, t0 \
	VPSUBD    a1, a0, t1 \
	VPMULLD   Y8, t0, t0 \
	RS8(t0) \
	VPMULLD   Y8, t1, t1 \
	RS8(t1) \
	BTF8(Y9, a2, Y10, a3, a0, o1) \
	BTFN8(Y9, a3, Y10, a2, a1, o3) \
	VMOVDQU32 t0, o0 \
	VMOVDQU32 t1, o2

// RECT scales one register by the rectangular Sqrt2 correction after the row
// pass: round shift by one, then newSqrt2 (which is cospi32) and 12 bits. Y14
// holds one and Y13 holds 2048; both are reloaded after the last butterfly.
#define RECT(v) \
	VPADDD  Y14, v, v \
	VPSRAD  $1, v, v \
	VPMULLD Y8, v, v \
	VPADDD  Y13, v, v \
	VPSRAD  $12, v, v

#define LOADRECT \
	MOVL         $1, AX \
	VMOVD        AX, X14 \
	VPBROADCASTD X14, Y14 \
	VPSRLD       $1, Y15, Y13

// fwdTxfm4x8AVX512 is the 4x8 DCT_DCT forward transform. The column pass is
// eight points over four lanes, so half of each register is idle; a 256 bit
// operation costs the same as a 128 bit one, so this is free.
TEXT ·fwdTxfm4x8AVX512(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), X0
	VMOVDQU (SI)(DX*1), X1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), X2
	VMOVDQU (AX)(DX*1), X3
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), X4
	VMOVDQU (AX)(DX*1), X5
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), X6
	VMOVDQU (AX)(DX*1), X7
	VPSLLD  $2, X0, X0
	VPSLLD  $2, X1, X1
	VPSLLD  $2, X2, X2
	VPSLLD  $2, X3, X3
	VPSLLD  $2, X4, X4
	VPSLLD  $2, X5, X5
	VPSLLD  $2, X6, X6
	VPSLLD  $2, X7, X7

	LOADC8
	FDCT8

	T4R(X0, X1, X2, X3, X24, X25, X26, X27, X16, X17, X18, X19)
	T4R(X4, X5, X6, X7, X24, X25, X26, X27, X20, X21, X22, X23)
	VINSERTI32X4 $1, X20, Y16, Y0
	VINSERTI32X4 $1, X21, Y17, Y1
	VINSERTI32X4 $1, X22, Y18, Y2
	VINSERTI32X4 $1, X23, Y19, Y3

	FDCT4R(Y0, Y1, Y2, Y3, Y16, Y17, Y18, Y19, Y20, Y21, Y0, Y1, Y2, Y3)

	LOADRECT
	RECT(Y0)
	RECT(Y1)
	RECT(Y2)
	RECT(Y3)

	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	VMOVDQU Y2, 64(DI)
	VMOVDQU Y3, 96(DI)
	VZEROUPPER
	RET

// fwdTxfm8x4AVX512 is the 8x4 DCT_DCT forward transform, the same passes as
// fwdTxfm4x8AVX512 in the other order.
TEXT ·fwdTxfm8x4AVX512(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), Y0
	VMOVDQU (SI)(DX*1), Y1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), Y2
	VMOVDQU (AX)(DX*1), Y3
	VPSLLD  $2, Y0, Y0
	VPSLLD  $2, Y1, Y1
	VPSLLD  $2, Y2, Y2
	VPSLLD  $2, Y3, Y3

	LOADC8
	FDCT4R(Y0, Y1, Y2, Y3, Y16, Y17, Y18, Y19, Y20, Y21, Y0, Y1, Y2, Y3)

	VEXTRACTI32X4 $1, Y0, X4
	VEXTRACTI32X4 $1, Y1, X5
	VEXTRACTI32X4 $1, Y2, X6
	VEXTRACTI32X4 $1, Y3, X7
	T4R(X0, X1, X2, X3, X16, X17, X18, X19, X0, X1, X2, X3)
	T4R(X4, X5, X6, X7, X16, X17, X18, X19, X4, X5, X6, X7)

	FDCT8

	LOADRECT
	RECT(Y0)
	RECT(Y1)
	RECT(Y2)
	RECT(Y3)
	RECT(Y4)
	RECT(Y5)
	RECT(Y6)
	RECT(Y7)

	VMOVDQU X0, (DI)
	VMOVDQU X1, 16(DI)
	VMOVDQU X2, 32(DI)
	VMOVDQU X3, 48(DI)
	VMOVDQU X4, 64(DI)
	VMOVDQU X5, 80(DI)
	VMOVDQU X6, 96(DI)
	VMOVDQU X7, 112(DI)
	VZEROUPPER
	RET

DATA fwd16Const<>+0(SB)/4, $5793
DATA fwd16Const<>+4(SB)/4, $3135
DATA fwd16Const<>+8(SB)/4, $7568
DATA fwd16Const<>+12(SB)/4, $1598
DATA fwd16Const<>+16(SB)/4, $8035
DATA fwd16Const<>+20(SB)/4, $6811
DATA fwd16Const<>+24(SB)/4, $4551
DATA fwd16Const<>+28(SB)/4, $803
DATA fwd16Const<>+32(SB)/4, $8153
DATA fwd16Const<>+36(SB)/4, $6333
DATA fwd16Const<>+40(SB)/4, $5197
DATA fwd16Const<>+44(SB)/4, $3862
DATA fwd16Const<>+48(SB)/4, $7225
DATA fwd16Const<>+52(SB)/4, $7839
DATA fwd16Const<>+56(SB)/4, $2378
DATA fwd16Const<>+60(SB)/4, $4096
DATA fwd16Const<>+64(SB)/4, $2
DATA fwd16Const<>+68(SB)/4, $2048
GLOBL fwd16Const<>(SB), RODATA|NOPTR, $80

#define K32  fwd16Const<>+0(SB)
#define K48  fwd16Const<>+4(SB)
#define K16  fwd16Const<>+8(SB)
#define K56  fwd16Const<>+12(SB)
#define K8   fwd16Const<>+16(SB)
#define K24  fwd16Const<>+20(SB)
#define K40  fwd16Const<>+24(SB)
#define K60  fwd16Const<>+28(SB)
#define K4   fwd16Const<>+32(SB)
#define K28  fwd16Const<>+36(SB)
#define K36  fwd16Const<>+40(SB)
#define K44  fwd16Const<>+44(SB)
#define K20  fwd16Const<>+48(SB)
#define K12  fwd16Const<>+52(SB)
#define K52  fwd16Const<>+56(SB)
#define KRND fwd16Const<>+60(SB)
#define KTWO fwd16Const<>+64(SB)
#define KSQR fwd16Const<>+68(SB)

// RSB rounds a product down by cos_bit, reading the rounding term as an
// embedded broadcast so no register has to hold it. Keeping all sixteen
// constants in memory is what leaves room for sixteen live rows.
#define RSB(v) \
	VPADDD.BCST KRND, v, v \
	VPSRAD      $13, v, v

#define MULK(k, a, dst) \
	VPMULLD.BCST k, a, dst

#define BTFK(ka, a, kb, b, t, dst) \
	MULK(ka, a, dst) \
	MULK(kb, b, t) \
	VPADDD t, dst, dst \
	RSB(dst)

#define BTFNK(ka, a, kb, b, t, dst) \
	MULK(ka, a, dst) \
	MULK(kb, b, t) \
	VPSUBD t, dst, dst \
	RSB(dst)

// BFLY is the in-place pairwise butterfly the 16 point transform is built from:
// hi becomes lo-hi and lo becomes lo+hi, with one temporary.
#define BFLY(lo, hi, t) \
	VPSUBD    hi, lo, t \
	VPADDD    hi, lo, lo \
	VMOVDQU32 t, hi

// HALF32 replaces a pair by rs(cospi32*(lo-hi)) and rs(cospi32*(lo+hi)).
#define HALF32(a, b, dlo, dhi, t0, t1) \
	VPSUBD  a, b, t0 \
	VPADDD  a, b, t1 \
	MULK(K32, t0, dlo) \
	RSB(dlo) \
	MULK(K32, t1, dhi) \
	RSB(dhi)

// FDCT16 runs a 16 point DCT across Y0..Y15 in place, with Y16..Y24 as
// temporaries. It leaves the coefficients permuted: frequency k is in the
// register given by fdct16Perm, which costs nothing because the transpose and
// the store both read the registers in whatever order they are told.
#define FDCT16 \
	BFLY(Y0, Y15, Y16) \
	BFLY(Y1, Y14, Y16) \
	BFLY(Y2, Y13, Y16) \
	BFLY(Y3, Y12, Y16) \
	BFLY(Y4, Y11, Y16) \
	BFLY(Y5, Y10, Y16) \
	BFLY(Y6, Y9, Y16) \
	BFLY(Y7, Y8, Y16) \
	\
	BFLY(Y0, Y7, Y16) \
	BFLY(Y1, Y6, Y16) \
	BFLY(Y2, Y5, Y16) \
	BFLY(Y3, Y4, Y16) \
	HALF32(Y10, Y13, Y10, Y13, Y16, Y17) \
	HALF32(Y11, Y12, Y11, Y12, Y16, Y17) \
	\
	BFLY(Y0, Y3, Y16) \
	BFLY(Y1, Y2, Y16) \
	HALF32(Y5, Y6, Y5, Y6, Y16, Y17) \
	BFLY(Y8, Y11, Y16) \
	BFLY(Y9, Y10, Y16) \
	BFLY(Y15, Y12, Y16) \
	BFLY(Y14, Y13, Y16) \
	\
	HALF32(Y1, Y0, Y1, Y0, Y16, Y17) \
	BTFK(K48, Y2, K16, Y3, Y18, Y16) \
	BTFNK(K48, Y3, K16, Y2, Y19, Y17) \
	VMOVDQU32 Y16, Y2 \
	VMOVDQU32 Y17, Y3 \
	BFLY(Y4, Y5, Y16) \
	BFLY(Y7, Y6, Y16) \
	BTFNK(K48, Y14, K16, Y9, Y18, Y16) \
	BTFK(K16, Y14, K48, Y9, Y19, Y17) \
	VMOVDQU32 Y16, Y9 \
	VMOVDQU32 Y17, Y14 \
	BTFNK(K48, Y13, K16, Y10, Y18, Y16) \
	MULK(K48, Y10, Y17) \
	MULK(K16, Y13, Y18) \
	VPADDD Y18, Y17, Y17 \
	VPBROADCASTD KRND, Y19 \
	VPSUBD Y17, Y19, Y17 \
	VPSRAD $13, Y17, Y17 \
	VMOVDQU32 Y16, Y13 \
	VMOVDQU32 Y17, Y10 \
	\
	BTFK(K56, Y4, K8, Y7, Y20, Y16) \
	BTFK(K24, Y5, K40, Y6, Y21, Y17) \
	BTFNK(K24, Y6, K40, Y5, Y22, Y18) \
	BTFNK(K56, Y7, K8, Y4, Y23, Y19) \
	VMOVDQU32 Y16, Y4 \
	VMOVDQU32 Y17, Y5 \
	VMOVDQU32 Y18, Y6 \
	VMOVDQU32 Y19, Y7 \
	BFLY(Y8, Y9, Y16) \
	BFLY(Y11, Y10, Y16) \
	BFLY(Y12, Y13, Y16) \
	BFLY(Y15, Y14, Y16) \
	\
	BTFK(K60, Y8, K4, Y15, Y24, Y16) \
	BTFK(K28, Y9, K36, Y14, Y24, Y17) \
	BTFK(K44, Y10, K20, Y13, Y24, Y18) \
	BTFK(K12, Y11, K52, Y12, Y24, Y19) \
	BTFNK(K12, Y12, K52, Y11, Y24, Y20) \
	BTFNK(K44, Y13, K20, Y10, Y24, Y21) \
	BTFNK(K28, Y14, K36, Y9, Y24, Y22) \
	BTFNK(K60, Y15, K4, Y8, Y24, Y23) \
	VMOVDQU32 Y16, Y8 \
	VMOVDQU32 Y17, Y9 \
	VMOVDQU32 Y18, Y10 \
	VMOVDQU32 Y19, Y11 \
	VMOVDQU32 Y20, Y12 \
	VMOVDQU32 Y21, Y13 \
	VMOVDQU32 Y22, Y14 \
	VMOVDQU32 Y23, Y15

#define T8(i0, i1, i2, i3, i4, i5, i6, i7, t0, t1, t2, t3, t4, t5, t6, t7, u0, u1, u2, u3, u4, u5, u6, u7) \
	VPUNPCKLDQ  i1, i0, t0 \
	VPUNPCKHDQ  i1, i0, t1 \
	VPUNPCKLDQ  i3, i2, t2 \
	VPUNPCKHDQ  i3, i2, t3 \
	VPUNPCKLDQ  i5, i4, t4 \
	VPUNPCKHDQ  i5, i4, t5 \
	VPUNPCKLDQ  i7, i6, t6 \
	VPUNPCKHDQ  i7, i6, t7 \
	VPUNPCKLQDQ t2, t0, u0 \
	VPUNPCKHQDQ t2, t0, u1 \
	VPUNPCKLQDQ t3, t1, u2 \
	VPUNPCKHQDQ t3, t1, u3 \
	VPUNPCKLQDQ t6, t4, u4 \
	VPUNPCKHQDQ t6, t4, u5 \
	VPUNPCKLQDQ t7, t5, u6 \
	VPUNPCKHQDQ t7, t5, u7 \
	VSHUFI32X4  $0x00, u4, u0, i0 \
	VSHUFI32X4  $0x00, u5, u1, i1 \
	VSHUFI32X4  $0x00, u6, u2, i2 \
	VSHUFI32X4  $0x00, u7, u3, i3 \
	VSHUFI32X4  $0x03, u4, u0, i4 \
	VSHUFI32X4  $0x03, u5, u1, i5 \
	VSHUFI32X4  $0x03, u6, u2, i6 \
	VSHUFI32X4  $0x03, u7, u3, i7

// FDCT8Z is FDCT8 over sixteen lanes: the rows live in Z16..Z23 and Z0..Z15
// are all temporaries, since the constants are embedded broadcasts.
#define FDCT8Z \
	VPADDD Z23, Z16, Z0 \
	VPADDD Z22, Z17, Z1 \
	VPADDD Z21, Z18, Z2 \
	VPADDD Z20, Z19, Z3 \
	VPSUBD Z20, Z19, Z4 \
	VPSUBD Z21, Z18, Z5 \
	VPSUBD Z22, Z17, Z6 \
	VPSUBD Z23, Z16, Z7 \
	VPADDD Z3, Z0, Z8 \
	VPADDD Z2, Z1, Z9 \
	VPSUBD Z2, Z1, Z10 \
	VPSUBD Z3, Z0, Z11 \
	BTFNK(K32, Z6, K32, Z5, Z13, Z12) \
	BTFK(K32, Z6, K32, Z5, Z14, Z13) \
	BTFK(K32, Z8, K32, Z9, Z15, Z16) \
	BTFNK(K32, Z8, K32, Z9, Z15, Z20) \
	BTFK(K48, Z10, K16, Z11, Z15, Z18) \
	BTFNK(K48, Z11, K16, Z10, Z15, Z22) \
	VPADDD Z12, Z4, Z8 \
	VPSUBD Z12, Z4, Z9 \
	VPSUBD Z13, Z7, Z10 \
	VPADDD Z13, Z7, Z11 \
	BTFK(K56, Z8, K8, Z11, Z15, Z17) \
	BTFK(K24, Z9, K40, Z10, Z15, Z21) \
	BTFNK(K24, Z10, K40, Z9, Z15, Z19) \
	BTFNK(K56, Z11, K8, Z8, Z15, Z23)

// RECT2 is the round shift by two the 8x16 and 16x8 shift schedules ask for,
// followed by the rectangular Sqrt2 correction.
#define RECT2(v) \
	VPADDD.BCST  KTWO, v, v \
	VPSRAD       $2, v, v \
	VPMULLD.BCST K32, v, v \
	VPADDD.BCST  KSQR, v, v \
	VPSRAD       $12, v, v

// fwdTxfm8x16AVX512 is the 8x16 DCT_DCT forward transform: sixteen points down
// the columns over eight lanes, then eight points along the rows over sixteen.
TEXT ·fwdTxfm8x16AVX512(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), Y0
	VMOVDQU (SI)(DX*1), Y1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), Y2
	VMOVDQU (AX)(DX*1), Y3
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y4
	VMOVDQU (AX)(DX*1), Y5
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y6
	VMOVDQU (AX)(DX*1), Y7
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y8
	VMOVDQU (AX)(DX*1), Y9
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y10
	VMOVDQU (AX)(DX*1), Y11
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y12
	VMOVDQU (AX)(DX*1), Y13
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y14
	VMOVDQU (AX)(DX*1), Y15

	VPSLLD $2, Y0, Y0
	VPSLLD $2, Y1, Y1
	VPSLLD $2, Y2, Y2
	VPSLLD $2, Y3, Y3
	VPSLLD $2, Y4, Y4
	VPSLLD $2, Y5, Y5
	VPSLLD $2, Y6, Y6
	VPSLLD $2, Y7, Y7
	VPSLLD $2, Y8, Y8
	VPSLLD $2, Y9, Y9
	VPSLLD $2, Y10, Y10
	VPSLLD $2, Y11, Y11
	VPSLLD $2, Y12, Y12
	VPSLLD $2, Y13, Y13
	VPSLLD $2, Y14, Y14
	VPSLLD $2, Y15, Y15

	FDCT16

	T8(Y0, Y8, Y4, Y12, Y2, Y10, Y6, Y14, Y16, Y17, Y18, Y19, Y20, Y21, Y22, Y23, Y24, Y25, Y26, Y27, Y28, Y29, Y30, Y31)
	T8(Y1, Y9, Y5, Y13, Y3, Y11, Y7, Y15, Y16, Y17, Y18, Y19, Y20, Y21, Y22, Y23, Y24, Y25, Y26, Y27, Y28, Y29, Y30, Y31)

	VINSERTI64X4 $1, Y1, Z0, Z16
	VINSERTI64X4 $1, Y9, Z8, Z17
	VINSERTI64X4 $1, Y5, Z4, Z18
	VINSERTI64X4 $1, Y13, Z12, Z19
	VINSERTI64X4 $1, Y3, Z2, Z20
	VINSERTI64X4 $1, Y11, Z10, Z21
	VINSERTI64X4 $1, Y7, Z6, Z22
	VINSERTI64X4 $1, Y15, Z14, Z23

	FDCT8Z

	RECT2(Z16)
	RECT2(Z17)
	RECT2(Z18)
	RECT2(Z19)
	RECT2(Z20)
	RECT2(Z21)
	RECT2(Z22)
	RECT2(Z23)

	VMOVDQU32 Z16, (DI)
	VMOVDQU32 Z17, 64(DI)
	VMOVDQU32 Z18, 128(DI)
	VMOVDQU32 Z19, 192(DI)
	VMOVDQU32 Z20, 256(DI)
	VMOVDQU32 Z21, 320(DI)
	VMOVDQU32 Z22, 384(DI)
	VMOVDQU32 Z23, 448(DI)
	VZEROUPPER
	RET

// fwdTxfm16x8AVX512 is the 16x8 DCT_DCT forward transform: eight points down
// the columns over sixteen lanes, then sixteen points along the rows over
// eight. The 16x8 transpose is two 8x8 transposes, one per half of each row.
TEXT ·fwdTxfm16x8AVX512(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), DX
	SHLQ $2, DX

	VMOVDQU32 (SI), Z16
	VMOVDQU32 (SI)(DX*1), Z17
	LEAQ      (SI)(DX*2), AX
	VMOVDQU32 (AX), Z18
	VMOVDQU32 (AX)(DX*1), Z19
	LEAQ      (AX)(DX*2), AX
	VMOVDQU32 (AX), Z20
	VMOVDQU32 (AX)(DX*1), Z21
	LEAQ      (AX)(DX*2), AX
	VMOVDQU32 (AX), Z22
	VMOVDQU32 (AX)(DX*1), Z23

	VPSLLD $2, Z16, Z16
	VPSLLD $2, Z17, Z17
	VPSLLD $2, Z18, Z18
	VPSLLD $2, Z19, Z19
	VPSLLD $2, Z20, Z20
	VPSLLD $2, Z21, Z21
	VPSLLD $2, Z22, Z22
	VPSLLD $2, Z23, Z23

	FDCT8Z

	VEXTRACTI64X4 $1, Z16, Y0
	VEXTRACTI64X4 $1, Z17, Y1
	VEXTRACTI64X4 $1, Z18, Y2
	VEXTRACTI64X4 $1, Z19, Y3
	VEXTRACTI64X4 $1, Z20, Y4
	VEXTRACTI64X4 $1, Z21, Y5
	VEXTRACTI64X4 $1, Z22, Y6
	VEXTRACTI64X4 $1, Z23, Y7

	T8(Y16, Y17, Y18, Y19, Y20, Y21, Y22, Y23, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15, Y24, Y25, Y26, Y27, Y28, Y29, Y30, Y31)
	T8(Y0, Y1, Y2, Y3, Y4, Y5, Y6, Y7, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15, Y24, Y25, Y26, Y27, Y28, Y29, Y30, Y31)

	VMOVDQU32 Y0, Y8
	VMOVDQU32 Y1, Y9
	VMOVDQU32 Y2, Y10
	VMOVDQU32 Y3, Y11
	VMOVDQU32 Y4, Y12
	VMOVDQU32 Y5, Y13
	VMOVDQU32 Y6, Y14
	VMOVDQU32 Y7, Y15
	VMOVDQU32 Y16, Y0
	VMOVDQU32 Y17, Y1
	VMOVDQU32 Y18, Y2
	VMOVDQU32 Y19, Y3
	VMOVDQU32 Y20, Y4
	VMOVDQU32 Y21, Y5
	VMOVDQU32 Y22, Y6
	VMOVDQU32 Y23, Y7

	FDCT16

	RECT2(Y0)
	RECT2(Y1)
	RECT2(Y2)
	RECT2(Y3)
	RECT2(Y4)
	RECT2(Y5)
	RECT2(Y6)
	RECT2(Y7)
	RECT2(Y8)
	RECT2(Y9)
	RECT2(Y10)
	RECT2(Y11)
	RECT2(Y12)
	RECT2(Y13)
	RECT2(Y14)
	RECT2(Y15)

	VMOVDQU Y0, (DI)
	VMOVDQU Y8, 32(DI)
	VMOVDQU Y4, 64(DI)
	VMOVDQU Y12, 96(DI)
	VMOVDQU Y2, 128(DI)
	VMOVDQU Y10, 160(DI)
	VMOVDQU Y6, 192(DI)
	VMOVDQU Y14, 224(DI)
	VMOVDQU Y1, 256(DI)
	VMOVDQU Y9, 288(DI)
	VMOVDQU Y5, 320(DI)
	VMOVDQU Y13, 352(DI)
	VMOVDQU Y3, 384(DI)
	VMOVDQU Y11, 416(DI)
	VMOVDQU Y7, 448(DI)
	VMOVDQU Y15, 480(DI)
	VZEROUPPER
	RET
