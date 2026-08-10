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

DATA lpfWordInv<>+0(SB)/8, $0x0b0a030209080100
DATA lpfWordInv<>+8(SB)/8, $0x0f0e07060d0c0504
GLOBL lpfWordInv<>(SB), RODATA|NOPTR, $16

#define LW 16

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
#define SP6 -7*LW(BX)
#define SP5 -6*LW(BX)
#define SP4 -5*LW(BX)
#define SP3 -4*LW(BX)
#define SP2 -3*LW(BX)
#define SP1 -2*LW(BX)
#define SP0 -1*LW(BX)
#define SQ0 0(BX)
#define SQ1 1*LW(BX)
#define SQ2 2*LW(BX)
#define SQ3 3*LW(BX)
#define SQ4 4*LW(BX)
#define SQ5 5*LW(BX)
#define SQ6 6*LW(BX)

#define OP2 17*LW(BX)
#define OP1 18*LW(BX)
#define OP0 19*LW(BX)
#define OQ0 20*LW(BX)
#define OQ1 21*LW(BX)
#define OQ2 22*LW(BX)

#define MFLAT 29*LW(BX)
#define MHEV  30*LW(BX)
#define BYTES 31*LW(BX)

// V0 to V15 and the offsets above carry the lane width, so one filter body
// serves both the four line and the eight line kernels.
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
#define V10 X10
#define V11 X11
#define V12 X12
#define V13 X13
#define V14 X14
#define V15 X15
#define MSKALL $0xffff
#define KALL   $0x0f

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
// the doubled sample, V13 carries the rounding term and V14 the mask that
// keeps whatever the narrow filter already wrote.
#define WIDE8(A, B, C, D, F, G, K, DST) \
	VMOVDQU   A, V4;            \
	VPADDD    B, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    F, V4, V4;        \
	VPADDD    G, V4, V4;        \
	VPADDD    K, V4, V4;        \
	VPADDD    V13, V4, V4;      \
	VPSRAD    $3, V4, V4;       \
	VPBLENDVB V14, DST, V4, V4; \
	VMOVDQU   V4, DST

#define WIDE6(A, B, C, D, F, DST)   \
	VMOVDQU   A, V4;            \
	VPADDD    B, V4, V4;        \
	VPADDD    B, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    F, V4, V4;        \
	VPADDD    V13, V4, V4;      \
	VPSRAD    $3, V4, V4;       \
	VPBLENDVB V14, DST, V4, V4; \
	VMOVDQU   V4, DST

// The same three, with the lane masks in K registers. K1 holds the lanes left
// alone, K2 the high edge variance, K3 the flat lanes and K5 the complement of
// whichever of the two is current, so a blend becomes a masked store.
#define GTAK(A, B, THR, T, M) \
	VPSUBD   B, A, T; \
	VPABSD   T, T;    \
	VPCMPGTD THR, T, M

#define ACCGTK(A, B, THR, T, M, ACC) \
	GTAK(A, B, THR, T, M);       \
	KORW M, ACC, ACC

#define WIDE8K(A, B, C, D, F, G, K, DST) \
	VMOVDQU   A, V4;            \
	VPADDD    B, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    F, V4, V4;        \
	VPADDD    G, V4, V4;        \
	VPADDD    K, V4, V4;        \
	VPADDD    V13, V4, V4;      \
	VPSRAD    $3, V4, V4;       \
	VMOVDQU32 V4, K5, DST

