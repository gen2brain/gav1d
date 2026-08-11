//go:build amd64 && !noasm

#include "textflag.h"

DATA pb127m127<>+0(SB)/4, $0x817f817f
GLOBL pb127m127<>(SB), RODATA|NOPTR, $4

DATA pw128<>+0(SB)/4, $0x00800080
GLOBL pw128<>(SB), RODATA|NOPTR, $4

DATA pw255<>+0(SB)/4, $0x00ff00ff
GLOBL pw255<>(SB), RODATA|NOPTR, $4

DATA pb1<>+0(SB)/4, $0x01010101
GLOBL pb1<>(SB), RODATA|NOPTR, $4

#define SMV_ROW           \
	VPBROADCASTW (R12), Y3;   \
	VPMADDUBSW   Y3, Y1, Y0;  \
	VPADDW       Y2, Y0, Y0;  \
	VPSRLW       $8, Y0, Y0;  \
	ADDQ         $2, R12

#define SMV_TOP            \
	VPUNPCKLBW X5, X0, X1; \
	VPMADDUBSW Y6, Y1, Y2; \
	VPADDW     Y1, Y2, Y2; \
	VPADDW     Y7, Y2, Y2; \
	MOVQ       R8, R11;    \
	MOVQ       R9, R12;    \
	MOVQ       DI, R14

// func smoothV8AVX2(dst *uint8, stride int, tl *uint8, weights *int8, w, h int)
TEXT ·smoothV8AVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ weights+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	MOVQ R8, AX
	NEGQ AX

	VPBROADCASTB (DX)(AX*1), Y5
	VPBROADCASTD pb127m127<>(SB), Y6
	VPBROADCASTD pw128<>(SB), Y7

	CMPQ CX, $4
	JEQ  smv4
	CMPQ CX, $8
	JEQ  smv8

	XORQ R10, R10

smv16col:
	VMOVDQU     1(DX)(R10*1), X0
	VPUNPCKLBW  X5, X0, X1
	VPUNPCKHBW  X5, X0, X2
	VINSERTI128 $1, X2, Y1, Y1
	VPMADDUBSW  Y6, Y1, Y2
	VPADDW      Y1, Y2, Y2
	VPADDW      Y7, Y2, Y2

	MOVQ R8, R11
	MOVQ R9, R12
	LEAQ (DI)(R10*1), R14

smv16row:
	SMV_ROW
	VPACKUSWB Y0, Y0, Y0
	VPERMQ    $0xD8, Y0, Y0
	VMOVDQU   X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smv16row

	ADDQ $16, R10
	CMPQ R10, CX
	JLT  smv16col

	VZEROUPPER
	RET

smv4:
	VMOVD 1(DX), X0
	SMV_TOP

smv4row:
	SMV_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVD     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smv4row

	VZEROUPPER
	RET

smv8:
	VMOVQ 1(DX), X0
	SMV_TOP

smv8row:
	SMV_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVQ     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smv8row

	VZEROUPPER
	RET

#define SMH_ROW              \
	VPBROADCASTB (R13), Y3;  \
	VPUNPCKLBW   Y4, Y3, Y3; \
	VPMADDUBSW   Y6, Y3, Y2; \
	VPADDW       Y3, Y2, Y2; \
	VPADDW       Y7, Y2, Y2; \
	VPMADDUBSW   Y11, Y3, Y0; \
	VPADDW       Y2, Y0, Y0; \
	VPSRLW       $8, Y0, Y0; \
	DECQ         R13

// func smoothH8AVX2(dst *uint8, stride int, tl *uint8, weights *int8, w, h int)
TEXT ·smoothH8AVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ weights+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	VPBROADCASTB (DX)(CX*1), Y4
	VPBROADCASTD pb127m127<>(SB), Y6
	VPBROADCASTD pw128<>(SB), Y7

	CMPQ CX, $4
	JEQ  smh4
	CMPQ CX, $8
	JEQ  smh8

	XORQ R10, R10

