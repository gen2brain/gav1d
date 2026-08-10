//go:build amd64 && !noasm

#include "textflag.h"

DATA pd128<>+0(SB)/4, $128
GLOBL pd128<>(SB), RODATA|NOPTR, $4

DATA pd256<>+0(SB)/4, $256
GLOBL pd256<>(SB), RODATA|NOPTR, $4

DATA pd8<>+0(SB)/4, $8
GLOBL pd8<>(SB), RODATA|NOPTR, $4

DATA pd32<>+0(SB)/4, $32
GLOBL pd32<>(SB), RODATA|NOPTR, $4

#define SMV16_ROW          \
	VPBROADCASTD (R12), Y3; \
	VPMADDWD     Y3, Y1, Y0; \
	VPADDD       Y7, Y0, Y0; \
	VPSRAD       $8, Y0, Y0; \
	VPACKUSDW    Y0, Y0, Y0; \
	VPERMQ       $0xD8, Y0, Y0; \
	ADDQ         $4, R12

// func smoothV16AVX2(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)
TEXT ·smoothV16AVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ weights+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	SHLQ $1, SI

	MOVQ R8, AX
	NEGQ AX
	VPBROADCASTW (DX)(AX*2), Y5
	VPBROADCASTD pd128<>(SB), Y7

	CMPQ CX, $4
	JEQ  smv164

	XORQ R10, R10

smv168col:
	VMOVDQU     2(DX)(R10*2), X0
	VPUNPCKLWD  X5, X0, X1
	VPUNPCKHWD  X5, X0, X2
	VINSERTI128 $1, X2, Y1, Y1

	MOVQ R8, R11
	MOVQ R9, R12
	LEAQ (DI)(R10*2), R14

smv168row:
	SMV16_ROW
	VMOVDQU X0, (R14)
	ADDQ    SI, R14
	DECQ    R11
	JNZ     smv168row

	ADDQ $8, R10
	CMPQ R10, CX
	JLT  smv168col

	VZEROUPPER
	RET

smv164:
	VMOVQ      2(DX), X0
	VPUNPCKLWD X5, X0, X1
	MOVQ       R8, R11
	MOVQ       R9, R12
	MOVQ       DI, R14

smv164row:
	SMV16_ROW
	VMOVQ X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   smv164row

	VZEROUPPER
	RET

#define SMH16_ROW           \
	VPBROADCASTW (R13), Y2; \
	VPUNPCKLWD   Y4, Y2, Y2; \
	VPMADDWD     Y11, Y2, Y0; \
	VPADDD       Y7, Y0, Y0; \
	VPSRAD       $8, Y0, Y0; \
	VPACKUSDW    Y0, Y0, Y0; \
	VPERMQ       $0xD8, Y0, Y0; \
	SUBQ         $2, R13

// func smoothH16AVX2(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)
TEXT ·smoothH16AVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ weights+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	SHLQ $1, SI

	VPBROADCASTW (DX)(CX*2), Y4
	VPBROADCASTD pd128<>(SB), Y7

	CMPQ CX, $4
	JEQ  smh164

	XORQ R10, R10

smh168col:
	VMOVDQU (R9)(R10*4), Y11
	MOVQ    R8, R11
	LEAQ    -2(DX), R13
	LEAQ    (DI)(R10*2), R14

smh168row:
	SMH16_ROW
	VMOVDQU X0, (R14)
	ADDQ    SI, R14
	DECQ    R11
	JNZ     smh168row

	ADDQ $8, R10
	CMPQ R10, CX
	JLT  smh168col

	VZEROUPPER
	RET

smh164:
	VMOVDQU (R9), Y11
	MOVQ    R8, R11
	LEAQ    -2(DX), R13
	MOVQ    DI, R14

smh164row:
	SMH16_ROW
	VMOVQ X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   smh164row

	VZEROUPPER
	RET

#define SM216_ROW           \
	VPBROADCASTW (R13), Y2; \
	VPUNPCKLWD   Y4, Y2, Y2; \
	VPMADDWD     Y11, Y2, Y8; \
	VPBROADCASTD (R12), Y3; \
	VPMADDWD     Y3, Y1, Y0; \
	VPADDD       Y8, Y0, Y0; \
	VPADDD       Y7, Y0, Y0; \
	VPSRAD       $9, Y0, Y0; \
	VPACKUSDW    Y0, Y0, Y0; \
	VPERMQ       $0xD8, Y0, Y0; \
	ADDQ         $4, R12;   \
	SUBQ         $2, R13