#define WIDE6K(A, B, C, D, F, DST)  \
	VMOVDQU   A, V4;            \
	VPADDD    B, V4, V4;        \
	VPADDD    B, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    C, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    D, V4, V4;        \
	VPADDD    F, V4, V4;        \
	VPADDD    V13, V4, V4;      \
	VPSRAD    $3, V4, V4;       \
	VMOVDQU32 V4, K5, DST

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
	VMOVDQU      SP1, V0; \
	VMOVDQU      SP0, V1; \
	VMOVDQU      SQ0, V2; \
	VMOVDQU      SQ1, V3; \
	VPBROADCASTD I, V8; \
	VPBROADCASTD H, V10; \
	VPSUBD       V1, V0, V4; \
	VPABSD       V4, V4; \
	VPCMPGTD     V8, V4, V15; \
	VPCMPGTD     V10, V4, V14; \
	VPSUBD       V2, V3, V4; \
	VPABSD       V4, V4; \
	VPCMPGTD     V8, V4, V5; \
	VPOR         V5, V15, V15; \
	VPCMPGTD     V10, V4, V5; \
	VPOR         V5, V14, V14; \
	VMOVDQU      V14, MHEV; \
	VPSUBD       V2, V1, V4; \
	VPABSD       V4, V4; \
	VPADDD       V4, V4, V4; \
	VPSUBD       V3, V0, V5; \
	VPABSD       V5, V5; \
	VPSRLD       $1, V5, V5; \
	VPADDD       V5, V4, V4; \
	VPBROADCASTD E, V9; \
	VPCMPGTD     V9, V4, V5; \
	VPOR         V5, V15, V15; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  fmdone; \
	ACCGT(V0, SP2, V8, V5, V15); \
	ACCGT(V3, SQ2, V8, V5, V15); \
	CMPL AX, $6; \
	JLE  fmdone; \
	VMOVDQU SP3, V6; \
	ACCGT(V6, SP2, V8, V5, V15); \
	VMOVDQU SQ3, V6; \
	ACCGT(V6, SQ2, V8, V5, V15); \
	fmdone:; \
	VPMOVMSKB V15, AX; \
	CMPL      AX, MSKALL; \
	JEQ       scatter; \
	VMOVDQU      MHEV, V14; \
	VPBROADCASTD NEGLIM, V8; \
	VPBROADCASTD LIM1, V9; \
	VPSUBD       V3, V0, V4; \
	CLIPD(V4, V8, V9); \
	VPAND        V14, V4, V4; \
	VPSUBD       V1, V2, V6; \
	VPADDD       V6, V6, V7; \
	VPADDD       V6, V7, V7; \
	VPADDD       V7, V4, V4; \
	CLIPD(V4, V8, V9); \
	VPBROADCASTD C4, V10; \
	VPADDD       V10, V4, V6; \
	VPMINSD      V9, V6, V6; \
	VPSRAD       $3, V6, V6; \
	VPBROADCASTD C3, V10; \
	VPADDD       V10, V4, V7; \
	VPMINSD      V9, V7, V7; \
	VPSRAD       $3, V7, V7; \
	VPXOR        V8, V8, V8; \
	VPBROADCASTD PIXMAX, V9; \
	VPADDD       V7, V1, V10; \
	CLIPD(V10, V8, V9); \
	VPSUBD       V6, V2, V11; \
	CLIPD(V11, V8, V9); \
	VPBROADCASTD C1, V12; \
	VPADDD       V12, V6, V12; \
	VPSRAD       $1, V12, V12; \
	VPADDD       V12, V0, V4; \
	CLIPD(V4, V8, V9); \
	VPBLENDVB    V14, V0, V4, V4; \
	VPSUBD       V12, V3, V5; \
	CLIPD(V5, V8, V9); \
	VPBLENDVB    V14, V3, V5, V5; \
	VPBLENDVB    V15, V0, V4, V4; \
	VMOVDQU      V4, OP1; \
	VPBLENDVB    V15, V1, V10, V10; \
	VMOVDQU      V10, OP0; \
	VPBLENDVB    V15, V2, V11, V11; \
	VMOVDQU      V11, OQ0; \
	VPBLENDVB    V15, V3, V5, V5; \
	VMOVDQU      V5, OQ1; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  scatter; \
	VPBROADCASTD FTHR, V8; \
	VMOVDQU      V15, V14; \
	ACCGT(V1, SP2, V8, V5, V14); \
	ACCGT(V0, V1, V8, V5, V14); \
	ACCGT(V3, V2, V8, V5, V14); \
	ACCGT(V2, SQ2, V8, V5, V14); \
	CMPL         AX, $8; \
	JL           flatindone; \
	ACCGT(V1, SP3, V8, V5, V14); \
	ACCGT(V2, SQ3, V8, V5, V14); \
	flatindone:; \
	VMOVDQU   V14, MFLAT; \
	VPMOVMSKB V14, DX; \
	CMPL      DX, MSKALL; \
	JEQ       scatter; \
	VPBROADCASTD C4, V13; \
	CMPL         AX, $6; \
	JG           eight; \
	WIDE6(SP2, SP2, V0, V1, V2, OP1); \
	WIDE6(SP2, V0, V1, V2, V3, OP0); \
	WIDE6(V0, V1, V2, V3, SQ2, OQ0); \
	WIDE6(V1, V2, V3, SQ2, SQ2, OQ1); \
	JMP          scatter; \
	eight:; \
	WIDE8(SP3, SP3, SP3, SP2, V0, V1, V2, OP2); \
	WIDE8(SP3, SP3, SP2, V0, V1, V2, V3, OP1); \
	WIDE8(SP3, SP2, V0, V1, V2, V3, SQ2, OP0); \
	WIDE8(SP2, V0, V1, V2, V3, SQ2, SQ3, OQ0); \
	WIDE8(V0, V1, V2, V3, SQ2, SQ3, SQ3, OQ1); \
	WIDE8(V1, V2, V3, SQ2, SQ3, SQ3, SQ3, OQ2); \
	CMPL AX, $16; \
	JL   scatter; \
	VPBROADCASTD FTHR, V8; \
	ACCGT(V1, SP6, V8, V5, V14); \
	ACCGT(V1, SP5, V8, V5, V14); \
	ACCGT(V1, SP4, V8, V5, V14); \
	ACCGT(V2, SQ4, V8, V5, V14); \
	ACCGT(V2, SQ5, V8, V5, V14); \
	ACCGT(V2, SQ6, V8, V5, V14); \
	VPMOVMSKB    V14, DX; \
	CMPL         DX, $0xffff; \
	JEQ          scatter; \
	VMOVDQU SP6, V0; \
	VMOVDQU V0, -13*LW(BX); \
	VMOVDQU V0, -12*LW(BX); \
	VMOVDQU V0, -11*LW(BX); \
	VMOVDQU V0, -10*LW(BX); \
	VMOVDQU V0, -9*LW(BX); \
	VMOVDQU V0, -8*LW(BX); \
	VMOVDQU SQ6, V0; \
	VMOVDQU V0, 7*LW(BX); \
	VMOVDQU V0, 8*LW(BX); \
	VMOVDQU V0, 9*LW(BX); \
	VMOVDQU V0, 10*LW(BX); \
	VMOVDQU V0, 11*LW(BX); \
	VMOVDQU V0, 12*LW(BX); \
	LEAQ         -12*LW(BX), CX; \
	VMOVDQU      (CX), V4; \
	VPADDD       1*LW(CX), V4, V4; \
	VPADDD       2*LW(CX), V4, V4; \
	VPADDD       3*LW(CX), V4, V4; \
	VPADDD       4*LW(CX), V4, V4; \
	VPADDD       5*LW(CX), V4, V4; \
	VPADDD       6*LW(CX), V4, V4; \
	VPADDD       7*LW(CX), V4, V4; \
	VPADDD       8*LW(CX), V4, V4; \
	VPADDD       9*LW(CX), V4, V4; \
	VPADDD       10*LW(CX), V4, V4; \
	VPADDD       11*LW(CX), V4, V4; \
	VPADDD       12*LW(CX), V4, V4; \
	VPBROADCASTD C4, V13; \
	VPADDD       V13, V13, V13; \
	LEAQ         -6*LW(BX), CX; \
	LEAQ         14*LW(BX), DX; \
	MOVQ         $12, R14; \
	fourteen:; \
	VMOVDQU   -1*LW(CX), V5; \
	VPADDD    (CX), V5, V5; \
	VPADDD    1*LW(CX), V5, V5; \
	VPADDD    V4, V5, V5; \
	VPADDD    V13, V5, V5; \
	VPSRAD    $4, V5, V5; \
	VPBLENDVB V14, (DX), V5, V5; \
	VMOVDQU   V5, (DX); \
	VPADDD    7*LW(CX), V4, V4; \
	VPSUBD    -6*LW(CX), V4, V4; \
	ADDQ      $LW, CX; \
	ADDQ      $LW, DX; \
	DECQ      R14; \
	JNZ       fourteen;

