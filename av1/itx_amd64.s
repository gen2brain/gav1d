//go:build amd64 && !noasm

#include "textflag.h"

// func itxClipAVX2(tmp *int32, n int, rnd int32, shift int, lo, hi int32)
#define WIDEN_SETUP                \
	MOVQ  dst+0(FP), DI;       \
	MOVQ  dstStride+8(FP), R8; \
	SHLQ  $2, R8;              \
	MOVQ  src+16(FP), SI;      \
	MOVQ  srcStride+24(FP), R9; \
	SHLQ  $1, R9;              \
	MOVQ  cols+32(FP), R10;    \
	MOVQ  n+40(FP), R11;       \
	MOVQ  lanes+48(FP), R12;   \
	VPXOR Y1, Y1, Y1

#define WIDEN_MASK(N)      \
	MOVL  $0xff, R13;  \
	BZHIQ N, R13, R13; \
	KMOVW R13, K1

TEXT ·itxClipAVX2(SB), NOSPLIT, $0-40
	MOVQ         tmp+0(FP), DI
	MOVQ         n+8(FP), CX
	MOVL         rnd+16(FP), AX
	VMOVD        AX, X1
	VPBROADCASTD X1, Y1
	MOVQ         shift+24(FP), AX
	VMOVD        AX, X2
	MOVL         lo+32(FP), AX
	VMOVD        AX, X3
	VPBROADCASTD X3, Y3
	MOVL         hi+36(FP), AX
	VMOVD        AX, X4
	VPBROADCASTD X4, Y4

loop:
	VMOVDQU  (DI), Y0
	VPADDD   Y1, Y0, Y0
	VPSRAD   X2, Y0, Y0
	VPMAXSD  Y3, Y0, Y0
	VPMINSD  Y4, Y0, Y0
	VMOVDQU  Y0, (DI)
	ADDQ     $32, DI
	SUBQ     $8, CX
	JNZ      loop

	VZEROUPPER
	RET

// ADDROW folds one vector of residual into the destination.
#define ADDROW(PX)                 \
	VMOVDQU (SI), Y0;          \
	VPADDD  Y5, Y0, Y0;        \
	VPSRAD  $4, Y0, Y0;        \
	VPADDD  PX, Y0, Y0;        \
	VPMAXSD Y6, Y0, Y0;        \
	VPMINSD Y7, Y0, Y0

#define ADDROW4(PX)                \
	VMOVDQU (SI), X0;          \
	VPADDD  X5, X0, X0;        \
	VPSRAD  $4, X0, X0;        \
	VPADDD  PX, X0, X0;        \
	VPMAXSD X6, X0, X0;        \
	VPMINSD X7, X0, X0

// func itxAdd8AVX2(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd8AVX2(SB), NOSPLIT, $0-52
	MOVQ         dst+0(FP), DI
	MOVQ         stride+8(FP), DX
	MOVQ         tmp+16(FP), SI
	MOVQ         w+24(FP), BX
	MOVQ         h+32(FP), CX
	MOVQ         ts+40(FP), R9
	MOVL         bitdepthMax+48(FP), AX
	VMOVD        AX, X7
	VPBROADCASTD X7, Y7
	VPXOR        Y6, Y6, Y6
	MOVL         $8, AX
	VMOVD        AX, X5
	VPBROADCASTD X5, Y5
	SHLQ         $2, R9
	CMPQ         BX, $4
	JEQ          rows8x4

rows8:
	XORQ R8, R8

cols8:
	VPMOVZXBD (DI)(R8*1), Y1
	ADDROW(Y1)
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VPACKUSWB X0, X0, X0
	VMOVQ     X0, (DI)(R8*1)
	ADDQ      $32, SI
	ADDQ      $8, R8
	CMPQ      R8, BX
	JLT       cols8

	ADDQ DX, DI
	DECQ CX
	JNZ  rows8

	VZEROUPPER
	RET

rows8x4:
	VPMOVZXBD (DI), X1
	ADDROW4(X1)
	VPACKUSDW X0, X0, X0
	VPACKUSWB X0, X0, X0
	VMOVD     X0, (DI)

	ADDQ R9, SI
	ADDQ DX, DI
	DECQ CX
	JNZ  rows8x4

	VZEROUPPER
	RET

