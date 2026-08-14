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