smh16col:
	VMOVDQU (R9)(R10*2), Y11
	MOVQ    R8, R11
	LEAQ    -1(DX), R13
	LEAQ    (DI)(R10*1), R14

smh16row:
	SMH_ROW
	VPACKUSWB Y0, Y0, Y0
	VPERMQ    $0xD8, Y0, Y0
	VMOVDQU   X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smh16row

	ADDQ $16, R10
	CMPQ R10, CX
	JLT  smh16col

	VZEROUPPER
	RET

smh4:
	VMOVDQU (R9), Y11
	MOVQ    R8, R11
	LEAQ    -1(DX), R13
	MOVQ    DI, R14

smh4row:
	SMH_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVD     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smh4row

	VZEROUPPER
	RET

smh8:
	VMOVDQU (R9), Y11
	MOVQ    R8, R11
	LEAQ    -1(DX), R13
	MOVQ    DI, R14

smh8row:
	SMH_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVQ     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       smh8row

	VZEROUPPER
	RET

#define SM2_ROW              \
	VPBROADCASTB (R13), Y3;  \
	VPUNPCKLBW   Y4, Y3, Y3; \
	VPMADDUBSW   Y6, Y3, Y8; \
	VPADDW       Y3, Y8, Y8; \
	VPMADDUBSW   Y11, Y3, Y9; \
	VPADDW       Y8, Y9, Y9; \
	VPBROADCASTW (R12), Y10; \
	VPMADDUBSW   Y10, Y1, Y0; \
	VPADDW       Y2, Y0, Y0; \
	VPAVGW       Y9, Y0, Y0; \
	VPSRLW       $8, Y0, Y0; \
	ADDQ         $2, R12;    \
	DECQ         R13

#define SM2_TOP            \
	VPUNPCKLBW X5, X0, X1; \
	VPMADDUBSW Y6, Y1, Y2; \
	VPADDW     Y1, Y2, Y2; \
	VPADDW     Y7, Y2, Y2; \
	VMOVDQU    (R15), Y11; \
	MOVQ       R8, R11;    \
	MOVQ       R9, R12;    \
	LEAQ       -1(DX), R13; \
	MOVQ       DI, R14

// func smooth8AVX2(dst *uint8, stride int, tl *uint8, vw, hw *int8, w, h int)
TEXT ·smooth8AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ vw+24(FP), R9
	MOVQ hw+32(FP), R15
	MOVQ w+40(FP), CX
	MOVQ h+48(FP), R8

	MOVQ R8, AX
	NEGQ AX

	VPBROADCASTB (DX)(AX*1), Y5
	VPBROADCASTB (DX)(CX*1), Y4
	VPBROADCASTD pb127m127<>(SB), Y6
	VPBROADCASTD pw255<>(SB), Y7

	CMPQ CX, $4
	JEQ  sm24
	CMPQ CX, $8
	JEQ  sm28

	XORQ R10, R10

sm216col:
	VMOVDQU     1(DX)(R10*1), X0
	VPUNPCKLBW  X5, X0, X1
	VPUNPCKHBW  X5, X0, X2
	VINSERTI128 $1, X2, Y1, Y1
	VPMADDUBSW  Y6, Y1, Y2
	VPADDW      Y1, Y2, Y2
	VPADDW      Y7, Y2, Y2
	VMOVDQU     (R15)(R10*2), Y11

	MOVQ R8, R11
	MOVQ R9, R12
	LEAQ -1(DX), R13
	LEAQ (DI)(R10*1), R14

sm216row:
	SM2_ROW
	VPACKUSWB Y0, Y0, Y0
	VPERMQ    $0xD8, Y0, Y0
	VMOVDQU   X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       sm216row

	ADDQ $16, R10
	CMPQ R10, CX
	JLT  sm216col

	VZEROUPPER
	RET

sm24:
	VMOVD 1(DX), X0
	SM2_TOP

sm24row:
	SM2_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVD     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       sm24row

	VZEROUPPER
	RET