// func itxAdd16AVX2(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd16AVX2(SB), NOSPLIT, $0-52
	MOVQ         dst+0(FP), DI
	MOVQ         stride+8(FP), DX
	MOVQ         tmp+16(FP), SI
	MOVQ         w+24(FP), BX
	MOVQ         h+32(FP), CX
	MOVQ         ts+40(FP), R9
	MOVL         bitdepthMax+48(FP), AX
	VMOVD        AX, X7
	VPBROADCASTD X7, Y7
	VPXOR        Y6, Y6, Y6
	MOVL         $8, AX
	VMOVD        AX, X5
	VPBROADCASTD X5, Y5
	SHLQ         $1, DX
	SHLQ         $2, R9
	CMPQ         BX, $4
	JEQ          rows16x4

rows16:
	XORQ R8, R8

cols16:
	VPMOVZXWD (DI)(R8*2), Y1
	ADDROW(Y1)
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (DI)(R8*2)
	ADDQ      $32, SI
	ADDQ      $8, R8
	CMPQ      R8, BX
	JLT       cols16

	ADDQ DX, DI
	DECQ CX
	JNZ  rows16

	VZEROUPPER
	RET

rows16x4:
	VPMOVZXWD (DI), X1
	ADDROW4(X1)
	VPACKUSDW X0, X0, X0
	VMOVQ     X0, (DI)

	ADDQ R9, SI
	ADDQ DX, DI
	DECQ CX
	JNZ  rows16x4

	VZEROUPPER
	RET

#define LOAD8(P, S)     \
	VMOVDQU (P), Y0; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y1; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y2; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y3; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y4; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y5; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y6; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y7

#define STORE8(P, S)     \
	VMOVDQU Y8, (P);  \
	ADDQ    S, P;     \
	VMOVDQU Y9, (P);  \
	ADDQ    S, P;     \
	VMOVDQU Y10, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y11, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y12, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y13, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y14, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y15, (P)

// func itxTransposeAVX2(wide, out *int32, w, ws, ts, n int)
TEXT ·itxTransposeAVX2(SB), NOSPLIT, $0-48
	MOVQ wide+0(FP), SI
	MOVQ out+8(FP), DI
	MOVQ w+16(FP), BX
	MOVQ ws+24(FP), DX
	MOVQ ts+32(FP), R9
	MOVQ n+40(FP), CX
	SHLQ $2, DX
	SHLQ $2, R9

trlane:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

trtap:
	MOVQ R10, R13
	LOAD8(R13, DX)

	VPUNPCKLDQ Y1, Y0, Y8
	VPUNPCKHDQ Y1, Y0, Y9
	VPUNPCKLDQ Y3, Y2, Y10
	VPUNPCKHDQ Y3, Y2, Y11
	VPUNPCKLDQ Y5, Y4, Y12
	VPUNPCKHDQ Y5, Y4, Y13
	VPUNPCKLDQ Y7, Y6, Y14
	VPUNPCKHDQ Y7, Y6, Y15

	VPUNPCKLQDQ Y10, Y8, Y0
	VPUNPCKHQDQ Y10, Y8, Y1
	VPUNPCKLQDQ Y11, Y9, Y2
	VPUNPCKHQDQ Y11, Y9, Y3
	VPUNPCKLQDQ Y14, Y12, Y4
	VPUNPCKHQDQ Y14, Y12, Y5
	VPUNPCKLQDQ Y15, Y13, Y6
	VPUNPCKHQDQ Y15, Y13, Y7

	VPERM2I128 $0x20, Y4, Y0, Y8
	VPERM2I128 $0x20, Y5, Y1, Y9
	VPERM2I128 $0x20, Y6, Y2, Y10
	VPERM2I128 $0x20, Y7, Y3, Y11
	VPERM2I128 $0x31, Y4, Y0, Y12
	VPERM2I128 $0x31, Y5, Y1, Y13
	VPERM2I128 $0x31, Y6, Y2, Y14
	VPERM2I128 $0x31, Y7, Y3, Y15

	MOVQ R11, R13
	STORE8(R13, R9)

	LEAQ (R10)(DX*8), R10
	ADDQ $32, R11
	SUBQ $8, R12
	JNZ  trtap

	ADDQ $32, SI
	LEAQ (DI)(R9*8), DI
	SUBQ $8, CX
	JNZ  trlane

	VZEROUPPER
	RET


