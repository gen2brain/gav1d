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

// Sixteen lines transpose four rows at a time, which leaves each quarter of a
// tap group in its own quadword; this gathers the four quarters back together.
DATA lpfPerm16<>+0(SB)/4, $0
DATA lpfPerm16<>+4(SB)/4, $4
DATA lpfPerm16<>+8(SB)/4, $8
DATA lpfPerm16<>+12(SB)/4, $12
DATA lpfPerm16<>+16(SB)/4, $1
DATA lpfPerm16<>+20(SB)/4, $5
DATA lpfPerm16<>+24(SB)/4, $9
DATA lpfPerm16<>+28(SB)/4, $13
DATA lpfPerm16<>+32(SB)/4, $2
DATA lpfPerm16<>+36(SB)/4, $6
DATA lpfPerm16<>+40(SB)/4, $10
DATA lpfPerm16<>+44(SB)/4, $14
DATA lpfPerm16<>+48(SB)/4, $3
DATA lpfPerm16<>+52(SB)/4, $7
DATA lpfPerm16<>+56(SB)/4, $11
DATA lpfPerm16<>+60(SB)/4, $15
GLOBL lpfPerm16<>(SB), RODATA|NOPTR, $64

// The sixteen bit transpose lands two taps to a quadword rather than four to a
// doubleword, so it needs its own gather and its own way back.
DATA lpfPermW<>+0(SB)/4, $0
DATA lpfPermW<>+4(SB)/4, $1
DATA lpfPermW<>+8(SB)/4, $4
DATA lpfPermW<>+12(SB)/4, $5
DATA lpfPermW<>+16(SB)/4, $8
DATA lpfPermW<>+20(SB)/4, $9
DATA lpfPermW<>+24(SB)/4, $12
DATA lpfPermW<>+28(SB)/4, $13
DATA lpfPermW<>+32(SB)/4, $2
DATA lpfPermW<>+36(SB)/4, $3
DATA lpfPermW<>+40(SB)/4, $6
DATA lpfPermW<>+44(SB)/4, $7
DATA lpfPermW<>+48(SB)/4, $10
DATA lpfPermW<>+52(SB)/4, $11
DATA lpfPermW<>+56(SB)/4, $14
DATA lpfPermW<>+60(SB)/4, $15
GLOBL lpfPermW<>(SB), RODATA|NOPTR, $64

DATA lpfPermWInv<>+0(SB)/4, $0
DATA lpfPermWInv<>+4(SB)/4, $1
DATA lpfPermWInv<>+8(SB)/4, $8
DATA lpfPermWInv<>+12(SB)/4, $9
DATA lpfPermWInv<>+16(SB)/4, $2
DATA lpfPermWInv<>+20(SB)/4, $3
DATA lpfPermWInv<>+24(SB)/4, $10
DATA lpfPermWInv<>+28(SB)/4, $11
DATA lpfPermWInv<>+32(SB)/4, $4
DATA lpfPermWInv<>+36(SB)/4, $5
DATA lpfPermWInv<>+40(SB)/4, $12
DATA lpfPermWInv<>+44(SB)/4, $13
DATA lpfPermWInv<>+48(SB)/4, $6
DATA lpfPermWInv<>+52(SB)/4, $7
DATA lpfPermWInv<>+56(SB)/4, $14
DATA lpfPermWInv<>+60(SB)/4, $15
GLOBL lpfPermWInv<>(SB), RODATA|NOPTR, $64

#define LW 16
#define MOVU VMOVDQU
#define PXOR VPXOR

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
	MOVU      A, V4;            \
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
	MOVU      V4, DST