sm28:
	VMOVQ 1(DX), X0
	SM2_TOP

sm28row:
	SM2_ROW
	VPACKUSWB Y0, Y0, Y0
	VMOVQ     X0, (R14)
	ADDQ      SI, R14
	DECQ      R11
	JNZ       sm28row

	VZEROUPPER
	RET

#define PAETH_ROW                \
	VPBROADCASTB (R13), X3;      \
	VPAVGB       X3, X6, X1;     \
	VPXOR        X3, X6, X0;     \
	VPAND        X4, X0, X0;     \
	VPSUBUSB     X1, X5, X2;     \
	VPSUBB       X0, X1, X1;     \
	VPSUBUSB     X5, X1, X1;     \
	VPOR         X2, X1, X1;     \
	VPADDUSB     X1, X1, X1;     \
	VPOR         X0, X1, X1;     \
	VPSUBUSB     X3, X5, X2;     \
	VPSUBUSB     X5, X3, X0;     \
	VPOR         X0, X2, X2;     \
	VPMINUB      X7, X2, X2;     \
	VPCMPEQB     X2, X7, X0;     \
	VPBLENDVB    X0, X3, X6, X0; \
	VPMINUB      X2, X1, X1;     \
	VPCMPEQB     X2, X1, X1;     \
	VPBLENDVB    X1, X0, X5, X0; \
	DECQ         R13

#define PAETH_TOP        \
	VPSUBUSB X6, X5, X7; \
	VPSUBUSB X5, X6, X0; \
	VPOR     X0, X7, X7; \
	MOVQ     R8, R11;    \
	LEAQ     -1(DX), R13

// func paeth8AVX2(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·paeth8AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	VPBROADCASTB (DX), X5
	VPBROADCASTD pb1<>(SB), X4

	CMPQ CX, $4
	JEQ  pth4
	CMPQ CX, $8
	JEQ  pth8

	XORQ R10, R10

pth16col:
	VMOVDQU 1(DX)(R10*1), X6
	PAETH_TOP
	LEAQ (DI)(R10*1), R14

pth16row:
	PAETH_ROW
	VMOVDQU X0, (R14)
	ADDQ    SI, R14
	DECQ    R11
	JNZ     pth16row

	ADDQ $16, R10
	CMPQ R10, CX
	JLT  pth16col
	RET

pth4:
	VMOVD 1(DX), X6
	PAETH_TOP
	MOVQ DI, R14

pth4row:
	PAETH_ROW
	VMOVD X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   pth4row
	RET

pth8:
	VMOVQ 1(DX), X6
	PAETH_TOP
	MOVQ DI, R14

pth8row:
	PAETH_ROW
	VMOVQ X0, (R14)
	ADDQ  SI, R14
	DECQ  R11
	JNZ   pth8row
	RET

// func splatDc8AVX2(dst *uint8, stride, w, h int, dc uint8)
TEXT ·splatDc8AVX2(SB), NOSPLIT, $0-33
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ w+16(FP), CX
	MOVQ h+24(FP), R8

	VPBROADCASTB dc+32(FP), Y0

	CMPQ CX, $4
	JEQ  sp4
	CMPQ CX, $8
	JEQ  sp8
	CMPQ CX, $16
	JEQ  sp16
	CMPQ CX, $32
	JEQ  sp32

