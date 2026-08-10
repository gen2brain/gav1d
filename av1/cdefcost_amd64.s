//go:build amd64 && !noasm

#include "textflag.h"

// Every direction cost is a weighted sum of squares, so the divide table folds
// into one weight per lane. The diagonal weight runs out and back because the
// pair at n and 14-n share a divisor, and the lone middle tap takes the 105
// that the horizontal and vertical sums use throughout.
DATA cdefW105<>+0(SB)/4, $105
DATA cdefW105<>+4(SB)/4, $105
DATA cdefW105<>+8(SB)/4, $105
DATA cdefW105<>+12(SB)/4, $105
DATA cdefW105<>+16(SB)/4, $105
DATA cdefW105<>+20(SB)/4, $105
DATA cdefW105<>+24(SB)/4, $105
DATA cdefW105<>+28(SB)/4, $105
GLOBL cdefW105<>(SB), RODATA|NOPTR, $32

DATA cdefWDiag<>+0(SB)/4, $840
DATA cdefWDiag<>+4(SB)/4, $420
DATA cdefWDiag<>+8(SB)/4, $280
DATA cdefWDiag<>+12(SB)/4, $210
DATA cdefWDiag<>+16(SB)/4, $168
DATA cdefWDiag<>+20(SB)/4, $140
DATA cdefWDiag<>+24(SB)/4, $120
DATA cdefWDiag<>+28(SB)/4, $105
DATA cdefWDiag<>+32(SB)/4, $120
DATA cdefWDiag<>+36(SB)/4, $140
DATA cdefWDiag<>+40(SB)/4, $168
DATA cdefWDiag<>+44(SB)/4, $210
DATA cdefWDiag<>+48(SB)/4, $280
DATA cdefWDiag<>+52(SB)/4, $420
DATA cdefWDiag<>+56(SB)/4, $840
DATA cdefWDiag<>+60(SB)/4, $0
GLOBL cdefWDiag<>(SB), RODATA|NOPTR, $64

DATA cdefWAlt<>+0(SB)/4, $420
DATA cdefWAlt<>+4(SB)/4, $210
DATA cdefWAlt<>+8(SB)/4, $140
DATA cdefWAlt<>+12(SB)/4, $105
DATA cdefWAlt<>+16(SB)/4, $105
DATA cdefWAlt<>+20(SB)/4, $105
DATA cdefWAlt<>+24(SB)/4, $105
DATA cdefWAlt<>+28(SB)/4, $105
DATA cdefWAlt<>+32(SB)/4, $140
DATA cdefWAlt<>+36(SB)/4, $210
DATA cdefWAlt<>+40(SB)/4, $420
DATA cdefWAlt<>+44(SB)/4, $0
DATA cdefWAlt<>+48(SB)/4, $0
DATA cdefWAlt<>+52(SB)/4, $0
DATA cdefWAlt<>+56(SB)/4, $0
DATA cdefWAlt<>+60(SB)/4, $0
GLOBL cdefWAlt<>(SB), RODATA|NOPTR, $64

#define REDUCE(IDX)              \
	VEXTRACTI128 $1, Y0, X1; \
	VPADDD  X1, X0, X0;      \
	VPSHUFD $0x4E, X0, X1;   \
	VPADDD  X1, X0, X0;      \
	VPSHUFD $0xB1, X0, X1;   \
	VPADDD  X1, X0, X0;      \
	VMOVD   X0, AX;          \
	MOVL    AX, IDX*4(DI)

#define COST8(OFF, W, IDX)     \
	VMOVDQU OFF(SI), Y0;   \
	VPMULLD Y0, Y0, Y0;    \
	VPMULLD W, Y0, Y0;     \
	REDUCE(IDX)

#define COST16(OFF, WLO, WHI, IDX) \
	VMOVDQU OFF(SI), Y0;       \
	VPMULLD Y0, Y0, Y0;        \
	VPMULLD WLO, Y0, Y0;       \
	VMOVDQU OFF+32(SI), Y2;    \
	VPMULLD Y2, Y2, Y2;        \
	VPMULLD WHI, Y2, Y2;       \
	VPADDD  Y2, Y0, Y0;        \
	REDUCE(IDX)

// func cdefDirCostsAVX2(s *cdefSums, cost *[8]uint32)
TEXT ·cdefDirCostsAVX2(SB), NOSPLIT, $0-16
	MOVQ s+0(FP), SI
	MOVQ cost+8(FP), DI

	VMOVDQU cdefW105<>(SB), Y15
	VMOVDQU cdefWDiag<>+0(SB), Y13
	VMOVDQU cdefWDiag<>+32(SB), Y14
	VMOVDQU cdefWAlt<>+0(SB), Y11
	VMOVDQU cdefWAlt<>+32(SB), Y12

	COST8(0, Y15, 2)
	COST8(32, Y15, 6)
	COST16(64, Y13, Y14, 0)
	COST16(128, Y13, Y14, 4)
	COST16(192, Y11, Y12, 1)
	COST16(256, Y11, Y12, 3)
	COST16(320, Y11, Y12, 5)
	COST16(384, Y11, Y12, 7)

	VZEROUPPER
	RET