// func widenCoefs16AVX2(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16AVX2(SB), NOSPLIT, $0-56
	WIDEN_SETUP
	TESTQ R10, R10
	JZ    wdone

wcol:
	MOVQ DI, DX
	MOVQ SI, CX
	XORQ BX, BX

w8:
	LEAQ 8(BX), AX
	CMPQ AX, R11
	JG   w1
	VPMOVSXWD (CX)(BX*2), Y0
	VMOVDQU   Y0, (DX)(BX*4)
	MOVQ      AX, BX
	JMP       w8

w1:
	CMPQ    BX, R11
	JGE     wz8
	MOVWLSX (CX)(BX*2), AX
	MOVL    AX, (DX)(BX*4)
	INCQ    BX
	JMP     w1

wz8:
	LEAQ    8(BX), AX
	CMPQ    AX, R12
	JG      wz1
	VMOVDQU Y1, (DX)(BX*4)
	MOVQ    AX, BX
	JMP     wz8

wz1:
	CMPQ BX, R12
	JGE  wnext
	MOVL $0, (DX)(BX*4)
	INCQ BX
	JMP  wz1

wnext:
	ADDQ R8, DI
	ADDQ R9, SI
	DECQ R10
	JNZ  wcol

wdone:
	VZEROUPPER
	RET

// func widenCoefs16Rect2AVX2(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16Rect2AVX2(SB), NOSPLIT, $0-56
	WIDEN_SETUP
	TESTQ R10, R10
	JZ    rdone

	MOVL         $181, AX
	MOVQ         AX, X2
	VPBROADCASTD X2, Y2
	MOVL         $128, AX
	MOVQ         AX, X3
	VPBROADCASTD X3, Y3

rcol:
	MOVQ DI, DX
	MOVQ SI, CX
	XORQ BX, BX

r8:
	LEAQ 8(BX), AX
	CMPQ AX, R11
	JG   r1
	VPMOVSXWD (CX)(BX*2), Y0
	VPMULLD   Y2, Y0, Y0
	VPADDD    Y3, Y0, Y0
	VPSRAD    $8, Y0, Y0
	VMOVDQU   Y0, (DX)(BX*4)
	MOVQ      AX, BX
	JMP       r8

r1:
	CMPQ    BX, R11
	JGE     rz8
	MOVWLSX (CX)(BX*2), AX
	IMULL   $181, AX
	ADDL    $128, AX
	SARL    $8, AX
	MOVL    AX, (DX)(BX*4)
	INCQ    BX
	JMP     r1

rz8:
	LEAQ    8(BX), AX
	CMPQ    AX, R12
	JG      rz1
	VMOVDQU Y1, (DX)(BX*4)
	MOVQ    AX, BX
	JMP     rz8

rz1:
	CMPQ BX, R12
	JGE  rnext
	MOVL $0, (DX)(BX*4)
	INCQ BX
	JMP  rz1

rnext:
	ADDQ R8, DI
	ADDQ R9, SI
	DECQ R10
	JNZ  rcol

rdone:
	VZEROUPPER
	RET

// func widenCoefs16AVX512(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16AVX512(SB), NOSPLIT, $0-56
	WIDEN_SETUP
	TESTQ R10, R10
	JZ    vwdone

vwcol:
	MOVQ DI, DX
	MOVQ SI, CX
	XORQ BX, BX

vw8:
	MOVQ R11, AX
	SUBQ BX, AX
	JLE  vwz8
	CMPQ AX, $8
	JL   vwpart
	VPMOVSXWD (CX)(BX*2), Y0
	VMOVDQU   Y0, (DX)(BX*4)
	ADDQ      $8, BX
	JMP       vw8

