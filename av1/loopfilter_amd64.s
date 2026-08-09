//go:build amd64 && !noasm

#include "textflag.h"

// The four lanes are the four lines of one edge. A vertical edge has its taps
// running along each line instead, so it is transposed on the way in and back
// out; the byte shuffle that does it is its own inverse.
DATA lpfShuf<>+0(SB)/8, $0x0d0905010c080400
DATA lpfShuf<>+8(SB)/8, $0x0f0b07030e0a0602
GLOBL lpfShuf<>(SB), RODATA|NOPTR, $16

// At sixteen bits the transpose deinterleaves words rather than bytes, and
// that permutation is not an involution, so the way back needs lpfShuf.
DATA lpfWord<>+0(SB)/8, $0x0d0c090805040100
DATA lpfWord<>+8(SB)/8, $0x0f0e0b0a07060302
GLOBL lpfWord<>(SB), RODATA|NOPTR, $16

#define E      0(DI)
#define I      4(DI)
#define H      8(DI)
#define FTHR   12(DI)
#define LIM1   16(DI)
#define NEGLIM 20(DI)
#define PIXMAX 24(DI)
#define C4     28(DI)
#define C3     32(DI)
#define C1     36(DI)
#define WD     40(DI)

// BX is the centre of the source array, which holds every sample widened to a
// lane and replicated past both ends for the sliding window.
#define SP6 -112(BX)
#define SP5 -96(BX)
#define SP4 -80(BX)
#define SP3 -64(BX)
#define SP2 -48(BX)
#define SP1 -32(BX)
#define SP0 -16(BX)
#define SQ0 0(BX)
#define SQ1 16(BX)
#define SQ2 32(BX)
#define SQ3 48(BX)
#define SQ4 64(BX)
#define SQ5 80(BX)
#define SQ6 96(BX)

#define OP2 272(BX)
#define OP1 288(BX)
#define OP0 304(BX)
#define OQ0 320(BX)
#define OQ1 336(BX)
#define OQ2 352(BX)

#define MFLAT 464(BX)
#define MHEV  480(BX)
#define BYTES 496(BX)

// GTA leaves set the lanes where |A-B| exceeds THR.
#define GTA(A, B, THR, T) \
	VPSUBD   B, A, T; \
	VPABSD   T, T;    \
	VPCMPGTD THR, T, T

#define ACCGT(A, B, THR, T, M) \
	GTA(A, B, THR, T);     \
	VPOR T, M, M

#define CLIPD(V, LO, HI)  \
	VPMAXSD LO, V, V; \
	VPMINSD HI, V, V

// WIDE8 and WIDE6 are one output each of the eight and six tap filters. D is
// the doubled sample, X13 carries the rounding term and X14 the mask that
// keeps whatever the narrow filter already wrote.
#define WIDE8(A, B, C, D, F, G, K, DST) \
	VMOVDQU   A, X4;            \
	VPADDD    B, X4, X4;        \
	VPADDD    C, X4, X4;        \
	VPADDD    D, X4, X4;        \
	VPADDD    D, X4, X4;        \
	VPADDD    F, X4, X4;        \
	VPADDD    G, X4, X4;        \
	VPADDD    K, X4, X4;        \
	VPADDD    X13, X4, X4;      \
	VPSRAD    $3, X4, X4;       \
	VPBLENDVB X14, DST, X4, X4; \
	VMOVDQU   X4, DST

#define WIDE6(A, B, C, D, F, DST)   \
	VMOVDQU   A, X4;            \
	VPADDD    B, X4, X4;        \
	VPADDD    B, X4, X4;        \
	VPADDD    C, X4, X4;        \
	VPADDD    C, X4, X4;        \
	VPADDD    D, X4, X4;        \
	VPADDD    D, X4, X4;        \
	VPADDD    F, X4, X4;        \
	VPADDD    X13, X4, X4;      \
	VPSRAD    $3, X4, X4;       \
	VPBLENDVB X14, DST, X4, X4; \
	VMOVDQU   X4, DST

