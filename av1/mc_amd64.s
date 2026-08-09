//go:build amd64 && !noasm

#include "textflag.h"

DATA subpelShufA<>+0(SB)/8, $0x0403020103020100
DATA subpelShufA<>+8(SB)/8, $0x0605040305040302
GLOBL subpelShufA<>(SB), RODATA|NOPTR, $16

DATA subpelShufB<>+0(SB)/8, $0x0807060507060504
DATA subpelShufB<>+8(SB)/8, $0x0a09080709080706
GLOBL subpelShufB<>(SB), RODATA|NOPTR, $16

DATA subpelShufC<>+0(SB)/8, $0x0c0b0a090b0a0908
DATA subpelShufC<>+8(SB)/8, $0x0e0d0c0b0d0c0b0a
GLOBL subpelShufC<>(SB), RODATA|NOPTR, $16

DATA pw34<>+0(SB)/4, $0x00220022
GLOBL pw34<>(SB), RODATA|NOPTR, $4

DATA pw512<>+0(SB)/4, $0x02000200
GLOBL pw512<>(SB), RODATA|NOPTR, $4

// HFILT spreads the row so that every dword holds four consecutive pixels, then
// pairs them with the filter halves. shufB serves twice: taps 4-7 of the low
// four outputs and taps 0-3 of the high four. Coefficients sum to 64 and the
// positive ones to at most 92, so 255*92 keeps every partial sum in a word.
#define HFILT(V0, V1, V2, V3, RND, SA, SB, SC, F03, F47) \
	VPSHUFB    SB, V0, V1;   \
	VPSHUFB    SC, V0, V2;   \
	VPSHUFB    SA, V0, V0;   \
	VPMADDUBSW F03, V1, V3;  \
	VPMADDUBSW F47, V1, V1;  \
	VPMADDUBSW F47, V2, V2;  \
	VPMADDUBSW F03, V0, V0;  \
	VPADDW     V3, V2, V2;   \
	VPADDW     V1, V0, V0;   \
	VPHADDW    V2, V0, V0;   \
	VPADDW     RND, V0, V0;  \
	VPSRAW     $6, V0, V0

// func put8tapHAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapHAVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ f+32(FP), AX
	MOVQ w+40(FP), DX
	MOVQ h+48(FP), BX

	VBROADCASTI128 subpelShufA<>(SB), Y6
	VBROADCASTI128 subpelShufB<>(SB), Y7
	VBROADCASTI128 subpelShufC<>(SB), Y8
	VPBROADCASTD   (AX), Y9
	VPBROADCASTD   4(AX), Y10
	VPBROADCASTD   pw34<>(SB), Y5

	CMPQ DX, $8
	JEQ  hrow8

hrow16:
	XORQ CX, CX

hcol16:
	VMOVDQU     (SI)(CX*1), X0
	VINSERTI128 $1, 8(SI)(CX*1), Y0, Y0
	HFILT(Y0, Y1, Y2, Y3, Y5, Y6, Y7, Y8, Y9, Y10)
	VPACKUSWB   Y0, Y0, Y0
	VPERMQ      $0x08, Y0, Y0
	VMOVDQU     X0, (DI)(CX*1)
	ADDQ        $16, CX
	CMPQ        CX, DX
	JLT         hcol16

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  hrow16

	VZEROUPPER
	RET

hrow8:
	VMOVDQU   (SI), X0
	HFILT(X0, X1, X2, X3, X5, X6, X7, X8, X9, X10)
	VPACKUSWB X0, X0, X0
	VMOVQ     X0, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  hrow8

	VZEROUPPER
	RET