// LPFBODY512 is LPFBODY with the lane masks in K registers, which turns the
// blends into masked stores and keeps the two masks out of memory.
#define LPFBODY512 \
	VMOVDQU      SP1, V0; \
	VMOVDQU      SP0, V1; \
	VMOVDQU      SQ0, V2; \
	VMOVDQU      SQ1, V3; \
	VPBROADCASTD I, V8; \
	VPBROADCASTD H, V10; \
	VPSUBD       V1, V0, V4; \
	VPABSD       V4, V4; \
	VPCMPGTD     V8, V4, K1; \
	VPCMPGTD     V10, V4, K2; \
	VPSUBD       V2, V3, V4; \
	VPABSD       V4, V4; \
	VPCMPGTD     V8, V4, K4; \
	KORW         K4, K1, K1; \
	VPCMPGTD     V10, V4, K4; \
	KORW         K4, K2, K2; \
	VPSUBD       V2, V1, V4; \
	VPABSD       V4, V4; \
	VPADDD       V4, V4, V4; \
	VPSUBD       V3, V0, V5; \
	VPABSD       V5, V5; \
	VPSRLD       $1, V5, V5; \
	VPADDD       V5, V4, V4; \
	VPBROADCASTD E, V9; \
	VPCMPGTD     V9, V4, K4; \
	KORW         K4, K1, K1; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  fmdone; \
	ACCGTK(V0, SP2, V8, V5, K4, K1); \
	ACCGTK(V3, SQ2, V8, V5, K4, K1); \
	CMPL AX, $6; \
	JLE  fmdone; \
	VMOVDQU SP3, V6; \
	ACCGTK(V6, SP2, V8, V5, K4, K1); \
	VMOVDQU SQ3, V6; \
	ACCGTK(V6, SQ2, V8, V5, K4, K1); \
	fmdone:; \
	KMOVW K1, AX; \
	CMPL  AX, KALL; \
	JEQ   scatter; \
	VPBROADCASTD NEGLIM, V8; \
	VPBROADCASTD LIM1, V9; \
	VPSUBD       V3, V0, V4; \
	CLIPD(V4, V8, V9); \
	VMOVDQU32.Z  V4, K2, V4; \
	VPSUBD       V1, V2, V6; \
	VPADDD       V6, V6, V7; \
	VPADDD       V6, V7, V7; \
	VPADDD       V7, V4, V4; \
	CLIPD(V4, V8, V9); \
	VPBROADCASTD C4, V10; \
	VPADDD       V10, V4, V6; \
	VPMINSD      V9, V6, V6; \
	VPSRAD       $3, V6, V6; \
	VPBROADCASTD C3, V10; \
	VPADDD       V10, V4, V7; \
	VPMINSD      V9, V7, V7; \
	VPSRAD       $3, V7, V7; \
	VPXOR        V8, V8, V8; \
	VPBROADCASTD PIXMAX, V9; \
	VPADDD       V7, V1, V10; \
	CLIPD(V10, V8, V9); \
	VPSUBD       V6, V2, V11; \
	CLIPD(V11, V8, V9); \
	VPBROADCASTD C1, V12; \
	VPADDD       V12, V6, V12; \
	VPSRAD       $1, V12, V12; \
	VPADDD       V12, V0, V4; \
	CLIPD(V4, V8, V9); \
	VPBLENDMD    V0, V4, K2, V4; \
	VPSUBD       V12, V3, V5; \
	CLIPD(V5, V8, V9); \
	VPBLENDMD    V3, V5, K2, V5; \
	KNOTW        K1, K5; \
	VMOVDQU32    V4, K5, OP1; \
	VMOVDQU32    V10, K5, OP0; \
	VMOVDQU32    V11, K5, OQ0; \
	VMOVDQU32    V5, K5, OQ1; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  scatter; \
	VPBROADCASTD FTHR, V8; \
	KMOVW        K1, K3; \
	ACCGTK(V1, SP2, V8, V5, K4, K3); \
	ACCGTK(V0, V1, V8, V5, K4, K3); \
	ACCGTK(V3, V2, V8, V5, K4, K3); \
	ACCGTK(V2, SQ2, V8, V5, K4, K3); \
	CMPL         AX, $8; \
	JL           flatindone; \
	ACCGTK(V1, SP3, V8, V5, K4, K3); \
	ACCGTK(V2, SQ3, V8, V5, K4, K3); \
	flatindone:; \
	KMOVW K3, DX; \
	CMPL  DX, KALL; \
	JEQ   scatter; \
	KNOTW        K3, K5; \
	VPBROADCASTD C4, V13; \
	CMPL         AX, $6; \
	JG           eight; \
	WIDE6K(SP2, SP2, V0, V1, V2, OP1); \
	WIDE6K(SP2, V0, V1, V2, V3, OP0); \
	WIDE6K(V0, V1, V2, V3, SQ2, OQ0); \
	WIDE6K(V1, V2, V3, SQ2, SQ2, OQ1); \
	JMP          scatter; \
	eight:; \
	WIDE8K(SP3, SP3, SP3, SP2, V0, V1, V2, OP2); \
	WIDE8K(SP3, SP3, SP2, V0, V1, V2, V3, OP1); \
	WIDE8K(SP3, SP2, V0, V1, V2, V3, SQ2, OP0); \
	WIDE8K(SP2, V0, V1, V2, V3, SQ2, SQ3, OQ0); \
	WIDE8K(V0, V1, V2, V3, SQ2, SQ3, SQ3, OQ1); \
	WIDE8K(V1, V2, V3, SQ2, SQ3, SQ3, SQ3, OQ2); \
	CMPL AX, $16; \
	JL   scatter; \
	VPBROADCASTD FTHR, V8; \
	ACCGTK(V1, SP6, V8, V5, K4, K3); \
	ACCGTK(V1, SP5, V8, V5, K4, K3); \
	ACCGTK(V1, SP4, V8, V5, K4, K3); \
	ACCGTK(V2, SQ4, V8, V5, K4, K3); \
	ACCGTK(V2, SQ5, V8, V5, K4, K3); \
	ACCGTK(V2, SQ6, V8, V5, K4, K3); \
	KMOVW        K3, DX; \
	CMPL         DX, KALL; \
	JEQ          scatter; \
	KNOTW   K3, K5; \
	VMOVDQU SP6, V0; \
	VMOVDQU V0, -13*LW(BX); \
	VMOVDQU V0, -12*LW(BX); \
	VMOVDQU V0, -11*LW(BX); \
	VMOVDQU V0, -10*LW(BX); \
	VMOVDQU V0, -9*LW(BX); \
	VMOVDQU V0, -8*LW(BX); \
	VMOVDQU SQ6, V0; \
	VMOVDQU V0, 7*LW(BX); \
	VMOVDQU V0, 8*LW(BX); \
	VMOVDQU V0, 9*LW(BX); \
	VMOVDQU V0, 10*LW(BX); \
	VMOVDQU V0, 11*LW(BX); \
	VMOVDQU V0, 12*LW(BX); \
	LEAQ         -12*LW(BX), CX; \
	VMOVDQU      (CX), V4; \
	VPADDD       1*LW(CX), V4, V4; \
	VPADDD       2*LW(CX), V4, V4; \
	VPADDD       3*LW(CX), V4, V4; \
	VPADDD       4*LW(CX), V4, V4; \
	VPADDD       5*LW(CX), V4, V4; \
	VPADDD       6*LW(CX), V4, V4; \
	VPADDD       7*LW(CX), V4, V4; \
	VPADDD       8*LW(CX), V4, V4; \
	VPADDD       9*LW(CX), V4, V4; \
	VPADDD       10*LW(CX), V4, V4; \
	VPADDD       11*LW(CX), V4, V4; \
	VPADDD       12*LW(CX), V4, V4; \
	VPBROADCASTD C4, V13; \
	VPADDD       V13, V13, V13; \
	LEAQ         -6*LW(BX), CX; \
	LEAQ         14*LW(BX), DX; \
	MOVQ         $12, R14; \
	fourteen:; \
	VMOVDQU   -1*LW(CX), V5; \
	VPADDD    (CX), V5, V5; \
	VPADDD    1*LW(CX), V5, V5; \
	VPADDD    V4, V5, V5; \
	VPADDD    V13, V5, V5; \
	VPSRAD    $4, V5, V5; \
	VMOVDQU32 V5, K5, (DX); \
	VPADDD    7*LW(CX), V4, V4; \
	VPSUBD    -6*LW(CX), V4, V4; \
	ADDQ      $LW, CX; \
	ADDQ      $LW, DX; \
	DECQ      R14; \
	JNZ       fourteen;