#define TRANSPOSE4(A, B, C, D, T0, T1, T2, T3) \
	VPUNPCKLDQ  B, A, T0;  \
	VPUNPCKHDQ  B, A, T1;  \
	VPUNPCKLDQ  D, C, T2;  \
	VPUNPCKHDQ  D, C, T3;  \
	VPUNPCKLQDQ T2, T0, A; \
	VPUNPCKHQDQ T2, T0, B; \
	VPUNPCKLQDQ T3, T1, C; \
	VPUNPCKHQDQ T3, T1, D

// LPFBODY masks the lanes and layers the filters, working in 32-bit lanes
// throughout, which is why both bit depths share it.
#define LPFBODY \
	VMOVDQU      SP1, X0; \
	VMOVDQU      SP0, X1; \
	VMOVDQU      SQ0, X2; \
	VMOVDQU      SQ1, X3; \
	VPBROADCASTD I, X8; \
	VPBROADCASTD H, X10; \
	VPSUBD       X1, X0, X4; \
	VPABSD       X4, X4; \
	VPCMPGTD     X8, X4, X15; \
	VPCMPGTD     X10, X4, X14; \
	VPSUBD       X2, X3, X4; \
	VPABSD       X4, X4; \
	VPCMPGTD     X8, X4, X5; \
	VPOR         X5, X15, X15; \
	VPCMPGTD     X10, X4, X5; \
	VPOR         X5, X14, X14; \
	VMOVDQU      X14, MHEV; \
	VPSUBD       X2, X1, X4; \
	VPABSD       X4, X4; \
	VPADDD       X4, X4, X4; \
	VPSUBD       X3, X0, X5; \
	VPABSD       X5, X5; \
	VPSRLD       $1, X5, X5; \
	VPADDD       X5, X4, X4; \
	VPBROADCASTD E, X9; \
	VPCMPGTD     X9, X4, X5; \
	VPOR         X5, X15, X15; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  fmdone; \
	ACCGT(X0, SP2, X8, X5, X15); \
	ACCGT(X3, SQ2, X8, X5, X15); \
	CMPL AX, $6; \
	JLE  fmdone; \
	VMOVDQU SP3, X6; \
	ACCGT(X6, SP2, X8, X5, X15); \
	VMOVDQU SQ3, X6; \
	ACCGT(X6, SQ2, X8, X5, X15); \
	fmdone:; \
	VPMOVMSKB X15, AX; \
	CMPL      AX, $0xffff; \
	JEQ       scatter; \
	VMOVDQU      MHEV, X14; \
	VPBROADCASTD NEGLIM, X8; \
	VPBROADCASTD LIM1, X9; \
	VPSUBD       X3, X0, X4; \
	CLIPD(X4, X8, X9); \
	VPAND        X14, X4, X4; \
	VPSUBD       X1, X2, X6; \
	VPADDD       X6, X6, X7; \
	VPADDD       X6, X7, X7; \
	VPADDD       X7, X4, X4; \
	CLIPD(X4, X8, X9); \
	VPBROADCASTD C4, X10; \
	VPADDD       X10, X4, X6; \
	VPMINSD      X9, X6, X6; \
	VPSRAD       $3, X6, X6; \
	VPBROADCASTD C3, X10; \
	VPADDD       X10, X4, X7; \
	VPMINSD      X9, X7, X7; \
	VPSRAD       $3, X7, X7; \
	VPXOR        X8, X8, X8; \
	VPBROADCASTD PIXMAX, X9; \
	VPADDD       X7, X1, X10; \
	CLIPD(X10, X8, X9); \
	VPSUBD       X6, X2, X11; \
	CLIPD(X11, X8, X9); \
	VPBROADCASTD C1, X12; \
	VPADDD       X12, X6, X12; \
	VPSRAD       $1, X12, X12; \
	VPADDD       X12, X0, X4; \
	CLIPD(X4, X8, X9); \
	VPBLENDVB    X14, X0, X4, X4; \
	VPSUBD       X12, X3, X5; \
	CLIPD(X5, X8, X9); \
	VPBLENDVB    X14, X3, X5, X5; \
	VPBLENDVB    X15, X0, X4, X4; \
	VMOVDQU      X4, OP1; \
	VPBLENDVB    X15, X1, X10, X10; \
	VMOVDQU      X10, OP0; \
	VPBLENDVB    X15, X2, X11, X11; \
	VMOVDQU      X11, OQ0; \
	VPBLENDVB    X15, X3, X5, X5; \
	VMOVDQU      X5, OQ1; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  scatter; \
	VPBROADCASTD FTHR, X8; \
	VMOVDQU      X15, X14; \
	ACCGT(X1, SP2, X8, X5, X14); \
	ACCGT(X0, X1, X8, X5, X14); \
	ACCGT(X3, X2, X8, X5, X14); \
	ACCGT(X2, SQ2, X8, X5, X14); \
	CMPL         AX, $8; \
	JL           flatindone; \
	ACCGT(X1, SP3, X8, X5, X14); \
	ACCGT(X2, SQ3, X8, X5, X14); \
	flatindone:; \
	VMOVDQU   X14, MFLAT; \
	VPMOVMSKB X14, DX; \
	CMPL      DX, $0xffff; \
	JEQ       scatter; \
	VPBROADCASTD C4, X13; \
	CMPL         AX, $6; \
	JG           eight; \
	WIDE6(SP2, SP2, X0, X1, X2, OP1); \
	WIDE6(SP2, X0, X1, X2, X3, OP0); \
	WIDE6(X0, X1, X2, X3, SQ2, OQ0); \
	WIDE6(X1, X2, X3, SQ2, SQ2, OQ1); \
	JMP          scatter; \
	eight:; \
	WIDE8(SP3, SP3, SP3, SP2, X0, X1, X2, OP2); \
	WIDE8(SP3, SP3, SP2, X0, X1, X2, X3, OP1); \
	WIDE8(SP3, SP2, X0, X1, X2, X3, SQ2, OP0); \
	WIDE8(SP2, X0, X1, X2, X3, SQ2, SQ3, OQ0); \
	WIDE8(X0, X1, X2, X3, SQ2, SQ3, SQ3, OQ1); \
	WIDE8(X1, X2, X3, SQ2, SQ3, SQ3, SQ3, OQ2); \
	CMPL AX, $16; \
	JL   scatter; \
	VPBROADCASTD FTHR, X8; \
	ACCGT(X1, SP6, X8, X5, X14); \
	ACCGT(X1, SP5, X8, X5, X14); \
	ACCGT(X1, SP4, X8, X5, X14); \
	ACCGT(X2, SQ4, X8, X5, X14); \
	ACCGT(X2, SQ5, X8, X5, X14); \
	ACCGT(X2, SQ6, X8, X5, X14); \
	VPMOVMSKB    X14, DX; \
	CMPL         DX, $0xffff; \
	JEQ          scatter; \
	VMOVDQU SP6, X0; \
	VMOVDQU X0, -208(BX); \
	VMOVDQU X0, -192(BX); \
	VMOVDQU X0, -176(BX); \
	VMOVDQU X0, -160(BX); \
	VMOVDQU X0, -144(BX); \
	VMOVDQU X0, -128(BX); \
	VMOVDQU SQ6, X0; \
	VMOVDQU X0, 112(BX); \
	VMOVDQU X0, 128(BX); \
	VMOVDQU X0, 144(BX); \
	VMOVDQU X0, 160(BX); \
	VMOVDQU X0, 176(BX); \
	VMOVDQU X0, 192(BX); \
	LEAQ         -192(BX), CX; \
	VMOVDQU      (CX), X4; \
	VPADDD       16(CX), X4, X4; \
	VPADDD       32(CX), X4, X4; \
	VPADDD       48(CX), X4, X4; \
	VPADDD       64(CX), X4, X4; \
	VPADDD       80(CX), X4, X4; \
	VPADDD       96(CX), X4, X4; \
	VPADDD       112(CX), X4, X4; \
	VPADDD       128(CX), X4, X4; \
	VPADDD       144(CX), X4, X4; \
	VPADDD       160(CX), X4, X4; \
	VPADDD       176(CX), X4, X4; \
	VPADDD       192(CX), X4, X4; \
	VPBROADCASTD C4, X13; \
	VPADDD       X13, X13, X13; \
	LEAQ         -96(BX), CX; \
	LEAQ         224(BX), DX; \
	MOVQ         $12, R14; \
	fourteen:; \
	VMOVDQU   -16(CX), X5; \
	VPADDD    (CX), X5, X5; \
	VPADDD    16(CX), X5, X5; \
	VPADDD    X4, X5, X5; \
	VPADDD    X13, X5, X5; \
	VPSRAD    $4, X5, X5; \
	VPBLENDVB X14, (DX), X5, X5; \
	VMOVDQU   X5, (DX); \
	VPADDD    112(CX), X4, X4; \
	VPSUBD    -96(CX), X4, X4; \
	ADDQ      $16, CX; \
	ADDQ      $16, DX; \
	DECQ      R14; \
	JNZ       fourteen;