// func smooth16AVX2(dst *uint16, stride int, tl *uint16, vw, hw *int16, w, h int)
TEXT ·smooth16AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ vw+24(FP), R9
	MOVQ hw+32(FP), R15
	MOVQ w+40(FP), CX
	MOVQ h+48(FP), R8

	SHLQ $1, SI

	MOVQ R8, AX
	NEGQ AX
	VPBROADCASTW (DX)(AX*2), Y5
	VPBROADCASTW (DX)(CX*2), Y4
	VPBROADCASTD pd256<>(SB), Y7

	CMPQ CX, $4
	JEQ  sm2164

	XORQ R10, R10

sm2168col:
	VMOVDQU     2(DX)(R10*2), X0
	VPUNPCKLWD  X5, X0, X1
	VPUNPCKHWD  X5, X0, X2
	VINSERTI128 $1, X2, Y1, Y1
	VMOVDQU     (R15)(R10*4), Y11

	MOVQ R8, R11
	MOVQ R9, R12
	LEAQ -2(DX), R13
	LEAQ (DI)(R10*2), R14

sm2168row:
	SM216_ROW
	VMOVDQU X0, (R14)
	ADDQ    SI, R14
	DECQ    R11
	JNZ     sm2168row

	ADDQ $8, R10
	CMPQ R10, CX
	JLT  sm2168col

	VZEROUPPER
	RET

sm2164:
	VMOVQ      2(DX), X0
	VPUNPCKLWD X5, X0, X1
	VMOVDQU    (R15), Y11
	MOVQ       R8, R11
	MOVQ       R9, R12
	LEAQ       -2(DX), R13
	MOVQ       DI, R14

sm2164row:
	SM216_ROW
	VMOVQ X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   sm2164row

	VZEROUPPER
	RET

#define PAETH16_ROW          \
	VPBROADCASTW (R13), X3;  \
	VPSUBW    X5, X3, X2;    \
	VPABSW    X2, X8;        \
	VPADDW    X6, X2, X2;    \
	VPABSW    X2, X2;        \
	VPMINSW   X8, X7, X9;    \
	VPCMPEQW  X9, X7, X0;    \
	VPBLENDVB X0, X3, X1, X0; \
	VPMINSW   X2, X9, X10;   \
	VPCMPEQW  X10, X9, X10;  \
	VPBLENDVB X10, X0, X5, X0; \
	SUBQ      $2, R13

// func paeth16AVX2(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·paeth16AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	SHLQ $1, SI

	VPBROADCASTW (DX), X5

	CMPQ CX, $4
	JEQ  pth164

	XORQ R10, R10

pth168col:
	VMOVDQU 2(DX)(R10*2), X1
	VPSUBW  X5, X1, X6
	VPABSW  X6, X7
	MOVQ    R8, R11
	LEAQ    -2(DX), R13
	LEAQ    (DI)(R10*2), R14

pth168row:
	PAETH16_ROW
	VMOVDQU X0, (R14)
	ADDQ    SI, R14
	DECQ    R11
	JNZ     pth168row

	ADDQ $8, R10
	CMPQ R10, CX
	JLT  pth168col
	RET

pth164:
	VMOVQ  2(DX), X1
	VPSUBW X5, X1, X6
	VPABSW X6, X7
	MOVQ   R8, R11
	LEAQ   -2(DX), R13
	MOVQ   DI, R14

pth164row:
	PAETH16_ROW
	VMOVQ X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   pth164row
	RET

// func splatDc16AVX2(dst *uint16, stride, w, h int, dc uint16)
TEXT ·splatDc16AVX2(SB), NOSPLIT, $0-34
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ w+16(FP), CX
	MOVQ h+24(FP), R8

	SHLQ $1, SI

	VPBROADCASTW dc+32(FP), Y0

	CMPQ CX, $4
	JEQ  sp164
	CMPQ CX, $8
	JEQ  sp168
	CMPQ CX, $16
	JEQ  sp1616
	CMPQ CX, $32
	JEQ  sp1632