// LPF8W filters eight contiguous lines at a time. Only a horizontal edge
// reaches it, so the lines need no transpose and the taps are the only thing
// that strides.
#define LPF8W(FILTER) \
	MOVQ dst+0(FP), SI;       \
	MOVQ strideb+16(FP), R9;  \
	MOVQ p+24(FP), DI;        \
	LEAQ 13*LW(SP), BX;       \
	MOVL WD, AX;              \
	MOVQ $-2, R12;            \
	MOVQ $4, R13;             \
	CMPL AX, $4;              \
	JLE  spanset;             \
	MOVQ $-3, R12;            \
	MOVQ $6, R13;             \
	CMPL AX, $6;              \
	JLE  spanset;             \
	MOVQ $-4, R12;            \
	MOVQ $8, R13;             \
	CMPL AX, $16;             \
	JLT  spanset;             \
	MOVQ $-7, R12;            \
	MOVQ $14, R13;            \
	spanset:;                 \
	MOVQ  R12, AX;            \
	IMULQ R9, AX;             \
	LEAQ  (SI)(AX*1), CX;     \
	MOVQ  R12, AX;            \
	IMULQ $LW, AX;            \
	LEAQ  (BX)(AX*1), DX;     \
	MOVQ  R13, R14;           \
	widenloop:;               \
	VPMOVZXBD (CX), V0;       \
	VMOVDQU   V0, (DX);       \
	VMOVDQU   V0, 20*LW(DX);  \
	ADDQ      R9, CX;         \
	ADDQ      $LW, DX;        \
	DECQ      R14;            \
	JNZ       widenloop;      \
	FILTER;                   \
	scatter:;                 \
	MOVL WD, AX;              \
	MOVQ $-2, R12;            \
	MOVQ $4, R13;             \
	CMPL AX, $6;              \
	JLE  scatterset;          \
	MOVQ $-3, R12;            \
	MOVQ $6, R13;             \
	CMPL AX, $16;             \
	JLT  scatterset;          \
	MOVQ $-6, R12;            \
	MOVQ $12, R13;            \
	scatterset:;              \
	MOVQ  R12, AX;            \
	IMULQ R9, AX;             \
	LEAQ  (SI)(AX*1), CX;     \
	packloop:;                \
	MOVQ      R12, AX;        \
	IMULQ     $LW, AX;        \
	LEAQ      (BX)(AX*1), DX; \
	VMOVDQU   20*LW(DX), V0;  \
	VPACKUSDW V0, V0, V0;     \
	VPERMQ    $0x08, V0, V0;  \
	VPACKUSWB X0, X0, X0;     \
	VMOVQ     X0, (CX);       \
	ADDQ      R9, CX;         \
	INCQ      R12;            \
	DECQ      R13;            \
	JNZ       packloop;       \
	VZEROUPPER