// func lpf8AVX2(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8AVX2(SB), NOSPLIT, $768-32
	MOVQ dst+0(FP), SI
	MOVQ stridea+8(FP), R8
	MOVQ strideb+16(FP), R9
	MOVQ p+24(FP), DI
	LEAQ 208(SP), BX

	MOVL WD, AX
	MOVQ $-2, R12
	MOVQ $4, R13
	CMPL AX, $4
	JLE  spanset
	MOVQ $-3, R12
	MOVQ $6, R13
	CMPL AX, $6
	JLE  spanset
	MOVQ $-4, R12
	MOVQ $8, R13
	CMPL AX, $16
	JLT  spanset
	MOVQ $-7, R12
	MOVQ $14, R13

spanset:
	CMPQ R8, $1
	JEQ  gathercols

	// The four lines are rows here, so read the span from each and transpose.
	CMPL AX, $16
	JGE  gatherwide
	MOVQ $-4, R12
	MOVQ $8, R13
	VMOVQ       -4(SI), X0
	VMOVQ       -4(SI)(R8*1), X1
	VMOVQ       -4(SI)(R8*2), X2
	LEAQ        (R8)(R8*2), R10
	VMOVQ       -4(SI)(R10*1), X3
	VPUNPCKLDQ  X1, X0, X4
	VPUNPCKLDQ  X3, X2, X5
	VPUNPCKLQDQ X5, X4, X6
	VPUNPCKHQDQ X5, X4, X7
	VMOVDQU     lpfShuf<>(SB), X8
	VPSHUFB     X8, X6, X6
	VPSHUFB     X8, X7, X7
	VMOVDQU     X6, 508(BX)
	VMOVDQU     X7, 524(BX)
	JMP         widen