#define WIDE6(A, B, C, D, F, DST)   \
	MOVU      A, V4;            \
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
	MOVU      V4, DST

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
	MOVU      A, V4;            \
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
	MOVU      A, V4;            \
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
	MOVU         SP1, V0; \
	MOVU         SP0, V1; \
	MOVU         SQ0, V2; \
	MOVU         SQ1, V3; \
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
	MOVU         V14, MHEV; \
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
	MOVU SP3, V6; \
	ACCGT(V6, SP2, V8, V5, V15); \
	MOVU SQ3, V6; \
	ACCGT(V6, SQ2, V8, V5, V15); \
	fmdone:; \
	VPMOVMSKB V15, AX; \
	CMPL      AX, MSKALL; \
	JEQ       scatter; \
	MOVU         MHEV, V14; \
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
	PXOR         V8, V8, V8; \
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
	MOVU         V4, OP1; \
	VPBLENDVB    V15, V1, V10, V10; \
	MOVU         V10, OP0; \
	VPBLENDVB    V15, V2, V11, V11; \
	MOVU         V11, OQ0; \
	VPBLENDVB    V15, V3, V5, V5; \
	MOVU         V5, OQ1; \
	MOVL WD, AX; \
	CMPL AX, $4; \
	JLE  scatter; \
	VPBROADCASTD FTHR, V8; \
	MOVU         V15, V14; \
	ACCGT(V1, SP2, V8, V5, V14); \
	ACCGT(V0, V1, V8, V5, V14); \
	ACCGT(V3, V2, V8, V5, V14); \
	ACCGT(V2, SQ2, V8, V5, V14); \
	CMPL         AX, $8; \
	JL           flatindone; \
	ACCGT(V1, SP3, V8, V5, V14); \
	ACCGT(V2, SQ3, V8, V5, V14); \
	flatindone:; \
	MOVU      V14, MFLAT; \
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
	CMPL         DX, MSKALL; \
	JEQ          scatter; \
	MOVU SP6, V0; \
	MOVU V0, -13*LW(BX); \
	MOVU V0, -12*LW(BX); \
	MOVU V0, -11*LW(BX); \
	MOVU V0, -10*LW(BX); \
	MOVU V0, -9*LW(BX); \
	MOVU V0, -8*LW(BX); \
	MOVU SQ6, V0; \
	MOVU V0, 7*LW(BX); \
	MOVU V0, 8*LW(BX); \
	MOVU V0, 9*LW(BX); \
	MOVU V0, 10*LW(BX); \
	MOVU V0, 11*LW(BX); \
	MOVU V0, 12*LW(BX); \
	LEAQ         -12*LW(BX), CX; \
	MOVU         (CX), V4; \
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
	MOVU      -1*LW(CX), V5; \
	VPADDD    (CX), V5, V5; \
	VPADDD    1*LW(CX), V5, V5; \
	VPADDD    V4, V5, V5; \
	VPADDD    V13, V5, V5; \
	VPSRAD    $4, V5, V5; \
	VPBLENDVB V14, (DX), V5, V5; \
	MOVU      V5, (DX); \
	VPADDD    7*LW(CX), V4, V4; \
	VPSUBD    -6*LW(CX), V4, V4; \
	ADDQ      $LW, CX; \
	ADDQ      $LW, DX; \
	DECQ      R14; \
	JNZ       fourteen;

// LPFBODY512 is LPFBODY with the lane masks in K registers, which turns the
// blends into masked stores and keeps the two masks out of memory.
#define LPFBODY512 \
	MOVU         SP1, V0; \
	MOVU         SP0, V1; \
	MOVU         SQ0, V2; \
	MOVU         SQ1, V3; \
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
	MOVU SP3, V6; \
	ACCGTK(V6, SP2, V8, V5, K4, K1); \
	MOVU SQ3, V6; \
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
	PXOR         V8, V8, V8; \
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
	MOVU SP6, V0; \
	MOVU V0, -13*LW(BX); \
	MOVU V0, -12*LW(BX); \
	MOVU V0, -11*LW(BX); \
	MOVU V0, -10*LW(BX); \
	MOVU V0, -9*LW(BX); \
	MOVU V0, -8*LW(BX); \
	MOVU SQ6, V0; \
	MOVU V0, 7*LW(BX); \
	MOVU V0, 8*LW(BX); \
	MOVU V0, 9*LW(BX); \
	MOVU V0, 10*LW(BX); \
	MOVU V0, 11*LW(BX); \
	MOVU V0, 12*LW(BX); \
	LEAQ         -12*LW(BX), CX; \
	MOVU         (CX), V4; \
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
	MOVU      -1*LW(CX), V5; \
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
#define PACKST            \
	MOVU      20*LW(DX), V0;  \
	VPACKUSDW V0, V0, V0;     \
	VPERMQ    $0x08, V0, V0;  \
	VPACKUSWB X0, X0, X0;     \
	VMOVQ     X0, (CX)

#define PACKST16          \
	MOVU      20*LW(DX), V0; \
	VPACKUSDW V0, V0, V0;    \
	VPERMQ    $0x08, V0, V0; \
	MOVU      X0, (CX)

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
	MOVU      V0, (DX);       \
	MOVU      V0, 20*LW(DX);  \
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
	PACKST;                   \
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
	MOVU      V0, (DX);       \
	MOVU      V0, 20*LW(DX);  \
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
	PACKST16;                 \
	LEAQ      (CX)(R9*2), CX; \
	INCQ      R12;            \
	DECQ      R13;            \
	JNZ       packloop;       \
	VZEROUPPER

// TRANSPOSE8B turns eight rows of eight bytes into eight taps of eight lines,
// two taps to a register. It is its own inverse, so the scatter reuses it.
#define TRANSPOSE8B(R0, R1, R2, R3, R4, R5, R6, R7, D0, D1, D2, D3) \
	VPUNPCKLBW R1, R0, D0; \
	VPUNPCKLBW R3, R2, D1; \
	VPUNPCKLBW R5, R4, D2; \
	VPUNPCKLBW R7, R6, D3; \
	VPUNPCKLWD D1, D0, R0; \
	VPUNPCKHWD D1, D0, R1; \
	VPUNPCKLWD D3, D2, R2; \
	VPUNPCKHWD D3, D2, R3; \
	VPUNPCKLDQ R2, R0, D0; \
	VPUNPCKHDQ R2, R0, D1; \
	VPUNPCKLDQ R3, R1, D2; \
	VPUNPCKHDQ R3, R1, D3

