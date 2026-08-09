//go:build amd64 && !noasm

#include "textflag.h"

DATA cdefRev<>+0(SB)/4, $7
DATA cdefRev<>+4(SB)/4, $6
DATA cdefRev<>+8(SB)/4, $5
DATA cdefRev<>+12(SB)/4, $4
DATA cdefRev<>+16(SB)/4, $3
DATA cdefRev<>+20(SB)/4, $2
DATA cdefRev<>+24(SB)/4, $1
DATA cdefRev<>+28(SB)/4, $0
GLOBL cdefRev<>(SB), RODATA|NOPTR, $32

DATA cdefPair<>+0(SB)/4, $0
DATA cdefPair<>+4(SB)/4, $2
DATA cdefPair<>+8(SB)/4, $4
DATA cdefPair<>+12(SB)/4, $6
DATA cdefPair<>+16(SB)/4, $0
DATA cdefPair<>+20(SB)/4, $2
DATA cdefPair<>+24(SB)/4, $4
DATA cdefPair<>+28(SB)/4, $6
GLOBL cdefPair<>(SB), RODATA|NOPTR, $32

#define HV0  0
#define HV1  32
#define DIA0 64
#define DIA1 128
#define ALT0 192
#define ALT1 256
#define ALT2 320
#define ALT3 384

// ADDAT accumulates a whole row into one of the padded arrays at an offset
// that only the row index decides.
#define ADDAT(OFF, SRC)         \
	VMOVDQU OFF(DI), Y4;    \
	VPADDD  SRC, Y4, Y4;    \
	VMOVDQU Y4, OFF(DI)

#define ADDAT4(OFF, SRC)        \
	VMOVDQU OFF(DI), X4;    \
	VPADDD  SRC, X4, X4;    \
	VMOVDQU X4, OFF(DI)

// ROW folds one row of eight samples into all eight partial sums. Y is the row,
// HY is Y>>1, which is where the alternating sums step half as fast.
#define ROW(Y, HY)                                \
	VPADDD       Y3, Y0, Y0;                  \
	VPHADDD      Y3, Y3, Y4;                  \
	VPHADDD      Y4, Y4, Y4;                  \
	VEXTRACTI128 $1, Y4, X5;                  \
	VPADDD       X5, X4, X4;                  \
	VMOVD        X4, AX;                      \
	MOVL         AX, (HV0+4*(Y))(DI);         \
	ADDAT(DIA0+4*(Y), Y3);                    \
	VPERMD       Y3, Y2, Y6;                  \
	ADDAT(DIA1+4*(Y), Y6);                    \
	ADDAT(ALT2+4*(3-(HY)), Y3);               \
	ADDAT(ALT3+4*(HY), Y3);                   \
	VPSRLQ       $32, Y3, Y5;                 \
	VPADDD       Y5, Y3, Y5;                  \
	VPERMD       Y5, Y7, Y5;                  \
	ADDAT4(ALT0+4*(Y), X5);                   \
	VPSHUFD      $0x1b, X5, X6;               \
	ADDAT4(ALT1+4*(Y), X6)

#define SETUP                              \
	MOVQ           s+16(FP), DI;       \
	VPXOR          Y0, Y0, Y0;         \
	MOVL           $128, AX;           \
	VMOVD          AX, X1;             \
	VPBROADCASTD   X1, Y1;             \
	VMOVDQU        cdefRev<>(SB), Y2;  \
	VMOVDQU        cdefPair<>(SB), Y7

// func cdefSums8AVX2(img *uint8, stride int, s *cdefSums)
TEXT ·cdefSums8AVX2(SB), NOSPLIT, $0-24
	MOVQ img+0(FP), SI
	MOVQ stride+8(FP), DX
	SETUP

	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(0, 0)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(1, 0)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(2, 1)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(3, 1)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(4, 2)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(5, 2)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(6, 3)
	ADDQ      DX, SI
	VPMOVZXBD (SI), Y3
	VPSUBD    Y1, Y3, Y3
	ROW(7, 3)

	VMOVDQU Y0, HV1(DI)
	VZEROUPPER
	RET

// func cdefSums16AVX2(img *uint16, stride int, s *cdefSums, shift int)
TEXT ·cdefSums16AVX2(SB), NOSPLIT, $0-32
	MOVQ img+0(FP), SI
	MOVQ stride+8(FP), DX
	SHLQ $1, DX
	SETUP
	MOVQ  shift+24(FP), AX
	VMOVD AX, X8

	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(0, 0)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(1, 0)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(2, 1)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(3, 1)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(4, 2)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(5, 2)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(6, 3)
	ADDQ      DX, SI
	VPMOVZXWD (SI), Y3
	VPSRLD    X8, Y3, Y3
	VPSUBD    Y1, Y3, Y3
	ROW(7, 3)

	VMOVDQU Y0, HV1(DI)
	VZEROUPPER
	RET