gatherwide:
	MOVQ        $-7, R12
	MOVQ        $16, R13
	VMOVDQU     -7(SI), X0
	VMOVDQU     -7(SI)(R8*1), X1
	VMOVDQU     -7(SI)(R8*2), X2
	LEAQ        (R8)(R8*2), R10
	VMOVDQU     -7(SI)(R10*1), X3
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7)
	VMOVDQU     lpfShuf<>(SB), X8
	VPSHUFB     X8, X0, X0
	VPSHUFB     X8, X1, X1
	VPSHUFB     X8, X2, X2
	VPSHUFB     X8, X3, X3
	VMOVDQU     X0, 496(BX)
	VMOVDQU     X1, 512(BX)
	VMOVDQU     X2, 528(BX)
	VMOVDQU     X3, 544(BX)
	JMP         widen

gathercols:
	MOVQ  R12, AX
	IMULQ R9, AX
	LEAQ  (SI)(AX*1), CX
	LEAQ  BYTES, DX
	LEAQ  (DX)(R12*4), DX
	ADDQ  $28, DX
	MOVQ  R13, R14

gathercolsloop:
	MOVL (CX), AX
	MOVL AX, (DX)
	ADDQ R9, CX
	ADDQ $4, DX
	DECQ R14
	JNZ  gathercolsloop