sp1664:
	VMOVDQU Y0, (DI)
	VMOVDQU Y0, 32(DI)
	VMOVDQU Y0, 64(DI)
	VMOVDQU Y0, 96(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp1664
	VZEROUPPER
	RET

sp164:
	VMOVQ X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   sp164
	VZEROUPPER
	RET

sp168:
	VMOVDQU X0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp168
	VZEROUPPER
	RET

sp1616:
	VMOVDQU Y0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp1616
	VZEROUPPER
	RET

sp1632:
	VMOVDQU Y0, (DI)
	VMOVDQU Y0, 32(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp1632
	VZEROUPPER
	RET

// func vpred16AVX2(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·vpred16AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	SHLQ $1, SI

	CMPQ CX, $4
	JEQ  vp164
	CMPQ CX, $8
	JEQ  vp168
	CMPQ CX, $16
	JEQ  vp1616
	CMPQ CX, $32
	JEQ  vp1632

	VMOVDQU 2(DX), Y0
	VMOVDQU 34(DX), Y1
	VMOVDQU 66(DX), Y2
	VMOVDQU 98(DX), Y3

vp1664:
	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	VMOVDQU Y2, 64(DI)
	VMOVDQU Y3, 96(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp1664
	VZEROUPPER
	RET

vp164:
	VMOVQ 2(DX), X0

vp164row:
	VMOVQ X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   vp164row
	RET

vp168:
	VMOVDQU 2(DX), X0

vp168row:
	VMOVDQU X0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp168row
	RET

vp1616:
	VMOVDQU 2(DX), Y0

vp1616row:
	VMOVDQU Y0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp1616row
	VZEROUPPER
	RET

vp1632:
	VMOVDQU 2(DX), Y0
	VMOVDQU 34(DX), Y1

vp1632row:
	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp1632row
	VZEROUPPER
	RET

// func hpred16AVX2(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·hpred16AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	SHLQ $1, SI
	LEAQ -2(DX), R13

	CMPQ CX, $4
	JEQ  hp164
	CMPQ CX, $8
	JEQ  hp168
	CMPQ CX, $16
	JEQ  hp1616
	CMPQ CX, $32
	JEQ  hp1632

hp1664:
	VPBROADCASTW (R13), Y0
	VMOVDQU      Y0, (DI)
	VMOVDQU      Y0, 32(DI)
	VMOVDQU      Y0, 64(DI)
	VMOVDQU      Y0, 96(DI)
	ADDQ         SI, DI
	SUBQ         $2, R13
	DECQ         R8
	JNZ          hp1664
	VZEROUPPER
	RET

hp164:
	VPBROADCASTW (R13), X0
	VMOVQ        X0, (DI)
	ADDQ         SI, DI
	SUBQ         $2, R13
	DECQ         R8
	JNZ          hp164
	RET

hp168:
	VPBROADCASTW (R13), X0
	VMOVDQU      X0, (DI)
	ADDQ         SI, DI
	SUBQ         $2, R13
	DECQ         R8
	JNZ          hp168
	RET

hp1616:
	VPBROADCASTW (R13), Y0
	VMOVDQU      Y0, (DI)
	ADDQ         SI, DI
	SUBQ         $2, R13
	DECQ         R8
	JNZ          hp1616
	VZEROUPPER
	RET

hp1632:
	VPBROADCASTW (R13), Y0
	VMOVDQU      Y0, (DI)
	VMOVDQU      Y0, 32(DI)
	ADDQ         SI, DI
	SUBQ         $2, R13
	DECQ         R8
	JNZ          hp1632
	VZEROUPPER
	RET

DATA fi16Shuf<>+0(SB)/8, $0x0504030201000d0c
DATA fi16Shuf<>+8(SB)/8, $0x80800b0a09080706
GLOBL fi16Shuf<>(SB), RODATA|NOPTR, $16

#define FI16_TAP(SEL, FV)     \
	VPSHUFD      SEL, X1, X2; \
	VPBROADCASTD X2, Y2;      \
	VPMADDWD     FV, Y2, Y13; \
	VPADDD       Y13, Y12, Y12

// func filterIntra16AVX2(dst *uint16, stride int, tl *uint16, f *int16, w, h int, bitdepthMax int32)
TEXT ·filterIntra16AVX2(SB), NOSPLIT, $0-52
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ f+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	SHLQ $1, SI

	VMOVDQU      (R9), Y8
	VMOVDQU      32(R9), Y9
	VMOVDQU      64(R9), Y10
	VMOVDQU      96(R9), Y11
	VPBROADCASTD pd8<>(SB), Y7
	MOVL         bitdepthMax+48(FP), AX
	VMOVD        AX, X6
	VPBROADCASTW X6, Y6
	VMOVDQU      fi16Shuf<>(SB), X14

	LEAQ 2(DX), R12
	XORQ R11, R11

fi16Row:
	MOVQ DI, R14
	LEAQ (DI)(SI*1), R15
	XORQ R10, R10

fi16Col:
	TESTQ R10, R10
	JNZ   fi16Mid

	MOVQ DX, R9
	SUBQ R11, R9
	SUBQ R11, R9
	LEAQ -2(R9), BX
	LEAQ -4(R9), R13
	JMP  fi16Have

fi16Mid:
	LEAQ -2(R12)(R10*2), R9
	LEAQ -2(R14)(R10*2), BX
	LEAQ -2(R15)(R10*2), R13

fi16Have:
	VMOVQ   (R12)(R10*2), X0
	VPINSRW $4, (BX), X0, X0
	VPINSRW $5, (R13), X0, X0
	VPINSRW $6, (R9), X0, X0
	VPSHUFB X14, X0, X1

	VPBROADCASTD X1, Y2
	VPMADDWD     Y8, Y2, Y12
	FI16_TAP($0x55, Y9)
	FI16_TAP($0xAA, Y10)
	FI16_TAP($0xFF, Y11)

	VPADDD    Y7, Y12, Y12
	VPSRAD    $4, Y12, Y12
	VPACKUSDW Y12, Y12, Y12
	VPERMQ    $0xD8, Y12, Y12
	VPMINUW   Y6, Y12, Y12
	VMOVQ     X12, (R14)(R10*2)
	VPEXTRQ   $1, X12, (R15)(R10*2)

	ADDQ $4, R10
	CMPQ R10, CX
	JLT  fi16Col

	LEAQ (R15), R12
	LEAQ (DI)(SI*2), DI
	ADDQ $2, R11
	CMPQ R11, R8
	JLT  fi16Row

	VZEROUPPER
	RET

#define Z116_WEIGHTS       \
	MOVQ  R11, AX;         \
	ANDQ  $0x3E, AX;       \
	MOVQ  $64, BX;         \
	SUBQ  AX, BX;          \
	SHLQ  $16, AX;         \
	ORQ   BX, AX;          \
	VMOVD AX, X3;          \
	MOVQ  R11, R15;        \
	SARQ  $6, R15;         \
	LEAQ  (DX)(R15*2), BX

#define Z116_X4(BASE)          \
	VMOVQ      (BASE), X0;     \
	VMOVQ      2(BASE), X2;    \
	VPUNPCKLWD X2, X0, X1;     \
	VPMADDWD   X3, X1, X0;     \
	VPADDD     X7, X0, X0;     \
	VPSRAD     $6, X0, X0;     \
	VPACKUSDW  X0, X0, X0

// func z1Full16AVX2(dst *uint16, stride int, top *uint16, w, rows, dx, xpos int)
TEXT ·z1Full16AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ top+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ rows+32(FP), R8
	MOVQ dx+40(FP), R9
	MOVQ xpos+48(FP), R11

	SHLQ $1, SI

	CMPQ CX, $4
	JEQ  z116w4
	CMPQ CX, $8
	JEQ  z116w8

	VPBROADCASTD pd32<>(SB), Y7

z116w16:
	Z116_WEIGHTS
	VPBROADCASTD X3, Y3
	MOVQ DI, R15
	XORQ R14, R14

z116w16col:
	VMOVDQU     (BX)(R14*2), X0
	VMOVDQU     2(BX)(R14*2), X2
	VPUNPCKLWD  X2, X0, X1
	VPUNPCKHWD  X2, X0, X4
	VINSERTI128 $1, X4, Y1, Y1
	VPMADDWD    Y3, Y1, Y0
	VPADDD      Y7, Y0, Y0
	VPSRAD      $6, Y0, Y0
	VPACKUSDW   Y0, Y0, Y0
	VPERMQ      $0xD8, Y0, Y0
	VMOVDQU     X0, (R15)(R14*2)
	ADDQ        $8, R14
	CMPQ        R14, CX
	JLT         z116w16col

	ADDQ SI, DI
	ADDQ R9, R11
	DECQ R8
	JNZ  z116w16
	VZEROUPPER
	RET

z116w4:
	VPBROADCASTD pd32<>(SB), X7

z116w4row:
	Z116_WEIGHTS
	VPSHUFD $0x00, X3, X3
	Z116_X4(BX)
	VMOVQ X0, (DI)
	ADDQ  SI, DI
	ADDQ  R9, R11
	DECQ  R8
	JNZ   z116w4row
	RET

z116w8:
	VPBROADCASTD pd32<>(SB), X7

z116w8row:
	Z116_WEIGHTS
	VPSHUFD $0x00, X3, X3
	Z116_X4(BX)
	VMOVQ X0, (DI)
	ADDQ  $8, BX
	Z116_X4(BX)
	VMOVQ X0, 8(DI)
	ADDQ  SI, DI
	ADDQ  R9, R11
	DECQ  R8
	JNZ   z116w8row
	RET