// LPF16W is LPF8W at sixteen bits, where the samples need no widening on the
// way in and the pack has to clip explicitly on the way out.
#define LPF16W(FILTER) \
	MOVQ dst+0(FP), SI;       \
	MOVQ strideb+16(FP), R9;  \
	MOVQ p+24(FP), DI;        \
	LEAQ 13*LW(SP), BX;       \
	MOVL WD, AX;              \
	MOVQ $-2, R12;            \
	MOVQ $4, R13;             \
	CMPL AX, $4;              \
	JLE  spanset;             \
	MOVQ $-3, R12;            \
	MOVQ $6, R13;             \
	CMPL AX, $6;              \
	JLE  spanset;             \
	MOVQ $-4, R12;            \
	MOVQ $8, R13;             \
	CMPL AX, $16;             \
	JLT  spanset;             \
	MOVQ $-7, R12;            \
	MOVQ $14, R13;            \
	spanset:;                 \
	MOVQ  R12, AX;            \
	IMULQ R9, AX;             \
	LEAQ  (SI)(AX*2), CX;     \
	MOVQ  R12, AX;            \
	IMULQ $LW, AX;            \
	LEAQ  (BX)(AX*1), DX;     \
	MOVQ  R13, R14;           \
	widenloop:;               \
	VPMOVZXWD (CX), V0;       \
	VMOVDQU   V0, (DX);       \
	VMOVDQU   V0, 20*LW(DX);  \
	LEAQ      (CX)(R9*2), CX; \
	ADDQ      $LW, DX;        \
	DECQ      R14;            \
	JNZ       widenloop;      \
	FILTER;                   \
	scatter:;                 \
	MOVL WD, AX;              \
	MOVQ $-2, R12;            \
	MOVQ $4, R13;             \
	CMPL AX, $6;              \
	JLE  scatterset;          \
	MOVQ $-3, R12;            \
	MOVQ $6, R13;             \
	CMPL AX, $16;             \
	JLT  scatterset;          \
	MOVQ $-6, R12;            \
	MOVQ $12, R13;            \
	scatterset:;              \
	MOVQ  R12, AX;            \
	IMULQ R9, AX;             \
	LEAQ  (SI)(AX*2), CX;     \
	packloop:;                \
	MOVQ      R12, AX;        \
	IMULQ     $LW, AX;        \
	LEAQ      (BX)(AX*1), DX; \
	VMOVDQU   20*LW(DX), V0;  \
	VPACKUSDW V0, V0, V0;     \
	VPERMQ    $0x08, V0, V0;  \
	VMOVDQU   X0, (CX);       \
	LEAQ      (CX)(R9*2), CX; \
	INCQ      R12;            \
	DECQ      R13;            \
	JNZ       packloop;       \
	VZEROUPPER