// VSETUP loads the seven rows above the first output and pairs them up. Each
// register holds two rows interleaved for all sixteen columns: a broadcast puts
// a row in both lanes, VSHUFPD picks one half of each of two rows, and then the
// low and high unpack yield two pairings from one pass.
#define VSETUP                       \
	VBROADCASTI128 (SI), Y4;     \
	VBROADCASTI128 (SI)(R9*1), Y5; \
	LEAQ           (SI)(R12*1), R10; \
	VBROADCASTI128 (SI)(R9*2), Y6; \
	VBROADCASTI128 (R10), Y0;    \
	VBROADCASTI128 (R10)(R9*1), Y1; \
	VBROADCASTI128 (R10)(R9*2), Y2; \
	ADDQ           R12, R10;     \
	VBROADCASTI128 (R10), Y3;    \
	VSHUFPD        $0x0c, Y0, Y4, Y4; \
	VSHUFPD        $0x0c, Y1, Y5, Y5; \
	VPUNPCKLBW     Y5, Y4, Y1;   \
	VPUNPCKHBW     Y5, Y4, Y4;   \
	VSHUFPD        $0x0c, Y2, Y6, Y6; \
	VPUNPCKLBW     Y6, Y5, Y2;   \
	VPUNPCKHBW     Y6, Y5, Y5;   \
	VSHUFPD        $0x0c, Y3, Y0, Y0; \
	VPUNPCKLBW     Y0, Y6, Y3;   \
	VPUNPCKHBW     Y0, Y6, Y6

// VBODY filters two output rows and slides the window down by two. Y13 ends
// holding both rows packed, the first in the low lane. Multiplying by 512 and
// taking the rounded high half is the rounding shift by six.
#define VBODY                        \
	VBROADCASTI128 (R10)(R9*1), Y12; \
	LEAQ           (R10)(R9*2), R10; \
	VPMADDUBSW     Y8, Y1, Y13;  \
	VPMADDUBSW     Y8, Y2, Y14;  \
	VMOVDQU        Y3, Y1;       \
	VPMADDUBSW     Y9, Y3, Y3;   \
	VMOVDQU        Y4, Y2;       \
	VPMADDUBSW     Y9, Y4, Y4;   \
	VPADDW         Y3, Y13, Y13; \
	VPADDW         Y4, Y14, Y14; \
	VMOVDQU        Y5, Y3;       \
	VMOVDQU        Y6, Y4;       \
	VPMADDUBSW     Y10, Y5, Y5;  \
	VPMADDUBSW     Y10, Y6, Y6;  \
	VPADDW         Y5, Y13, Y13; \
	VBROADCASTI128 (R10), Y5;    \
	VPADDW         Y6, Y14, Y14; \
	VSHUFPD        $0x0d, Y12, Y0, Y6; \
	VSHUFPD        $0x0c, Y5, Y12, Y0; \
	VPUNPCKLBW     Y0, Y6, Y5;   \
	VPUNPCKHBW     Y0, Y6, Y6;   \
	VPMADDUBSW     Y11, Y5, Y12; \
	VPADDW         Y12, Y13, Y13; \
	VPMADDUBSW     Y11, Y6, Y12; \
	VPADDW         Y12, Y14, Y14; \
	VPMULHRSW      Y7, Y13, Y13; \
	VPMULHRSW      Y7, Y14, Y14; \
	VPACKUSWB      Y14, Y13, Y13; \
	VPERMQ         $0xd8, Y13, Y13

// func put8tapVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapVAVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ f+32(FP), AX
	MOVQ w+40(FP), DX
	MOVQ h+48(FP), BX

	VPBROADCASTW (AX), Y8
	VPBROADCASTW 2(AX), Y9
	VPBROADCASTW 4(AX), Y10
	VPBROADCASTW 6(AX), Y11
	VPBROADCASTD pw512<>(SB), Y7

	LEAQ (R9)(R9*2), R12

	CMPQ DX, $8
	JEQ  vcol8

vcol16:
	MOVQ BX, R13
	MOVQ DI, R11
	VSETUP

vrow16:
	VBODY
	VMOVDQU      X13, (R11)
	VEXTRACTI128 $1, Y13, (R11)(R8*1)
	LEAQ         (R11)(R8*2), R11
	SUBQ         $2, R13
	JG           vrow16

	ADDQ $16, SI
	ADDQ $16, DI
	SUBQ $16, DX
	JG   vcol16

	VZEROUPPER
	RET

vcol8:
	MOVQ BX, R13
	MOVQ DI, R11
	VSETUP

vrow8:
	VBODY
	VMOVQ        X13, (R11)
	VEXTRACTI128 $1, Y13, X12
	VMOVQ        X12, (R11)(R8*1)
	LEAQ         (R11)(R8*2), R11
	SUBQ         $2, R13
	JG           vrow8

	VZEROUPPER
	RET