// LPF8HW filters eight rows of a vertical edge. The taps run along each row, so
// the lines are transposed on the way in and back out; only the narrower tap
// widths come here, which fixes the span at eight.
#define LPF8HW(FILTER) \
	MOVQ dst+0(FP), SI;        \
	MOVQ stridea+8(FP), R8;    \
	MOVQ p+24(FP), DI;         \
	LEAQ 13*LW(SP), BX;        \
	MOVQ $-4, R12;             \
	MOVQ $8, R13;              \
	LEAQ (R8)(R8*2), R10;      \
	VMOVQ -4(SI), X0;          \
	VMOVQ -4(SI)(R8*1), X1;    \
	VMOVQ -4(SI)(R8*2), X2;    \
	VMOVQ -4(SI)(R10*1), X3;   \
	LEAQ (SI)(R8*4), CX;       \
	VMOVQ -4(CX), X4;          \
	VMOVQ -4(CX)(R8*1), X5;    \
	VMOVQ -4(CX)(R8*2), X6;    \
	VMOVQ -4(CX)(R10*1), X7;   \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11); \
	MOVU X8, 31*LW+0(BX);   \
	MOVU X9, 31*LW+16(BX);  \
	MOVU X10, 31*LW+32(BX); \
	MOVU X11, 31*LW+48(BX); \
	LEAQ  31*LW(BX), CX;       \
	MOVQ  R12, AX;             \
	IMULQ $LW, AX;             \
	LEAQ  (BX)(AX*1), DX;      \
	MOVQ  R13, R14;            \
	widenloop:;                \
	VPMOVZXBD (CX), V0;        \
	MOVU      V0, (DX);        \
	MOVU      V0, 20*LW(DX);   \
	ADDQ      $8, CX;          \
	ADDQ      $LW, DX;         \
	DECQ      R14;             \
	JNZ       widenloop;       \
	FILTER;                    \
	scatter:;                  \
	LEAQ  31*LW(BX), CX;       \
	MOVQ  $-4, AX;             \
	IMULQ $LW, AX;             \
	LEAQ  (BX)(AX*1), DX;      \
	MOVQ  $8, R14;             \
	packloop:;                 \
	MOVU      20*LW(DX), V0;   \
	VPACKUSDW V0, V0, V0;      \
	VPERMQ    $0x08, V0, V0;   \
	VPACKUSWB X0, X0, X0;      \
	VMOVQ     X0, (CX);        \
	ADDQ      $8, CX;          \
	ADDQ      $LW, DX;         \
	DECQ      R14;             \
	JNZ       packloop;        \
	VMOVQ 31*LW+0(BX), X0;     \
	VMOVQ 31*LW+8(BX), X1;     \
	VMOVQ 31*LW+16(BX), X2;    \
	VMOVQ 31*LW+24(BX), X3;    \
	VMOVQ 31*LW+32(BX), X4;    \
	VMOVQ 31*LW+40(BX), X5;    \
	VMOVQ 31*LW+48(BX), X6;    \
	VMOVQ 31*LW+56(BX), X7;    \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11); \
	LEAQ (R8)(R8*2), R10;      \
	VMOVQ   X8, -4(SI);        \
	VPEXTRQ $1, X8, AX;        \
	MOVQ    AX, -4(SI)(R8*1);  \
	VMOVQ   X9, -4(SI)(R8*2);  \
	VPEXTRQ $1, X9, AX;        \
	MOVQ    AX, -4(SI)(R10*1); \
	LEAQ (SI)(R8*4), CX;       \
	VMOVQ   X10, -4(CX);       \
	VPEXTRQ $1, X10, AX;       \
	MOVQ    AX, -4(CX)(R8*1);  \
	VMOVQ   X11, -4(CX)(R8*2); \
	VPEXTRQ $1, X11, AX;       \
	MOVQ    AX, -4(CX)(R10*1); \
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
	MOVU        lpfShuf<>(SB), X8; \
	VPSHUFB     X8, X6, X6; \
	VPSHUFB     X8, X7, X7; \
	MOVU        X6, 508(BX); \
	MOVU        X7, 524(BX); \
	JMP         widen; \