sp64:
	VMOVDQU Y0, (DI)
	VMOVDQU Y0, 32(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp64
	VZEROUPPER
	RET

sp4:
	VMOVD X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   sp4
	VZEROUPPER
	RET

sp8:
	VMOVQ X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   sp8
	VZEROUPPER
	RET

sp16:
	VMOVDQU X0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp16
	VZEROUPPER
	RET

sp32:
	VMOVDQU Y0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     sp32
	VZEROUPPER
	RET

// func vpred8AVX2(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·vpred8AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	CMPQ CX, $4
	JEQ  vp4
	CMPQ CX, $8
	JEQ  vp8
	CMPQ CX, $16
	JEQ  vp16
	CMPQ CX, $32
	JEQ  vp32

	VMOVDQU 1(DX), Y0
	VMOVDQU 33(DX), Y1

vp64:
	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp64
	VZEROUPPER
	RET

vp4:
	VMOVD 1(DX), X0

vp4row:
	VMOVD X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   vp4row
	RET

vp8:
	VMOVQ 1(DX), X0

vp8row:
	VMOVQ X0, (DI)
	ADDQ  SI, DI
	DECQ  R8
	JNZ   vp8row
	RET

vp16:
	VMOVDQU 1(DX), X0

vp16row:
	VMOVDQU X0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp16row
	RET

vp32:
	VMOVDQU 1(DX), Y0

vp32row:
	VMOVDQU Y0, (DI)
	ADDQ    SI, DI
	DECQ    R8
	JNZ     vp32row
	VZEROUPPER
	RET

// func hpred8AVX2(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·hpred8AVX2(SB), NOSPLIT, $0-40
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ h+32(FP), R8

	LEAQ -1(DX), R13

	CMPQ CX, $4
	JEQ  hp4
	CMPQ CX, $8
	JEQ  hp8
	CMPQ CX, $16
	JEQ  hp16
	CMPQ CX, $32
	JEQ  hp32

hp64:
	VPBROADCASTB (R13), Y0
	VMOVDQU      Y0, (DI)
	VMOVDQU      Y0, 32(DI)
	ADDQ         SI, DI
	DECQ         R13
	DECQ         R8
	JNZ          hp64
	VZEROUPPER
	RET

hp4:
	VPBROADCASTB (R13), X0
	VMOVD        X0, (DI)
	ADDQ         SI, DI
	DECQ         R13
	DECQ         R8
	JNZ          hp4
	RET

hp8:
	VPBROADCASTB (R13), X0
	VMOVQ        X0, (DI)
	ADDQ         SI, DI
	DECQ         R13
	DECQ         R8
	JNZ          hp8
	RET

hp16:
	VPBROADCASTB (R13), X0
	VMOVDQU      X0, (DI)
	ADDQ         SI, DI
	DECQ         R13
	DECQ         R8
	JNZ          hp16
	RET

hp32:
	VPBROADCASTB (R13), Y0
	VMOVDQU      Y0, (DI)
	ADDQ         SI, DI
	DECQ         R13
	DECQ         R8
	JNZ          hp32
	VZEROUPPER
	RET

DATA pw8<>+0(SB)/4, $0x00080008
GLOBL pw8<>(SB), RODATA|NOPTR, $4

DATA fiShuf<>+0(SB)/8, $0x0201020100060006
DATA fiShuf<>+8(SB)/8, $0x8005800504030403
GLOBL fiShuf<>(SB), RODATA|NOPTR, $16

// func filterIntra8AVX2(dst *uint8, stride int, tl *uint8, f *int8, w, h int)
TEXT ·filterIntra8AVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ tl+16(FP), DX
	MOVQ f+24(FP), R9
	MOVQ w+32(FP), CX
	MOVQ h+40(FP), R8

	VMOVDQU      (R9), X8
	VMOVDQU      16(R9), X9
	VMOVDQU      32(R9), X10
	VMOVDQU      48(R9), X11
	VMOVDQU      fiShuf<>(SB), X6
	VPBROADCASTD pw8<>(SB), X7

	LEAQ 1(DX), R12
	XORQ R11, R11

fiRow:
	MOVQ DI, R14
	LEAQ (DI)(SI*1), R15

	MOVQ DX, AX
	SUBQ R11, AX
	LEAQ -1(AX), BX
	LEAQ -2(AX), R13

	XORQ R10, R10

fiCol:
	VMOVD      (R12)(R10*1), X0
	VPINSRB    $4, (BX), X0, X0
	VPINSRB    $5, (R13), X0, X0
	VPINSRB    $6, (AX), X0, X0
	VPSHUFB    X6, X0, X0
	VPSHUFD    $0x00, X0, X1
	VPMADDUBSW X8, X1, X1
	VPSHUFD    $0x55, X0, X4
	VPMADDUBSW X9, X4, X4
	VPADDW     X7, X1, X1
	VPADDW     X4, X1, X1
	VPSHUFD    $0xAA, X0, X4
	VPMADDUBSW X10, X4, X4
	VPADDW     X4, X1, X1
	VPSHUFD    $0xFF, X0, X4
	VPMADDUBSW X11, X4, X4
	VPADDW     X4, X1, X1
	VPSRAW     $4, X1, X1
	VPACKUSWB  X1, X1, X1
	VMOVD      X1, (R14)(R10*1)
	VPEXTRD    $1, X1, (R15)(R10*1)

	LEAQ 3(R10), AX
	LEAQ (R14)(AX*1), BX
	LEAQ (R15)(AX*1), R13
	ADDQ R12, AX

	ADDQ $4, R10
	CMPQ R10, CX
	JLT  fiCol

	MOVQ R15, R12
	LEAQ (DI)(SI*2), DI
	ADDQ $2, R11
	CMPQ R11, R8
	JLT  fiRow
	RET

DATA pw32<>+0(SB)/4, $0x00200020
GLOBL pw32<>(SB), RODATA|NOPTR, $4

#define Z1_WEIGHTS         \
	MOVQ  R11, AX;         \
	ANDQ  $0x3E, AX;       \
	MOVQ  $64, BX;         \
	SUBQ  AX, BX;          \
	SHLQ  $8, AX;          \
	ORQ   BX, AX;          \
	VMOVD AX, X5;          \
	VPBROADCASTW X5, Y5;   \
	MOVQ  R11, R13;        \
	SARQ  $6, R13;         \
	LEAQ  (DX)(R13*1), BX

// func z1Full8AVX2(dst *uint8, stride int, top *uint8, w, rows, dx, xpos int)
TEXT ·z1Full8AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ stride+8(FP), SI
	MOVQ top+16(FP), DX
	MOVQ w+24(FP), CX
	MOVQ rows+32(FP), R8
	MOVQ dx+40(FP), R9
	MOVQ xpos+48(FP), R11

	VPBROADCASTD pw32<>(SB), Y7

	CMPQ CX, $4
	JEQ  z1w4
	CMPQ CX, $8
	JEQ  z1w8

z1w16:
	Z1_WEIGHTS
	MOVQ DI, R15
	XORQ R14, R14

z1w16col:
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
	VMOVDQU     X2, (R15)(R14*1)
	ADDQ        $16, R14
	CMPQ        R14, CX
	JLT         z1w16col

	ADDQ SI, DI
	ADDQ R9, R11
	DECQ R8
	JNZ  z1w16
	VZEROUPPER
	RET

z1w4:
	Z1_WEIGHTS
	VMOVD      (BX), X0
	VMOVD      1(BX), X1
	VPUNPCKLBW X1, X0, X2
	VPMADDUBSW X5, X2, X2
	VPADDW     X7, X2, X2
	VPSRLW     $6, X2, X2
	VPACKUSWB  X2, X2, X2
	VMOVD      X2, (DI)
	ADDQ       SI, DI
	ADDQ       R9, R11
	DECQ       R8
	JNZ        z1w4
	VZEROUPPER
	RET

z1w8:
	Z1_WEIGHTS
	VMOVQ      (BX), X0
	VMOVQ      1(BX), X1
	VPUNPCKLBW X1, X0, X2
	VPMADDUBSW X5, X2, X2
	VPADDW     X7, X2, X2
	VPSRLW     $6, X2, X2
	VPACKUSWB  X2, X2, X2
	VMOVQ      X2, (DI)
	ADDQ       SI, DI
	ADDQ       R9, R11
	DECQ       R8
	JNZ        z1w8
	VZEROUPPER
	RET