widen:
	LEAQ BYTES, CX
	LEAQ (CX)(R12*4), CX
	ADDQ $28, CX
	MOVQ R12, AX
	SHLQ $4, AX
	LEAQ (BX)(AX*1), DX
	MOVQ R13, R14

widenloop:
	VPMOVZXBD (CX), X0
	VMOVDQU   X0, (DX)
	VMOVDQU   X0, 320(DX)
	ADDQ      $4, CX
	ADDQ      $16, DX
	DECQ      R14
	JNZ       widenloop

	LPFBODY

scatter:
	MOVL WD, AX
	MOVQ $-2, R12
	MOVQ $4, R13
	CMPL AX, $6
	JLE  scatterset
	MOVQ $-3, R12
	MOVQ $6, R13
	CMPL AX, $16
	JLT  scatterset
	MOVQ $-6, R12
	MOVQ $12, R13

scatterset:
	CMPQ R8, $1
	JEQ  scattercols

	LEAQ BYTES, CX
	LEAQ (CX)(R12*4), CX
	ADDQ $28, CX
	JMP  packloop

scattercols:
	MOVQ  R12, AX
	IMULQ R9, AX
	LEAQ  (SI)(AX*1), CX

packloop:
	MOVQ      R12, AX
	SHLQ      $4, AX
	LEAQ      (BX)(AX*1), DX
	ADDQ      $320, DX
	VMOVDQU   (DX), X0
	VPACKUSDW X0, X0, X0
	VPACKUSWB X0, X0, X0
	VMOVD     X0, AX
	MOVL      AX, (CX)
	INCQ      R12
	CMPQ      R8, $1
	JEQ       packnextcol
	ADDQ      $4, CX
	JMP       packnext

packnextcol:
	ADDQ R9, CX

packnext:
	DECQ R13
	JNZ  packloop

	CMPQ R8, $1
	JEQ  done

	LEAQ    (R8)(R8*2), R10
	VMOVDQU lpfShuf<>(SB), X8
	MOVL    WD, AX
	CMPL    AX, $16
	JGE     scatterwide

	VMOVDQU    508(BX), X6
	VMOVDQU    524(BX), X7
	VPSHUFB    X8, X6, X6
	VPSHUFB    X8, X7, X7
	VPUNPCKLDQ X7, X6, X0
	VPUNPCKHDQ X7, X6, X1
	VMOVQ      X0, -4(SI)
	VPEXTRQ    $1, X0, AX
	MOVQ       AX, -4(SI)(R8*1)
	VMOVQ      X1, -4(SI)(R8*2)
	VPEXTRQ    $1, X1, AX
	MOVQ       AX, -4(SI)(R10*1)
	JMP        done

scatterwide:
	VMOVDQU 496(BX), X0
	VMOVDQU 512(BX), X1
	VMOVDQU 528(BX), X2
	VMOVDQU 544(BX), X3
	VPSHUFB X8, X0, X0
	VPSHUFB X8, X1, X1
	VPSHUFB X8, X2, X2
	VPSHUFB X8, X3, X3
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7)
	VMOVDQU X0, -7(SI)
	VMOVDQU X1, -7(SI)(R8*1)
	VMOVDQU X2, -7(SI)(R8*2)
	VMOVDQU X3, -7(SI)(R10*1)

done:
	VZEROUPPER
	RET