// LPF8 and LPF16 gather the four lines, run FILTER over them and scatter the
// result back, so the two filter bodies share all of the plumbing.
#define LPF8(FILTER) \
	MOVQ dst+0(FP), SI; \
	MOVQ stridea+8(FP), R8; \
	MOVQ strideb+16(FP), R9; \
	MOVQ p+24(FP), DI; \
	LEAQ 208(SP), BX; \
	MOVL WD, AX; \
	MOVQ $-2, R12; \
	MOVQ $4, R13; \
	CMPL AX, $4; \
	JLE  spanset; \
	MOVQ $-3, R12; \
	MOVQ $6, R13; \
	CMPL AX, $6; \
	JLE  spanset; \
	MOVQ $-4, R12; \
	MOVQ $8, R13; \
	CMPL AX, $16; \
	JLT  spanset; \
	MOVQ $-7, R12; \
	MOVQ $14, R13; \
spanset:; \
	CMPQ R8, $1; \
	JEQ  gathercols; \
	/* The four lines are rows here, so read the span from each and transpose. */; \
	CMPL AX, $16; \
	JGE  gatherwide; \
	MOVQ $-4, R12; \
	MOVQ $8, R13; \
	VMOVQ       -4(SI), X0; \
	VMOVQ       -4(SI)(R8*1), X1; \
	VMOVQ       -4(SI)(R8*2), X2; \
	LEAQ        (R8)(R8*2), R10; \
	VMOVQ       -4(SI)(R10*1), X3; \
	VPUNPCKLDQ  X1, X0, X4; \
	VPUNPCKLDQ  X3, X2, X5; \
	VPUNPCKLQDQ X5, X4, X6; \
	VPUNPCKHQDQ X5, X4, X7; \
	VMOVDQU     lpfShuf<>(SB), X8; \
	VPSHUFB     X8, X6, X6; \
	VPSHUFB     X8, X7, X7; \
	VMOVDQU     X6, 508(BX); \
	VMOVDQU     X7, 524(BX); \
	JMP         widen; \
gatherwide:; \
	MOVQ        $-7, R12; \
	MOVQ        $16, R13; \
	VMOVDQU     -7(SI), X0; \
	VMOVDQU     -7(SI)(R8*1), X1; \
	VMOVDQU     -7(SI)(R8*2), X2; \
	LEAQ        (R8)(R8*2), R10; \
	VMOVDQU     -7(SI)(R10*1), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU     lpfShuf<>(SB), X8; \
	VPSHUFB     X8, X0, X0; \
	VPSHUFB     X8, X1, X1; \
	VPSHUFB     X8, X2, X2; \
	VPSHUFB     X8, X3, X3; \
	VMOVDQU     X0, 496(BX); \
	VMOVDQU     X1, 512(BX); \
	VMOVDQU     X2, 528(BX); \
	VMOVDQU     X3, 544(BX); \
	JMP         widen; \
gathercols:; \
	MOVQ  R12, AX; \
	IMULQ R9, AX; \
	LEAQ  (SI)(AX*1), CX; \
	LEAQ  BYTES, DX; \
	LEAQ  (DX)(R12*4), DX; \
	ADDQ  $28, DX; \
	MOVQ  R13, R14; \
gathercolsloop:; \
	MOVL (CX), AX; \
	MOVL AX, (DX); \
	ADDQ R9, CX; \
	ADDQ $4, DX; \
	DECQ R14; \
	JNZ  gathercolsloop; \
widen:; \
	LEAQ BYTES, CX; \
	LEAQ (CX)(R12*4), CX; \
	ADDQ $28, CX; \
	MOVQ R12, AX; \
	SHLQ $4, AX; \
	LEAQ (BX)(AX*1), DX; \
	MOVQ R13, R14; \
widenloop:; \
	VPMOVZXBD (CX), X0; \
	VMOVDQU   X0, (DX); \
	VMOVDQU   X0, 320(DX); \
	ADDQ      $4, CX; \
	ADDQ      $LW, DX; \
	DECQ      R14; \
	JNZ       widenloop; \
	FILTER; \
scatter:; \
	MOVL WD, AX; \
	MOVQ $-2, R12; \
	MOVQ $4, R13; \
	CMPL AX, $6; \
	JLE  scatterset; \
	MOVQ $-3, R12; \
	MOVQ $6, R13; \
	CMPL AX, $16; \
	JLT  scatterset; \
	MOVQ $-6, R12; \
	MOVQ $12, R13; \