gatherwide:; \
	MOVQ        $-7, R12; \
	MOVQ        $16, R13; \
	MOVU        -7(SI), X0; \
	MOVU        -7(SI)(R8*1), X1; \
	MOVU        -7(SI)(R8*2), X2; \
	LEAQ        (R8)(R8*2), R10; \
	MOVU        -7(SI)(R10*1), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	MOVU        lpfShuf<>(SB), X8; \
	VPSHUFB     X8, X0, X0; \
	VPSHUFB     X8, X1, X1; \
	VPSHUFB     X8, X2, X2; \
	VPSHUFB     X8, X3, X3; \
	MOVU        X0, 496(BX); \
	MOVU        X1, 512(BX); \
	MOVU        X2, 528(BX); \
	MOVU        X3, 544(BX); \
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
	MOVU      X0, (DX); \
	MOVU      X0, 320(DX); \
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
	MOVU      (DX), X0; \
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
	MOVU lpfShuf<>(SB), X8; \
	MOVL    WD, AX; \
	CMPL    AX, $16; \
	JGE     scatterwide; \
	MOVU       508(BX), X6; \
	MOVU       524(BX), X7; \
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
	MOVU 496(BX), X0; \
	MOVU 512(BX), X1; \
	MOVU 528(BX), X2; \
	MOVU 544(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	MOVU X0, -7(SI); \
	MOVU X1, -7(SI)(R8*1); \
	MOVU X2, -7(SI)(R8*2); \
	MOVU X3, -7(SI)(R10*1); \
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
	MOVU lpfWord<>(SB), X8; \
	CMPL    AX, $16; \
	JGE     gatherwide; \
	MOVQ    $-4, R12; \
	MOVQ    $8, R13; \
	MOVU -8(SI), X0; \
	MOVU -8(SI)(R8*2), X1; \
	MOVU -8(SI)(R8*4), X2; \
	MOVU -8(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	MOVU X0, 520(BX); \
	MOVU X1, 536(BX); \
	MOVU X2, 552(BX); \
	MOVU X3, 568(BX); \
	JMP     widen; \
gatherwide:; \
	MOVQ    $-7, R12; \
	MOVQ    $16, R13; \
	MOVU -14(SI), X0; \
	MOVU -14(SI)(R8*2), X1; \
	MOVU -14(SI)(R8*4), X2; \
	MOVU -14(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	MOVU X0, 496(BX); \
	MOVU X1, 512(BX); \
	MOVU X2, 528(BX); \
	MOVU X3, 544(BX); \
	MOVU 2(SI), X0; \
	MOVU 2(SI)(R8*2), X1; \
	MOVU 2(SI)(R8*4), X2; \
	MOVU 2(SI)(R10*2), X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	MOVU X0, 560(BX); \
	MOVU X1, 576(BX); \
	MOVU X2, 592(BX); \
	MOVU X3, 608(BX); \
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
	MOVU      X0, (DX); \
	MOVU      X0, 320(DX); \
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
	MOVU      (DX), X0; \
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
	MOVU lpfWordInv<>(SB), X8; \
	MOVL    WD, AX; \
	CMPL    AX, $16; \
	JGE     scatterwide; \
	MOVU 520(BX), X0; \
	MOVU 536(BX), X1; \
	MOVU 552(BX), X2; \
	MOVU 568(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	MOVU X0, -8(SI); \
	MOVU X1, -8(SI)(R8*2); \
	MOVU X2, -8(SI)(R8*4); \
	MOVU X3, -8(SI)(R10*2); \
	JMP     done; \
scatterwide:; \
	MOVU 496(BX), X0; \
	MOVU 512(BX), X1; \
	MOVU 528(BX), X2; \
	MOVU 544(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	MOVU X0, -14(SI); \
	MOVU X1, -14(SI)(R8*2); \
	MOVU X2, -14(SI)(R8*4); \
	MOVU X3, -14(SI)(R10*2); \
	MOVU 560(BX), X0; \
	MOVU 576(BX), X1; \
	MOVU 592(BX), X2; \
	MOVU 608(BX), X3; \
	VPSHUFB X8, X0, X0; \
	VPSHUFB X8, X1, X1; \
	VPSHUFB X8, X2, X2; \
	VPSHUFB X8, X3, X3; \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	MOVU X0, 2(SI); \
	MOVU X1, 2(SI)(R8*2); \
	MOVU X2, 2(SI)(R8*4); \
	MOVU X3, 2(SI)(R10*2); \
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

// func lpf8HWAVX2(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8HWAVX2(SB), 0, $1536-32
	LPF8HW(LPFBODY)
	RET

// func lpf8HWAVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8HWAVX512(SB), 0, $1536-32
	LPF8HW(LPFBODY512)
	RET

// The zmm kernel takes sixteen lines. LPF8W is written in terms of LW, so only
// the width, the registers and the pack change. AVX-512 has no untyped move or
// xor, and packs a dword down to a byte in one instruction. Nothing may follow
// these redefinitions.
#undef LW
#define LW 64
#undef V0
#define V0 Z0
#undef V1
#define V1 Z1
#undef V2
#define V2 Z2
#undef V3
#define V3 Z3
#undef V4
#define V4 Z4
#undef V5
#define V5 Z5
#undef V6
#define V6 Z6
#undef V7
#define V7 Z7
#undef V8
#define V8 Z8
#undef V9
#define V9 Z9
#undef V10
#define V10 Z10
#undef V11
#define V11 Z11
#undef V12
#define V12 Z12
#undef V13
#define V13 Z13
#undef V14
#define V14 Z14
#undef V15
#define V15 Z15
#undef MOVU
#define MOVU VMOVDQU32
#undef PXOR
#define PXOR VPXORD
#undef KALL
#define KALL $0xffff
#undef PACKST
#define PACKST               \
	MOVU      20*LW(DX), V0; \
	VPMOVUSDB V0, (CX)
#undef PACKST16
#define PACKST16             \
	MOVU      20*LW(DX), V0; \
	VPMOVUSDW V0, (CX)

// func lpf8ZAVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8ZAVX512(SB), 0, $2816-32
	LPF8W(LPFBODY512)
	RET

// func lpf16ZAVX512(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16ZAVX512(SB), 0, $2816-32
	LPF16W(LPFBODY512)
	RET

// LPF8HZ is LPF8HW over sixteen rows. Two eight by eight transposes give each
// tap as two halves, and one qword interleave joins them into a full lane.
#define JOIN8B(A0, A1, A2, A3, B0, B1, B2, B3, D0, D1, D2, D3, D4, D5, D6, D7) \
	VPUNPCKLQDQ B0, A0, D0; \
	VPUNPCKHQDQ B0, A0, D1; \
	VPUNPCKLQDQ B1, A1, D2; \
	VPUNPCKHQDQ B1, A1, D3; \
	VPUNPCKLQDQ B2, A2, D4; \
	VPUNPCKHQDQ B2, A2, D5; \
	VPUNPCKLQDQ B3, A3, D6; \
	VPUNPCKHQDQ B3, A3, D7

#define LPF8HZ(FILTER) \
	MOVQ dst+0(FP), SI;        \
	MOVQ stridea+8(FP), R8;    \
	MOVQ p+24(FP), DI;         \
	LEAQ 13*LW(SP), BX;        \
	LEAQ (R8)(R8*2), R10;      \
	VMOVQ -4(SI), X0;          \
	VMOVQ -4(SI)(R8*1), X1;    \
	VMOVQ -4(SI)(R8*2), X2;    \
	VMOVQ -4(SI)(R10*1), X3;   \
	LEAQ (SI)(R8*4), CX;       \
	VMOVQ -4(CX), X4;          \
	VMOVQ -4(CX)(R8*1), X5;    \
	VMOVQ -4(CX)(R8*2), X6;    \
	VMOVQ -4(CX)(R10*1), X7;   \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11); \
	LEAQ (CX)(R8*4), CX;       \
	VMOVQ -4(CX), X0;          \
	VMOVQ -4(CX)(R8*1), X1;    \
	VMOVQ -4(CX)(R8*2), X2;    \
	VMOVQ -4(CX)(R10*1), X3;   \
	LEAQ (CX)(R8*4), CX;       \
	VMOVQ -4(CX), X4;          \
	VMOVQ -4(CX)(R8*1), X5;    \
	VMOVQ -4(CX)(R8*2), X6;    \
	VMOVQ -4(CX)(R10*1), X7;   \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X12, X13, X14, X15); \
	JOIN8B(X8, X9, X10, X11, X12, X13, X14, X15, X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, 31*LW+0(BX);   \
	VMOVDQU X1, 31*LW+16(BX);  \
	VMOVDQU X2, 31*LW+32(BX);  \
	VMOVDQU X3, 31*LW+48(BX);  \
	VMOVDQU X4, 31*LW+64(BX);  \
	VMOVDQU X5, 31*LW+80(BX);  \
	VMOVDQU X6, 31*LW+96(BX);  \
	VMOVDQU X7, 31*LW+112(BX); \
	LEAQ  31*LW(BX), CX;       \
	LEAQ  -4*LW(BX), DX;       \
	MOVQ  $8, R14;             \
	widenloop:;                \
	VPMOVZXBD (CX), V0;        \
	MOVU      V0, (DX);        \
	MOVU      V0, 20*LW(DX);   \
	ADDQ      $16, CX;         \
	ADDQ      $LW, DX;         \
	DECQ      R14;             \
	JNZ       widenloop;       \
	FILTER;                    \
	scatter:;                  \
	LEAQ  31*LW(BX), CX;       \
	LEAQ  -4*LW(BX), DX;       \
	MOVQ  $8, R14;             \
	packloop:;                 \
	MOVU      20*LW(DX), V0;   \
	VPMOVUSDB V0, (CX);        \
	ADDQ      $16, CX;         \
	ADDQ      $LW, DX;         \
	DECQ      R14;             \
	JNZ       packloop;        \
	VMOVQ 31*LW+0(BX), X0;     \
	VMOVQ 31*LW+16(BX), X1;    \
	VMOVQ 31*LW+32(BX), X2;    \
	VMOVQ 31*LW+48(BX), X3;    \
	VMOVQ 31*LW+64(BX), X4;    \
	VMOVQ 31*LW+80(BX), X5;    \
	VMOVQ 31*LW+96(BX), X6;    \
	VMOVQ 31*LW+112(BX), X7;   \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11); \
	VMOVQ 31*LW+8(BX), X0;     \
	VMOVQ 31*LW+24(BX), X1;    \
	VMOVQ 31*LW+40(BX), X2;    \
	VMOVQ 31*LW+56(BX), X3;    \
	VMOVQ 31*LW+72(BX), X4;    \
	VMOVQ 31*LW+88(BX), X5;    \
	VMOVQ 31*LW+104(BX), X6;   \
	VMOVQ 31*LW+120(BX), X7;   \
	TRANSPOSE8B(X0, X1, X2, X3, X4, X5, X6, X7, X12, X13, X14, X15); \
	LEAQ (R8)(R8*2), R10;      \
	VMOVQ   X8, -4(SI);        \
	VPEXTRQ $1, X8, AX;        \
	MOVQ    AX, -4(SI)(R8*1);  \
	VMOVQ   X9, -4(SI)(R8*2);  \
	VPEXTRQ $1, X9, AX;        \
	MOVQ    AX, -4(SI)(R10*1); \
	LEAQ (SI)(R8*4), CX;       \
	VMOVQ   X10, -4(CX);       \
	VPEXTRQ $1, X10, AX;       \
	MOVQ    AX, -4(CX)(R8*1);  \
	VMOVQ   X11, -4(CX)(R8*2); \
	VPEXTRQ $1, X11, AX;       \
	MOVQ    AX, -4(CX)(R10*1); \
	LEAQ (CX)(R8*4), CX;       \
	VMOVQ   X12, -4(CX);       \
	VPEXTRQ $1, X12, AX;       \
	MOVQ    AX, -4(CX)(R8*1);  \
	VMOVQ   X13, -4(CX)(R8*2); \
	VPEXTRQ $1, X13, AX;       \
	MOVQ    AX, -4(CX)(R10*1); \
	LEAQ (CX)(R8*4), CX;       \
	VMOVQ   X14, -4(CX);       \
	VPEXTRQ $1, X14, AX;       \
	MOVQ    AX, -4(CX)(R8*1);  \
	VMOVQ   X15, -4(CX)(R8*2); \
	VPEXTRQ $1, X15, AX;       \
	MOVQ    AX, -4(CX)(R10*1); \
	VZEROUPPER

// func lpf8HZAVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8HZAVX512(SB), 0, $2944-32
	LPF8HZ(LPFBODY512)
	RET

// The fourteen taps fill a whole register, so a vertical edge cannot be
// transposed a quadword at a time. TRIN takes four rows to four quarter taps
// and TROUT takes four quarter taps back to four rows; the two are the same
// sixteen by sixteen transpose read in opposite directions.
#define TRIN(GOUT) \
	VMOVDQU -7(CX), X0;                         \
	VMOVDQU -7(CX)(R8*1), X1;                   \
	VMOVDQU -7(CX)(R8*2), X2;                   \
	VMOVDQU -7(CX)(R10*1), X3;                  \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0;                         \
	VPSHUFB X8, X1, X1;                         \
	VPSHUFB X8, X2, X2;                         \
	VPSHUFB X8, X3, X3;                         \
	VMOVDQU X0, 31*LW+GOUT+0(BX);               \
	VMOVDQU X1, 31*LW+GOUT+64(BX);              \
	VMOVDQU X2, 31*LW+GOUT+128(BX);             \
	VMOVDQU X3, 31*LW+GOUT+192(BX);             \
	LEAQ (CX)(R8*4), CX

#define TROUT(GIN, GOUT) \
	VMOVDQU 31*LW+GIN+0(BX), X0;                \
	VMOVDQU 31*LW+GIN+16(BX), X1;               \
	VMOVDQU 31*LW+GIN+32(BX), X2;               \
	VMOVDQU 31*LW+GIN+48(BX), X3;               \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0;                         \
	VPSHUFB X8, X1, X1;                         \
	VPSHUFB X8, X2, X2;                         \
	VPSHUFB X8, X3, X3;                         \
	VMOVDQU X0, 35*LW+GOUT+0(BX);               \
	VMOVDQU X1, 35*LW+GOUT+64(BX);              \
	VMOVDQU X2, 35*LW+GOUT+128(BX);             \
	VMOVDQU X3, 35*LW+GOUT+192(BX)

#define STOREROWS4(Z, X, R) \
	VMOVDQU       X, -7(R);           \
	VEXTRACTI32X4 $1, Z, X4;          \
	VMOVDQU       X4, -7(R)(R8*1);    \
	VEXTRACTI32X4 $2, Z, X5;          \
	VMOVDQU       X5, -7(R)(R8*2);    \
	VEXTRACTI32X4 $3, Z, X6;          \
	VMOVDQU       X6, -7(R)(R10*1)

#define LPF8HZ14(FILTER) \
	MOVQ dst+0(FP), SI;            \
	MOVQ stridea+8(FP), R8;        \
	MOVQ p+24(FP), DI;             \
	LEAQ 13*LW(SP), BX;            \
	LEAQ (R8)(R8*2), R10;          \
	VMOVDQU32 lpfPerm16<>(SB), Z16; \
	VMOVDQU   lpfShuf<>(SB), X8;   \
	MOVQ SI, CX;                   \
	TRIN(0);                       \
	TRIN(16);                      \
	TRIN(32);                      \
	TRIN(48);                      \
	VPERMD    31*LW+0(BX), Z16, Z0;   \
	VPERMD    31*LW+64(BX), Z16, Z1;  \
	VPERMD    31*LW+128(BX), Z16, Z2; \
	VPERMD    31*LW+192(BX), Z16, Z3; \
	MOVU      Z0, 31*LW+0(BX);        \
	MOVU      Z1, 31*LW+64(BX);       \
	MOVU      Z2, 31*LW+128(BX);      \
	MOVU      Z3, 31*LW+192(BX);      \
	LEAQ  31*LW(BX), CX;           \
	LEAQ  -7*LW(BX), DX;           \
	MOVQ  $14, R14;                \
	widenloop:;                    \
	VPMOVZXBD (CX), V0;            \
	MOVU      V0, (DX);            \
	MOVU      V0, 20*LW(DX);       \
	ADDQ      $16, CX;             \
	ADDQ      $LW, DX;             \
	DECQ      R14;                 \
	JNZ       widenloop;           \
	FILTER;                        \
	scatter:;                      \
	LEAQ  31*LW+16(BX), CX;        \
	LEAQ  -6*LW(BX), DX;           \
	MOVQ  $12, R14;                \
	packloop:;                     \
	MOVU      20*LW(DX), V0;       \
	VPMOVUSDB V0, (CX);            \
	ADDQ      $16, CX;             \
	ADDQ      $LW, DX;             \
	DECQ      R14;                 \
	JNZ       packloop;            \
	VMOVDQU lpfShuf<>(SB), X8;     \
	TROUT(0, 0);                   \
	TROUT(64, 16);                 \
	TROUT(128, 32);                \
	TROUT(192, 48);                \
	VPERMD    35*LW+0(BX), Z16, Z0;   \
	VPERMD    35*LW+64(BX), Z16, Z1;  \
	VPERMD    35*LW+128(BX), Z16, Z2; \
	VPERMD    35*LW+192(BX), Z16, Z3; \
	STOREROWS4(Z0, X0, SI);        \
	LEAQ (SI)(R8*4), CX;           \
	STOREROWS4(Z1, X1, CX);        \
	LEAQ (CX)(R8*4), CX;           \
	STOREROWS4(Z2, X2, CX);        \
	LEAQ (CX)(R8*4), CX;           \
	STOREROWS4(Z3, X3, CX);        \
	VZEROUPPER

// func lpf8HZ14AVX512(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8HZ14AVX512(SB), 0, $3328-32
	LPF8HZ14(LPFBODY512)
	RET

// At sixteen bits a row of taps is a whole register even at eight taps, so both
// widths transpose in quarters. TRINW turns four rows into four tap pairs and
// TROUTW turns four tap pairs back into four rows.
#define TRINW4(BOFF, GOUT) \
	VMOVDQU BOFF(CX), X0;                       \
	VMOVDQU BOFF(CX)(R8*2), X1;                 \
	VMOVDQU BOFF(CX)(R8*4), X2;                 \
	VMOVDQU BOFF(CX)(R10*2), X3;                \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VPSHUFB X8, X0, X0;                         \
	VPSHUFB X8, X1, X1;                         \
	VPSHUFB X8, X2, X2;                         \
	VPSHUFB X8, X3, X3;                         \
	VMOVDQU X0, 31*LW+GOUT+0(BX);               \
	VMOVDQU X1, 31*LW+GOUT+64(BX);              \
	VMOVDQU X2, 31*LW+GOUT+128(BX);             \
	VMOVDQU X3, 31*LW+GOUT+192(BX);             \
	LEAQ (CX)(R8*8), CX

#define TROUTW4(GIN, BOFF) \
	VMOVDQU 31*LW+GIN+0(BX), X0;                \
	VMOVDQU 31*LW+GIN+64(BX), X1;               \
	VMOVDQU 31*LW+GIN+128(BX), X2;              \
	VMOVDQU 31*LW+GIN+192(BX), X3;              \
	VPSHUFB X8, X0, X0;                         \
	VPSHUFB X8, X1, X1;                         \
	VPSHUFB X8, X2, X2;                         \
	VPSHUFB X8, X3, X3;                         \
	TRANSPOSE4(X0, X1, X2, X3, X4, X5, X6, X7); \
	VMOVDQU X0, BOFF(CX);                       \
	VMOVDQU X1, BOFF(CX)(R8*2);                 \
	VMOVDQU X2, BOFF(CX)(R8*4);                 \
	VMOVDQU X3, BOFF(CX)(R10*2);                \
	LEAQ (CX)(R8*8), CX

#define GATHERW(P, A, B, C, D) \
	VPERMD 31*LW+A(BX), P, Z0; \
	VPERMD 31*LW+B(BX), P, Z1; \
	VPERMD 31*LW+C(BX), P, Z2; \
	VPERMD 31*LW+D(BX), P, Z3; \
	MOVU   Z0, 31*LW+A(BX);    \
	MOVU   Z1, 31*LW+B(BX);    \
	MOVU   Z2, 31*LW+C(BX);    \
	MOVU   Z3, 31*LW+D(BX)

#define LPF16HZ(FILTER) \
	MOVQ dst+0(FP), SI;               \
	MOVQ stridea+8(FP), R8;           \
	MOVQ p+24(FP), DI;                \
	LEAQ 13*LW(SP), BX;               \
	LEAQ (R8)(R8*2), R10;             \
	VMOVDQU32 lpfPermW<>(SB), Z16;    \
	VMOVDQU32 lpfPermWInv<>(SB), Z17; \
	VMOVDQU   lpfWord<>(SB), X8;      \
	MOVQ SI, CX;                      \
	TRINW4(-8, 0);                    \
	TRINW4(-8, 16);                   \
	TRINW4(-8, 32);                   \
	TRINW4(-8, 48);                   \
	GATHERW(Z16, 0, 64, 128, 192);    \
	LEAQ  31*LW(BX), CX;              \
	LEAQ  -4*LW(BX), DX;              \
	MOVQ  $8, R14;                    \
	widenloop:;                       \
	VPMOVZXWD (CX), V0;               \
	MOVU      V0, (DX);               \
	MOVU      V0, 20*LW(DX);          \
	ADDQ      $32, CX;                \
	ADDQ      $LW, DX;                \
	DECQ      R14;                    \
	JNZ       widenloop;              \
	FILTER;                           \
	scatter:;                         \
	LEAQ  31*LW(BX), CX;              \
	LEAQ  -4*LW(BX), DX;              \
	MOVQ  $8, R14;                    \
	packloop:;                        \
	MOVU      20*LW(DX), V0;          \
	VPMOVUSDW V0, (CX);               \
	ADDQ      $32, CX;                \
	ADDQ      $LW, DX;                \
	DECQ      R14;                    \
	JNZ       packloop;               \
	GATHERW(Z17, 0, 64, 128, 192);    \
	VMOVDQU lpfWordInv<>(SB), X8;     \
	MOVQ SI, CX;                      \
	TROUTW4(0, -8);                   \
	TROUTW4(16, -8);                  \
	TROUTW4(32, -8);                  \
	TROUTW4(48, -8);                  \
	VZEROUPPER

#define LPF16HZ14(FILTER) \
	MOVQ dst+0(FP), SI;               \
	MOVQ stridea+8(FP), R8;           \
	MOVQ p+24(FP), DI;                \
	LEAQ 13*LW(SP), BX;               \
	LEAQ (R8)(R8*2), R10;             \
	VMOVDQU32 lpfPermW<>(SB), Z16;    \
	VMOVDQU32 lpfPermWInv<>(SB), Z17; \
	VMOVDQU   lpfWord<>(SB), X8;      \
	MOVQ SI, CX;                      \
	TRINW4(-14, 0);                   \
	TRINW4(-14, 16);                  \
	TRINW4(-14, 32);                  \
	TRINW4(-14, 48);                  \
	MOVQ SI, CX;                      \
	TRINW4(2, 256);                   \
	TRINW4(2, 272);                   \
	TRINW4(2, 288);                   \
	TRINW4(2, 304);                   \
	GATHERW(Z16, 0, 64, 128, 192);    \
	GATHERW(Z16, 256, 320, 384, 448); \
	LEAQ  31*LW(BX), CX;              \
	LEAQ  -7*LW(BX), DX;              \
	MOVQ  $14, R14;                   \
	widenloop:;                       \
	VPMOVZXWD (CX), V0;               \
	MOVU      V0, (DX);               \
	MOVU      V0, 20*LW(DX);          \
	ADDQ      $32, CX;                \
	ADDQ      $LW, DX;                \
	DECQ      R14;                    \
	JNZ       widenloop;              \
	FILTER;                           \
	scatter:;                         \
	LEAQ  31*LW(BX), CX;              \
	LEAQ  -7*LW(BX), DX;              \
	MOVQ  $14, R14;                   \
	packloop:;                        \
	MOVU      20*LW(DX), V0;          \
	VPMOVUSDW V0, (CX);               \
	ADDQ      $32, CX;                \
	ADDQ      $LW, DX;                \
	DECQ      R14;                    \
	JNZ       packloop;               \
	GATHERW(Z17, 0, 64, 128, 192);    \
	GATHERW(Z17, 256, 320, 384, 448); \
	VMOVDQU lpfWordInv<>(SB), X8;     \
	MOVQ SI, CX;                      \
	TROUTW4(0, -14);                  \
	TROUTW4(16, -14);                 \
	TROUTW4(32, -14);                 \
	TROUTW4(48, -14);                 \
	MOVQ SI, CX;                      \
	TROUTW4(256, 2);                  \
	TROUTW4(272, 2);                  \
	TROUTW4(288, 2);                  \
	TROUTW4(304, 2);                  \
	VZEROUPPER

// func lpf16HZAVX512(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16HZAVX512(SB), 0, $3072-32
	LPF16HZ(LPFBODY512)
	RET

// func lpf16HZ14AVX512(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16HZ14AVX512(SB), 0, $3328-32
	LPF16HZ14(LPFBODY512)
	RET