vwpart:
	WIDEN_MASK(AX)
	VPMOVSXWD.Z (CX)(BX*2), K1, Y0
	VMOVDQU32 Y0, K1, (DX)(BX*4)
	MOVQ      R11, BX

vwz8:
	MOVQ R12, AX
	SUBQ BX, AX
	JLE  vwnext
	CMPQ AX, $8
	JL   vwzpart
	VMOVDQU Y1, (DX)(BX*4)
	ADDQ    $8, BX
	JMP     vwz8

vwzpart:
	WIDEN_MASK(AX)
	VMOVDQU32 Y1, K1, (DX)(BX*4)

vwnext:
	ADDQ R8, DI
	ADDQ R9, SI
	DECQ R10
	JNZ  vwcol

vwdone:
	VZEROUPPER
	RET

// func widenCoefs16Rect2AVX512(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16Rect2AVX512(SB), NOSPLIT, $0-56
	WIDEN_SETUP
	TESTQ R10, R10
	JZ    vrdone

	MOVL         $181, AX
	MOVQ         AX, X2
	VPBROADCASTD X2, Y2
	MOVL         $128, AX
	MOVQ         AX, X3
	VPBROADCASTD X3, Y3

vrcol:
	MOVQ DI, DX
	MOVQ SI, CX
	XORQ BX, BX

vr8:
	MOVQ R11, AX
	SUBQ BX, AX
	JLE  vrz8
	CMPQ AX, $8
	JL   vrpart
	VPMOVSXWD (CX)(BX*2), Y0
	VPMULLD   Y2, Y0, Y0
	VPADDD    Y3, Y0, Y0
	VPSRAD    $8, Y0, Y0
	VMOVDQU   Y0, (DX)(BX*4)
	ADDQ      $8, BX
	JMP       vr8

vrpart:
	WIDEN_MASK(AX)
	VPMOVSXWD.Z (CX)(BX*2), K1, Y0
	VPMULLD   Y2, Y0, Y0
	VPADDD    Y3, Y0, Y0
	VPSRAD    $8, Y0, Y0
	VMOVDQU32 Y0, K1, (DX)(BX*4)
	MOVQ      R11, BX

vrz8:
	MOVQ R12, AX
	SUBQ BX, AX
	JLE  vrnext
	CMPQ AX, $8
	JL   vrzpart
	VMOVDQU Y1, (DX)(BX*4)
	ADDQ    $8, BX
	JMP     vrz8

vrzpart:
	WIDEN_MASK(AX)
	VMOVDQU32 Y1, K1, (DX)(BX*4)

vrnext:
	ADDQ R8, DI
	ADDQ R9, SI
	DECQ R10
	JNZ  vrcol

vrdone:
	VZEROUPPER
	RET

// func itxDC8AVX2(dst *uint8, stride int, dc int32, w, h int, bitdepthMax int32)
TEXT ·itxDC8AVX2(SB), NOSPLIT, $0-44
	MOVQ  dst+0(FP), DI
	MOVQ  stride+8(FP), SI
	MOVL  dc+16(FP), AX
	MOVQ  w+24(FP), DX
	MOVQ  h+32(FP), CX

	XORL R8, R8
	TESTL AX, AX
	JNS  dcpos8
	NEGL AX
	MOVL $1, R8

dcpos8:
	CMPL AX, $255
	JBE  dcsat8
	MOVL $255, AX

dcsat8:
	MOVD         AX, X0
	VPBROADCASTB X0, Y0

rowloop8:
	XORQ BX, BX

colloop8:
	MOVQ DX, R9
	SUBQ BX, R9
	CMPQ R9, $32
	JB   tail8
	VMOVDQU (DI)(BX*1), Y1
	TESTL   R8, R8
	JNZ     sub8
	VPADDUSB Y0, Y1, Y1
	JMP      st8

sub8:
	VPSUBUSB Y0, Y1, Y1

st8:
	VMOVDQU Y1, (DI)(BX*1)
	ADDQ    $32, BX
	JMP     colloop8

tail8:
	TESTQ R9, R9
	JZ    rownext8
	MOVBLZX (DI)(BX*1), R10
	TESTL   R8, R8
	JNZ     tsub8
	ADDL    AX, R10
	CMPL    R10, $255
	JBE     tst8
	MOVL    $255, R10
	JMP     tst8