scatterset:; \
	CMPQ R8, $1; \
	JEQ  scattercols; \
	LEAQ BYTES, CX; \
	LEAQ (CX)(R12*4), CX; \
	ADDQ $28, CX; \
	JMP  packloop; \
scattercols:; \
	MOVQ  R12, AX; \
	IMULQ R9, AX; \
	LEAQ  (SI)(AX*1), CX; \
packloop:; \
	MOVQ      R12, AX; \
	SHLQ      $4, AX; \
	LEAQ      (BX)(AX*1), DX; \
	ADDQ      $320, DX; \
	VMOVDQU   (DX), X0; \
	VPACKUSDW X0, X0, X0; \
	VPACKUSWB X0, X0, X0; \
	VMOVD     X0, AX; \
	MOVL      AX, (CX); \
	INCQ      R12; \
	CMPQ      R8, $1; \
	JEQ       packnextcol; \
	ADDQ      $4, CX; \
	JMP       packnext; \
packnextcol:; \
	ADDQ R9, CX; \
packnext:; \
	DECQ R13; \
	JNZ  packloop; \
	CMPQ R8, $1; \
	JEQ  done; \
	LEAQ    (R8)(R8*2), R10; \
	VMOVDQU lpfShuf<>(SB), X8; \
	MOVL    WD, AX; \
	CMPL    AX, $16; \
	JGE     scatterwide; \
	VMOVDQU    508(BX), X6; \
	VMOVDQU    524(BX), X7; \
	VPSHUFB    X8, X6, X6; \
	VPSHUFB    X8, X7, X7; \
	VPUNPCKLDQ X7, X6, X0; \
	VPUNPCKHDQ X7, X6, X1; \
	VMOVQ      X0, -4(SI); \
	VPEXTRQ    $1, X0, AX; \
	MOVQ       AX, -4(SI)(R8*1); \
	VMOVQ      X1, -4(SI)(R8*2); \
	VPEXTRQ    $1, X1, AX; \
	MOVQ       AX, -4(SI)(R10*1); \
	JMP        done; \