tsub8:
	SUBL AX, R10
	JAE  tst8
	XORL R10, R10

tst8:
	MOVB R10, (DI)(BX*1)
	INCQ BX
	DECQ R9
	JMP  tail8

rownext8:
	ADDQ SI, DI
	DECQ CX
	JNZ  rowloop8
	VZEROUPPER
	RET

// func itxDC16AVX2(dst *uint16, stride int, dc int32, w, h int, bitdepthMax int32)
TEXT ·itxDC16AVX2(SB), NOSPLIT, $0-44
	MOVQ  dst+0(FP), DI
	MOVQ  stride+8(FP), SI
	MOVL  dc+16(FP), AX
	MOVQ  w+24(FP), DX
	MOVQ  h+32(FP), R11
	MOVL  bitdepthMax+40(FP), R12
	SHLQ  $1, SI
	SHLQ  $1, DX

	XORL  R8, R8
	TESTL AX, AX
	JNS   dcpos16
	NEGL  AX
	MOVL  $1, R8

dcpos16:
	CMPL AX, $65535
	JBE  dcsat16
	MOVL $65535, AX

dcsat16:
	MOVD         AX, X0
	VPBROADCASTW X0, Y0
	MOVD         R12, X2
	VPBROADCASTW X2, Y2

rowloop16:
	XORQ BX, BX

colloop16:
	MOVQ DX, R9
	SUBQ BX, R9
	CMPQ R9, $32
	JB   tail16
	VMOVDQU (DI)(BX*1), Y1
	TESTL   R8, R8
	JNZ     sub16
	VPADDUSW Y0, Y1, Y1
	VPMINUW  Y2, Y1, Y1
	JMP      st16

sub16:
	VPSUBUSW Y0, Y1, Y1

st16:
	VMOVDQU Y1, (DI)(BX*1)
	ADDQ    $32, BX
	JMP     colloop16

tail16:
	TESTQ R9, R9
	JZ    rownext16
	MOVWLZX (DI)(BX*1), R10
	TESTL   R8, R8
	JNZ     tsub16
	ADDL    AX, R10
	CMPL    R10, R12
	JBE     tst16
	MOVL    R12, R10
	JMP     tst16

tsub16:
	SUBL AX, R10
	JAE  tst16
	XORL R10, R10

tst16:
	MOVW R10, (DI)(BX*1)
	ADDQ $2, BX
	SUBQ $2, R9
	JMP  tail16

rownext16:
	ADDQ SI, DI
	DECQ R11
	JNZ  rowloop16
	VZEROUPPER
	RET

// The residual rows are contiguous, so sixteen pixels widen in one load and
// VPMOVUSDB packs them back in one store.
#define ADDROWZ(PX)          \
	VMOVDQU32 (SI), Z0;  \
	VPADDD    Z5, Z0, Z0; \
	VPSRAD    $4, Z0, Z0; \
	VPADDD    PX, Z0, Z0; \
	VPMAXSD   Z6, Z0, Z0; \
	VPMINSD   Z7, Z0, Z0

// func itxAdd8AVX512(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd8AVX512(SB), NOSPLIT, $0-52
	MOVQ         dst+0(FP), DI
	MOVQ         stride+8(FP), DX
	MOVQ         tmp+16(FP), SI
	MOVQ         w+24(FP), BX
	MOVQ         h+32(FP), CX
	MOVL         bitdepthMax+48(FP), AX
	VPBROADCASTD AX, Z7
	VPXORD       Z6, Z6, Z6
	MOVL         $8, AX
	VPBROADCASTD AX, Z5

rowsz:
	XORQ R8, R8

colsz:
	VPMOVZXBD (DI)(R8*1), Z1
	ADDROWZ(Z1)
	VPMOVUSDB Z0, (DI)(R8*1)
	ADDQ      $64, SI
	ADDQ      $16, R8
	CMPQ      R8, BX
	JLT       colsz

	ADDQ DX, DI
	DECQ CX
	JNZ  rowsz

	VZEROUPPER
	RET