scatterwide:; \
	VMOVDQU 496(BX), X0; \
	VMOVDQU 512(BX), X1; \
	VMOVDQU 528(BX), X2; \
	VMOVDQU 544(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, -7(SI); \
	VMOVDQU X1, -7(SI)(R8*1); \
	VMOVDQU X2, -7(SI)(R8*2); \
	VMOVDQU X3, -7(SI)(R10*1); \
done:; \
	VZEROUPPER

#define LPF16(FILTER) \
	MOVQ dst+0(FP), SI; \
	MOVQ stridea+8(FP), R8; \
	MOVQ strideb+16(FP), R9; \
	MOVQ p+24(FP), DI; \
	LEAQ 216(SP), BX; \
	LEAQ (R8)(R8*2), R10; \
	MOVL WD, AX; \
	MOVQ $-2, R12; \
	MOVQ $4, R13; \
	CMPL AX, $4; \
	JLE  spanset; \
	MOVQ $-3, R12; \
	MOVQ $6, R13; \
	CMPL AX, $6; \
	JLE  spanset; \
	MOVQ $-4, R12; \
	MOVQ $8, R13; \
	CMPL AX, $16; \
	JLT  spanset; \
	MOVQ $-7, R12; \
	MOVQ $14, R13; \
spanset:; \
	CMPQ R8, $1; \
	JEQ  gathercols; \
	VMOVDQU lpfWord<>(SB), X8; \
	CMPL    AX, $16; \
	JGE     gatherwide; \
	MOVQ    $-4, R12; \
	MOVQ    $8, R13; \
	VMOVDQU -8(SI), X0; \
	VMOVDQU -8(SI)(R8*2), X1; \
	VMOVDQU -8(SI)(R8*4), X2; \
	VMOVDQU -8(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	VMOVDQU X0, 520(BX); \
	VMOVDQU X1, 536(BX); \
	VMOVDQU X2, 552(BX); \
	VMOVDQU X3, 568(BX); \
	JMP     widen; \
gatherwide:; \
	MOVQ    $-7, R12; \
	MOVQ    $16, R13; \
	VMOVDQU -14(SI), X0; \
	VMOVDQU -14(SI)(R8*2), X1; \
	VMOVDQU -14(SI)(R8*4), X2; \
	VMOVDQU -14(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	VMOVDQU X0, 496(BX); \
	VMOVDQU X1, 512(BX); \
	VMOVDQU X2, 528(BX); \
	VMOVDQU X3, 544(BX); \
	VMOVDQU 2(SI), X0; \
	VMOVDQU 2(SI)(R8*2), X1; \
	VMOVDQU 2(SI)(R8*4), X2; \
	VMOVDQU 2(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	VMOVDQU X0, 560(BX); \
	VMOVDQU X1, 576(BX); \
	VMOVDQU X2, 592(BX); \
	VMOVDQU X3, 608(BX); \
	JMP     widen; \
gathercols:; \
	MOVQ  R12, AX; \
	IMULQ R9, AX; \
	LEAQ  (SI)(AX*2), CX; \
	LEAQ  552(BX), DX; \
	LEAQ  (DX)(R12*8), DX; \
	MOVQ  R13, R14; \
gathercolsloop:; \
	MOVQ (CX), AX; \
	MOVQ AX, (DX); \
	LEAQ (CX)(R9*2), CX; \
	ADDQ $8, DX; \
	DECQ R14; \
	JNZ  gathercolsloop; \
widen:; \
	LEAQ 552(BX), CX; \
	LEAQ (CX)(R12*8), CX; \
	MOVQ R12, AX; \
	SHLQ $4, AX; \
	LEAQ (BX)(AX*1), DX; \
	MOVQ R13, R14; \
widenloop:; \
	VPMOVZXWD (CX), X0; \
	VMOVDQU   X0, (DX); \
	VMOVDQU   X0, 320(DX); \
	ADDQ      $8, CX; \
	ADDQ      $LW, DX; \
	DECQ      R14; \
	JNZ       widenloop; \
	FILTER; \
scatter:; \
	MOVL WD, AX; \
	MOVQ $-2, R12; \
	MOVQ $4, R13; \
	CMPL AX, $6; \
	JLE  scatterset; \
	MOVQ $-3, R12; \
	MOVQ $6, R13; \
	CMPL AX, $16; \
	JLT  scatterset; \
	MOVQ $-6, R12; \
	MOVQ $12, R13; \
scatterset:; \
	CMPQ R8, $1; \
	JEQ  scattercols; \
	LEAQ 552(BX), CX; \
	LEAQ (CX)(R12*8), CX; \
	JMP  packloop; \
scattercols:; \
	MOVQ  R12, AX; \
	IMULQ R9, AX; \
	LEAQ  (SI)(AX*2), CX; \
packloop:; \
	MOVQ      R12, AX; \
	SHLQ      $4, AX; \
	LEAQ      (BX)(AX*1), DX; \
	ADDQ      $320, DX; \
	VMOVDQU   (DX), X0; \
	VPACKUSDW X0, X0, X0; \
	VMOVQ     X0, (CX); \
	INCQ      R12; \
	CMPQ      R8, $1; \
	JEQ       packnextcol; \
	ADDQ      $8, CX; \
	JMP       packnext; \
packnextcol:; \
	LEAQ (CX)(R9*2), CX; \
packnext:; \
	DECQ R13; \
	JNZ  packloop; \
	CMPQ R8, $1; \
	JEQ  done; \
	VMOVDQU lpfWordInv<>(SB), X8; \
	MOVL    WD, AX; \
	CMPL    AX, $16; \
	JGE     scatterwide; \
	VMOVDQU 520(BX), X0; \
	VMOVDQU 536(BX), X1; \
	VMOVDQU 552(BX), X2; \
	VMOVDQU 568(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, -8(SI); \
	VMOVDQU X1, -8(SI)(R8*2); \
	VMOVDQU X2, -8(SI)(R8*4); \
	VMOVDQU X3, -8(SI)(R10*2); \
	JMP     done; \
scatterwide:; \
	VMOVDQU 496(BX), X0; \
	VMOVDQU 512(BX), X1; \
	VMOVDQU 528(BX), X2; \
	VMOVDQU 544(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, -14(SI); \
	VMOVDQU X1, -14(SI)(R8*2); \
	VMOVDQU X2, -14(SI)(R8*4); \
	VMOVDQU X3, -14(SI)(R10*2); \
	VMOVDQU 560(BX), X0; \
	VMOVDQU 576(BX), X1; \
	VMOVDQU 592(BX), X2; \
	VMOVDQU 608(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, 2(SI); \
	VMOVDQU X1, 2(SI)(R8*2); \
	VMOVDQU X2, 2(SI)(R8*4); \
	VMOVDQU X3, 2(SI)(R10*2); \
done:; \
	VZEROUPPER

// func lpf8AVX2(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8AVX2(SB), NOSPLIT, $768-32
	LPF8(LPFBODY)
	RET

// func lpf8AVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8AVX512(SB), NOSPLIT, $768-32
	LPF8(LPFBODY512)
	RET

// func lpf16AVX2(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16AVX2(SB), 0, $832-32
	LPF16(LPFBODY)
	RET

// func lpf16AVX512(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16AVX512(SB), 0, $832-32
	LPF16(LPFBODY512)
	RET

// The eight line kernels reuse every macro above at twice the lane width.
#undef LW
#define LW 32
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
#undef V10
#define V10 Y10
#undef V11
#define V11 Y11
#undef V12
#define V12 Y12
#undef V13
#define V13 Y13
#undef V14
#define V14 Y14
#undef V15
#define V15 Y15
#undef MSKALL
#define MSKALL $0xffffffff
#undef KALL
#define KALL   $0xff

// func lpf8WAVX2(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8WAVX2(SB), 0, $1408-32
	LPF8W(LPFBODY)
	RET

// func lpf8WAVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8WAVX512(SB), 0, $1408-32
	LPF8W(LPFBODY512)
	RET

// func lpf16WAVX2(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16WAVX2(SB), 0, $1408-32
	LPF16W(LPFBODY)
	RET

// func lpf16WAVX512(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16WAVX512(SB), 0, $1408-32
	LPF16W(LPFBODY512)
	RET
