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

DATA pw2<>+0(SB)/4, $0x00020002
GLOBL pw2<>(SB), RODATA|NOPTR, $4

DATA pd512<>+0(SB)/4, $0x00000200
GLOBL pd512<>(SB), RODATA|NOPTR, $4

// HFILT spreads the row so that every dword holds four consecutive pixels, then
// pairs them with the filter halves. shufB serves twice: taps 4-7 of the low
// four outputs and taps 0-3 of the high four. Coefficients sum to 64 and the
// positive ones to at most 92, so 255*92 keeps every partial sum in a word.
#define HTAPS(V0, V1, V2, V3, RND, SA, SB, SC, F03, F47) \
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
	VPADDW     RND, V0, V0
#define PUT8TAPHV(TAPS) \
	MOVQ dst+0(FP), DI; \
	MOVQ dstStride+8(FP), R8; \
	MOVQ src+16(FP), SI; \
	MOVQ srcStride+24(FP), R9; \
	MOVQ mid+32(FP), BX; \
	MOVQ w+56(FP), DX; \
	MOVQ           fh+40(FP), AX; \
	VBROADCASTI128 subpelShufA<>(SB), Y6; \
	VBROADCASTI128 subpelShufB<>(SB), Y7; \
	VBROADCASTI128 subpelShufC<>(SB), Y8; \
	VPBROADCASTD   (AX), Y9; \
	VPBROADCASTD   4(AX), Y10; \
	VPBROADCASTD   pw2<>(SB), Y5; \
	MOVQ         fv+48(FP), AX; \
	VPBROADCASTD (AX), Y11; \
	VPBROADCASTD 4(AX), Y12; \
	VPBROADCASTD 8(AX), Y13; \
	VPBROADCASTD 12(AX), Y14; \
	VPBROADCASTD pd512<>(SB), Y15; \
	CMPQ DX, $8; \
	JEQ  hvh8; \
hvh16:; \
	MOVQ SI, R10; \
	MOVQ BX, R12; \
	MOVQ h+64(FP), R13; \
	ADDQ $7, R13; \
hvhrow16:; \
	VMOVDQU     (R10), X0; \
	VINSERTI128 $1, 8(R10), Y0, Y0; \
	HTAPS(Y0, Y1, Y2, Y3, Y5, Y6, Y7, Y8, Y9, Y10); \
	VPSRAW      $2, Y0, Y0; \
	VMOVDQU     Y0, (R12); \
	ADDQ        R9, R10; \
	ADDQ        $32, R12; \
	DECQ        R13; \
	JNZ         hvhrow16; \
	MOVQ BX, R12; \
	MOVQ DI, R11; \
	MOVQ h+64(FP), R13; \
hvvrow16:; \
	TAPS(Y0, Y1, Y2, Y3, Y4, Y11, Y12, Y13, Y14, Y15); \
	VPACKUSWB Y2, Y2, Y2; \
	VPERMQ    $0x08, Y2, Y2; \
	VMOVDQU   X2, (R11); \
	ADDQ      R8, R11; \
	ADDQ      $32, R12; \
	DECQ      R13; \
	JNZ       hvvrow16; \
	ADDQ $16, SI; \
	ADDQ $16, DI; \
	SUBQ $16, DX; \
	JG   hvh16; \
	VZEROUPPER; \
	RET; \
hvh8:; \
	MOVQ BX, R12; \
	MOVQ h+64(FP), R13; \
	ADDQ $7, R13; \
hvhrow8:; \
	VMOVDQU (SI), X0; \
	HTAPS(X0, X1, X2, X3, X5, X6, X7, X8, X9, X10); \
	VPSRAW  $2, X0, X0; \
	VMOVDQU X0, (R12); \
	ADDQ    R9, SI; \
	ADDQ    $32, R12; \
	DECQ    R13; \
	JNZ     hvhrow8; \
	MOVQ BX, R12; \
	MOVQ h+64(FP), R13; \
hvvrow8:; \
	TAPS(X0, X1, X2, X3, X4, X11, X12, X13, X14, X15); \
	VPACKUSWB X2, X2, X2; \
	VMOVQ     X2, (DI); \
	ADDQ      R8, DI; \
	ADDQ      $32, R12; \
	DECQ      R13; \
	JNZ       hvvrow8; \
	VZEROUPPER; \
	RET; \
	/* H4TAPS filters four outputs from twelve source bytes, which is all a two or */; \
	/* four wide row reaches. */

#define PREP8TAPHV(TAPS) \
	MOVQ dst+0(FP), DI; \
	MOVQ src+8(FP), SI; \
	MOVQ srcStride+16(FP), R9; \
	MOVQ mid+24(FP), BX; \
	MOVQ w+48(FP), DX; \
	MOVQ DX, R8; \
	MOVQ           fh+32(FP), AX; \
	VBROADCASTI128 subpelShufA<>(SB), Y6; \
	VBROADCASTI128 subpelShufB<>(SB), Y7; \
	VBROADCASTI128 subpelShufC<>(SB), Y8; \
	VPBROADCASTD   (AX), Y9; \
	VPBROADCASTD   4(AX), Y10; \
	VPBROADCASTD   pw2<>(SB), Y5; \
	MOVQ         fv+40(FP), AX; \
	VPBROADCASTD (AX), Y11; \
	VPBROADCASTD 4(AX), Y12; \
	VPBROADCASTD 8(AX), Y13; \
	VPBROADCASTD 12(AX), Y14; \
	VPBROADCASTD pd32<>(SB), Y15; \
	CMPQ DX, $4; \
	JEQ  prphv4; \
	CMPQ DX, $8; \
	JEQ  prphv8; \
prphv16:; \
	MOVQ SI, R10; \
	MOVQ BX, R12; \
	MOVQ h+56(FP), R13; \
	ADDQ $7, R13; \
prphvh16:; \
	VMOVDQU     (R10), X0; \
	VINSERTI128 $1, 8(R10), Y0, Y0; \
	HTAPS(Y0, Y1, Y2, Y3, Y5, Y6, Y7, Y8, Y9, Y10); \
	VPSRAW      $2, Y0, Y0; \
	VMOVDQU     Y0, (R12); \
	ADDQ        R9, R10; \
	ADDQ        $32, R12; \
	DECQ        R13; \
	JNZ         prphvh16; \
	MOVQ BX, R12; \
	MOVQ DI, R11; \
	MOVQ h+56(FP), R13; \
prphvv16:; \
	TAPS(Y0, Y1, Y2, Y3, Y4, Y11, Y12, Y13, Y14, Y15); \
	VMOVDQU Y3, (R11); \
	LEAQ    (R11)(R8*2), R11; \
	ADDQ    $32, R12; \
	DECQ    R13; \
	JNZ     prphvv16; \
	ADDQ $16, SI; \
	ADDQ $32, DI; \
	SUBQ $16, DX; \
	JG   prphv16; \
	VZEROUPPER; \
	RET; \
prphv8:; \
	MOVQ BX, R12; \
	MOVQ h+56(FP), R13; \
	ADDQ $7, R13; \
prphvh8:; \
	VMOVDQU (SI), X0; \
	HTAPS(X0, X1, X2, X3, X5, X6, X7, X8, X9, X10); \
	VPSRAW  $2, X0, X0; \
	VMOVDQU X0, (R12); \
	ADDQ    R9, SI; \
	ADDQ    $32, R12; \
	DECQ    R13; \
	JNZ     prphvh8; \
	MOVQ BX, R12; \
	MOVQ h+56(FP), R13; \
prphvv8:; \
	TAPS(X0, X1, X2, X3, X4, X11, X12, X13, X14, X15); \
	VMOVDQU X3, (DI); \
	ADDQ    $32, R12; \
	LEAQ    (DI)(R8*2), DI; \
	DECQ    R13; \
	JNZ     prphvv8; \
	VZEROUPPER; \
	RET; \
prphv4:; \
	MOVQ BX, R12; \
	MOVQ h+56(FP), R13; \
	ADDQ $7, R13; \
prphvh4:; \
	H4LOAD(SI, X0); \
	H4TAPS(X0, X1, X2, X5, X6, X7, X9, X10); \
	VPSRAW $2, X1, X1; \
	VMOVQ  X1, (R12); \
	ADDQ   R9, SI; \
	ADDQ   $32, R12; \
	DECQ   R13; \
	JNZ    prphvh4; \
	MOVQ BX, R12; \
	MOVQ h+56(FP), R13; \
prphvv4:; \
	TAPS(X0, X1, X2, X3, X4, X11, X12, X13, X14, X15); \
	VMOVQ X3, (DI); \
	ADDQ  $32, R12; \
	LEAQ  (DI)(R8*2), DI; \
	DECQ  R13; \
	JNZ   prphvv4; \
	VZEROUPPER


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
	HTAPS(Y0, Y1, Y2, Y3, Y5, Y6, Y7, Y8, Y9, Y10)
	VPSRAW      $6, Y0, Y0
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
	HTAPS(X0, X1, X2, X3, X5, X6, X7, X8, X9, X10)
	VPSRAW    $6, X0, X0
	VPACKUSWB X0, X0, X0
	VMOVQ     X0, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  hrow8

	VZEROUPPER
	RET

// VSETUP loads the seven rows above the first output, one register per row
// pairing over all sixteen columns. Broadcasting a row to both lanes and
// picking halves with VSHUFPD gets two pairings out of one unpack.
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

// VBODY filters two output rows into Y13, the first in the low lane, and slides
// the window down by two. A rounding multiply-high by 512 is the shift by six.
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

// VTAPS filters eight scratch rows. The scratch is too wide for word products,
// so taps pair through VPMADDWD, which splits the columns over two dword
// accumulators; VPACKSSDW puts them back in order.
#define VTAPS(A, B, ACC0, ACC1, T, F0, F1, F2, F3, RND) \
	VMOVDQU    0(R12), A;     \
	VMOVDQU    32(R12), B;    \
	VPUNPCKLWD B, A, ACC0;    \
	VPUNPCKHWD B, A, ACC1;    \
	VPMADDWD   F0, ACC0, ACC0; \
	VPMADDWD   F0, ACC1, ACC1; \
	VMOVDQU    64(R12), A;    \
	VMOVDQU    96(R12), B;    \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F1, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F1, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VMOVDQU    128(R12), A;   \
	VMOVDQU    160(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F2, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F2, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VMOVDQU    192(R12), A;   \
	VMOVDQU    224(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F3, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F3, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VPADDD     RND, ACC0, ACC0; \
	VPADDD     RND, ACC1, ACC1; \
	VPSRAD     $10, ACC0, ACC0; \
	VPSRAD     $10, ACC1, ACC1; \
	VPACKSSDW  ACC1, ACC0, ACC0

#define VTAPSN(A, B, ACC0, ACC1, T, F0, F1, F2, F3, RND) \
	VMOVDQU    0(R12), A;     \
	VMOVDQU    32(R12), B;    \
	VPUNPCKLWD B, A, ACC0;    \
	VPUNPCKHWD B, A, ACC1;    \
	VPMADDWD   F0, ACC0, ACC0; \
	VPMADDWD   F0, ACC1, ACC1; \
	VMOVDQU    64(R12), A;    \
	VMOVDQU    96(R12), B;    \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F1, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F1, T, ACC1; \
	VMOVDQU    128(R12), A;   \
	VMOVDQU    160(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F2, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F2, T, ACC1; \
	VMOVDQU    192(R12), A;   \
	VMOVDQU    224(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F3, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F3, T, ACC1; \
	VPADDD     RND, ACC0, ACC0; \
	VPADDD     RND, ACC1, ACC1; \
	VPSRAD     $10, ACC0, ACC0; \
	VPSRAD     $10, ACC1, ACC1; \
	VPACKSSDW  ACC1, ACC0, ACC0

// func put8tapHVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)

// func put8tapHVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·put8tapHVAVX2(SB), NOSPLIT, $0-72
	PUT8TAPHV(VTAPS)
	RET

// func put8tapHVAVX512(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·put8tapHVAVX512(SB), NOSPLIT, $0-72
	PUT8TAPHV(VTAPSN)
	RET

#define H4TAPS(V0, V1, V2, RND, SA, SB, F03, F47) \
	VPSHUFB    SB, V0, V2;   \
	VPSHUFB    SA, V0, V1;   \
	VPMADDUBSW F03, V1, V1;  \
	VPMADDUBSW F47, V2, V2;  \
	VPADDW     V2, V1, V1;   \
	VPHADDW    V1, V1, V1;   \
	VPADDW     RND, V1, V1

#define H4LOAD(P, V) \
	VMOVQ   (P), V;  \
	VPINSRD $2, 8(P), V, V

// func put8tapH4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapH4AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ f+32(FP), AX
	MOVQ w+40(FP), DX
	MOVQ h+48(FP), BX

	VMOVDQU      subpelShufA<>(SB), X6
	VMOVDQU      subpelShufB<>(SB), X7
	VPBROADCASTD (AX), X9
	VPBROADCASTD 4(AX), X10
	VPBROADCASTD pw34<>(SB), X5

	CMPQ DX, $2
	JEQ  h4row2

h4row4:
	H4LOAD(SI, X0)
	H4TAPS(X0, X1, X2, X5, X6, X7, X9, X10)
	VPSRAW    $6, X1, X1
	VPACKUSWB X1, X1, X1
	VMOVD     X1, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  h4row4

	VZEROUPPER
	RET

h4row2:
	H4LOAD(SI, X0)
	H4TAPS(X0, X1, X2, X5, X6, X7, X9, X10)
	VPSRAW    $6, X1, X1
	VPACKUSWB X1, X1, X1
	VPEXTRW   $0, X1, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  h4row2

	VZEROUPPER
	RET

// V4TAPS filters one four wide row from the eight rows above R10.
#define V4TAPS(V0, V1, V2, F0, F1, F2, F3) \
	VMOVD      (R10), V0;        \
	VMOVD      (R10)(R9*1), V1;  \
	VPUNPCKLBW V1, V0, V0;       \
	VPMADDUBSW F0, V0, V0;       \
	LEAQ       (R10)(R9*2), R14; \
	VMOVD      (R14), V1;        \
	VMOVD      (R14)(R9*1), V2;  \
	VPUNPCKLBW V2, V1, V1;       \
	VPMADDUBSW F1, V1, V1;       \
	VPADDW     V1, V0, V0;       \
	LEAQ       (R14)(R9*2), R14; \
	VMOVD      (R14), V1;        \
	VMOVD      (R14)(R9*1), V2;  \
	VPUNPCKLBW V2, V1, V1;       \
	VPMADDUBSW F2, V1, V1;       \
	VPADDW     V1, V0, V0;       \
	LEAQ       (R14)(R9*2), R14; \
	VMOVD      (R14), V1;        \
	VMOVD      (R14)(R9*1), V2;  \
	VPUNPCKLBW V2, V1, V1;       \
	VPMADDUBSW F3, V1, V1;       \
	VPADDW     V1, V0, V0

// func put8tapV4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapV4AVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ f+32(FP), AX
	MOVQ w+40(FP), DX
	MOVQ h+48(FP), BX

	VPBROADCASTW (AX), X8
	VPBROADCASTW 2(AX), X9
	VPBROADCASTW 4(AX), X10
	VPBROADCASTW 6(AX), X11
	VPBROADCASTD pw512<>(SB), X7

	CMPQ DX, $2
	JEQ  v4row2

v4row4:
	MOVQ SI, R10
	V4TAPS(X0, X1, X2, X8, X9, X10, X11)
	VPMULHRSW X7, X0, X0
	VPACKUSWB X0, X0, X0
	VMOVD     X0, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  v4row4

	VZEROUPPER
	RET

v4row2:
	MOVQ SI, R10
	V4TAPS(X0, X1, X2, X8, X9, X10, X11)
	VPMULHRSW X7, X0, X0
	VPACKUSWB X0, X0, X0
	VPEXTRW   $0, X0, (DI)

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ BX
	JNZ  v4row2

	VZEROUPPER
	RET

// V4MID filters one four wide row out of the sixteen bit intermediate rows.
#define V4MID(A, B, ACC, T, F0, F1, F2, F3, RND) \
	VMOVQ      (R12), A;      \
	VMOVQ      32(R12), B;    \
	VPUNPCKLWD B, A, ACC;     \
	VPMADDWD   F0, ACC, ACC;  \
	VMOVQ      64(R12), A;    \
	VMOVQ      96(R12), B;    \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F1, T, T;      \
	VPADDD     T, ACC, ACC;   \
	VMOVQ      128(R12), A;   \
	VMOVQ      160(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F2, T, T;      \
	VPADDD     T, ACC, ACC;   \
	VMOVQ      192(R12), A;   \
	VMOVQ      224(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F3, T, T;      \
	VPADDD     T, ACC, ACC;   \
	VPADDD     RND, ACC, ACC; \
	VPSRAD     $10, ACC, ACC; \
	VPACKSSDW  ACC, ACC, ACC

// func put8tapHV4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·put8tapHV4AVX2(SB), NOSPLIT, $0-72
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ mid+32(FP), BX
	MOVQ w+56(FP), DX

	MOVQ         fh+40(FP), AX
	VMOVDQU      subpelShufA<>(SB), X6
	VMOVDQU      subpelShufB<>(SB), X7
	VPBROADCASTD (AX), X9
	VPBROADCASTD 4(AX), X10
	VPBROADCASTD pw2<>(SB), X5

	MOVQ         fv+48(FP), AX
	VPBROADCASTD (AX), X11
	VPBROADCASTD 4(AX), X12
	VPBROADCASTD 8(AX), X13
	VPBROADCASTD 12(AX), X14
	VPBROADCASTD pd512<>(SB), X15

	MOVQ BX, R12
	MOVQ h+64(FP), R13
	ADDQ $7, R13

hv4h:
	H4LOAD(SI, X0)
	H4TAPS(X0, X1, X2, X5, X6, X7, X9, X10)
	VPSRAW $2, X1, X1
	VMOVQ  X1, (R12)

	ADDQ R9, SI
	ADDQ $32, R12
	DECQ R13
	JNZ  hv4h

	MOVQ BX, R12
	MOVQ h+64(FP), R13
	CMPQ DX, $2
	JEQ  hv4v2

hv4v4:
	V4MID(X0, X1, X2, X3, X11, X12, X13, X14, X15)
	VPACKUSWB X2, X2, X2
	VMOVD     X2, (DI)

	ADDQ R8, DI
	ADDQ $32, R12
	DECQ R13
	JNZ  hv4v4

	VZEROUPPER
	RET

hv4v2:
	V4MID(X0, X1, X2, X3, X11, X12, X13, X14, X15)
	VPACKUSWB X2, X2, X2
	VPEXTRW   $0, X2, (DI)

	ADDQ R8, DI
	ADDQ $32, R12
	DECQ R13
	JNZ  hv4v2

	VZEROUPPER
	RET

DATA pw8192<>+0(SB)/4, $0x20002000
GLOBL pw8192<>(SB), RODATA|NOPTR, $4

DATA pd32<>+0(SB)/4, $0x00000020
GLOBL pd32<>(SB), RODATA|NOPTR, $4

// func prep8tapHAVX2(tmp *int16, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·prep8tapHAVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), R9
	MOVQ f+24(FP), AX
	MOVQ w+32(FP), DX
	MOVQ h+40(FP), BX

	VBROADCASTI128 subpelShufA<>(SB), Y6
	VBROADCASTI128 subpelShufB<>(SB), Y7
	VBROADCASTI128 subpelShufC<>(SB), Y8
	VPBROADCASTD   (AX), Y9
	VPBROADCASTD   4(AX), Y10
	VPBROADCASTD   pw2<>(SB), Y5

	CMPQ DX, $4
	JEQ  prph4
	CMPQ DX, $8
	JEQ  prph8

prph16:
	XORQ CX, CX

prpcolh16:
	VMOVDQU     (SI)(CX*1), X0
	VINSERTI128 $1, 8(SI)(CX*1), Y0, Y0
	HTAPS(Y0, Y1, Y2, Y3, Y5, Y6, Y7, Y8, Y9, Y10)
	VPSRAW      $2, Y0, Y0
	VMOVDQU     Y0, (DI)(CX*2)
	ADDQ        $16, CX
	CMPQ        CX, DX
	JLT         prpcolh16

	ADDQ R9, SI
	LEAQ (DI)(DX*2), DI
	DECQ BX
	JNZ  prph16

	VZEROUPPER
	RET

prph8:
	VMOVDQU (SI), X0
	HTAPS(X0, X1, X2, X3, X5, X6, X7, X8, X9, X10)
	VPSRAW  $2, X0, X0
	VMOVDQU X0, (DI)

	ADDQ R9, SI
	ADDQ $16, DI
	DECQ BX
	JNZ  prph8

	VZEROUPPER
	RET

prph4:
	H4LOAD(SI, X0)
	H4TAPS(X0, X1, X2, X5, X6, X7, X9, X10)
	VPSRAW $2, X1, X1
	VMOVQ  X1, (DI)

	ADDQ R9, SI
	ADDQ $8, DI
	DECQ BX
	JNZ  prph4

	VZEROUPPER
	RET

// func prep8tapVAVX2(tmp *int16, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·prep8tapVAVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), R9
	MOVQ f+24(FP), AX
	MOVQ w+32(FP), DX
	MOVQ h+40(FP), BX
	MOVQ DX, R8

	VPBROADCASTW (AX), Y8
	VPBROADCASTW 2(AX), Y9
	VPBROADCASTW 4(AX), Y10
	VPBROADCASTW 6(AX), Y11
	VPBROADCASTD pw8192<>(SB), Y7

	LEAQ (R9)(R9*2), R12

	CMPQ DX, $4
	JEQ  prpv4

prpv8:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R13

prpvrow8:
	MOVQ       R10, R14
	VMOVDQU    (R14), X0
	VMOVDQU    (R14)(R9*1), X1
	VPUNPCKLBW X1, X0, X2
	VPMADDUBSW X8, X2, X2
	LEAQ       (R14)(R9*2), R14
	VMOVDQU    (R14), X0
	VMOVDQU    (R14)(R9*1), X1
	VPUNPCKLBW X1, X0, X3
	VPMADDUBSW X9, X3, X3
	VPADDW     X3, X2, X2
	LEAQ       (R14)(R9*2), R14
	VMOVDQU    (R14), X0
	VMOVDQU    (R14)(R9*1), X1
	VPUNPCKLBW X1, X0, X3
	VPMADDUBSW X10, X3, X3
	VPADDW     X3, X2, X2
	LEAQ       (R14)(R9*2), R14
	VMOVDQU    (R14), X0
	VMOVDQU    (R14)(R9*1), X1
	VPUNPCKLBW X1, X0, X3
	VPMADDUBSW X11, X3, X3
	VPADDW     X3, X2, X2
	VPMULHRSW  X7, X2, X2
	VMOVDQU    X2, (R11)

	ADDQ R9, R10
	LEAQ (R11)(R8*2), R11
	DECQ R13
	JNZ  prpvrow8

	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, DX
	JG   prpv8

	VZEROUPPER
	RET

prpv4:
	MOVQ SI, R10
	V4TAPS(X0, X1, X2, X8, X9, X10, X11)
	VPMULHRSW X7, X0, X0
	VMOVQ     X0, (DI)

	ADDQ R9, SI
	ADDQ $8, DI
	DECQ BX
	JNZ  prpv4

	VZEROUPPER
	RET

#define PVTAPS(A, B, T, ACC0, ACC1, F0, F1, F2, F3, RND) \
	VMOVDQU    (R12), A;      \
	VMOVDQU    32(R12), B;    \
	VPUNPCKLWD B, A, ACC0;    \
	VPUNPCKHWD B, A, ACC1;    \
	VPMADDWD   F0, ACC0, ACC0; \
	VPMADDWD   F0, ACC1, ACC1; \
	VMOVDQU    64(R12), A;    \
	VMOVDQU    96(R12), B;    \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F1, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F1, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VMOVDQU    128(R12), A;   \
	VMOVDQU    160(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F2, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F2, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VMOVDQU    192(R12), A;   \
	VMOVDQU    224(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPMADDWD   F3, T, T;      \
	VPADDD     T, ACC0, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPMADDWD   F3, T, T;      \
	VPADDD     T, ACC1, ACC1; \
	VPADDD     RND, ACC0, ACC0; \
	VPADDD     RND, ACC1, ACC1; \
	VPSRAD     $6, ACC0, ACC0; \
	VPSRAD     $6, ACC1, ACC1; \
	VPACKSSDW  ACC1, ACC0, ACC0

#define PVTAPSN(A, B, T, ACC0, ACC1, F0, F1, F2, F3, RND) \
	VMOVDQU    (R12), A;      \
	VMOVDQU    32(R12), B;    \
	VPUNPCKLWD B, A, ACC0;    \
	VPUNPCKHWD B, A, ACC1;    \
	VPMADDWD   F0, ACC0, ACC0; \
	VPMADDWD   F0, ACC1, ACC1; \
	VMOVDQU    64(R12), A;    \
	VMOVDQU    96(R12), B;    \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F1, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F1, T, ACC1; \
	VMOVDQU    128(R12), A;   \
	VMOVDQU    160(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F2, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F2, T, ACC1; \
	VMOVDQU    192(R12), A;   \
	VMOVDQU    224(R12), B;   \
	VPUNPCKLWD B, A, T;       \
	VPDPWSSD   F3, T, ACC0; \
	VPUNPCKHWD B, A, T;       \
	VPDPWSSD   F3, T, ACC1; \
	VPADDD     RND, ACC0, ACC0; \
	VPADDD     RND, ACC1, ACC1; \
	VPSRAD     $6, ACC0, ACC0; \
	VPSRAD     $6, ACC1, ACC1; \
	VPACKSSDW  ACC1, ACC0, ACC0

// func prep8tapHVAVX2(tmp *int16, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)

// func prep8tapHVAVX2(tmp *int16, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·prep8tapHVAVX2(SB), NOSPLIT, $0-64
	PREP8TAPHV(PVTAPS)
	RET

// func prep8tapHVAVX512(tmp *int16, src *uint8, srcStride int, mid *int16, fh *[8]int8, fv *[8]int16, w, h int)
TEXT ·prep8tapHVAVX512(SB), NOSPLIT, $0-64
	PREP8TAPHV(PVTAPSN)
	RET

DATA pw2048<>+0(SB)/4, $0x08000800
GLOBL pw2048<>(SB), RODATA|NOPTR, $4

DATA pd128<>+0(SB)/4, $0x00000080
GLOBL pd128<>(SB), RODATA|NOPTR, $4

DATA pd8<>+0(SB)/4, $0x00000008
GLOBL pd8<>(SB), RODATA|NOPTR, $4

#define BILH                    \
	VMOVQ      (SI), X0;    \
	VMOVQ      1(SI), X1;   \
	VPUNPCKLBW X1, X0, X0;  \
	VPMADDUBSW X14, X0, X0

#define BILV                       \
	VMOVQ      (SI), X0;       \
	VMOVQ      (SI)(R9*1), X1; \
	VPUNPCKLBW X1, X0, X0;     \
	VPMADDUBSW X14, X0, X0

#define BILCOEF(OFF, REG)    \
	MOVL  OFF, AX;       \
	MOVL  $16, CX;       \
	SUBL  AX, CX;        \
	SHLL  $8, AX;        \
	ORL   CX, AX;        \
	VMOVD AX, X13;       \
	VPBROADCASTW X13, REG


#define STPX(LBL)             \
	VPACKUSWB X0, X0, X0; \
	CMPQ      R12, $4;    \
	JLE       LBL;        \
	VMOVQ     X0, (DI)

#define STTMP(LBL)         \
	CMPQ  R12, $4;     \
	JLE   LBL;         \
	VMOVDQU X0, (DI)

// func putBilinHAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mx int32, w, h int)
TEXT ·putBilinHAVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ w+40(FP), BX
	MOVQ h+48(FP), DX

	BILCOEF(mx+32(FP), X14)
	VPBROADCASTD pw2048<>(SB), X15

blhrow:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

blhcol:
	BILH
	VPMULHRSW X15, X0, X0
	STPX(blh4)
	JMP       blhnext

blh4:
	VMOVD X0, (DI)

blhnext:
	ADDQ $8, SI
	ADDQ $8, DI
	SUBQ $8, R12
	JG   blhcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(R8*1), DI
	DECQ DX
	JNZ  blhrow

	VZEROUPPER
	RET

// func putBilinVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, my int32, w, h int)
TEXT ·putBilinVAVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ w+40(FP), BX
	MOVQ h+48(FP), DX

	BILCOEF(my+32(FP), X14)
	VPBROADCASTD pw2048<>(SB), X15

blvrow:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

blvcol:
	BILV
	VPMULHRSW X15, X0, X0
	STPX(blv4)
	JMP       blvnext

blv4:
	VMOVD X0, (DI)

blvnext:
	ADDQ $8, SI
	ADDQ $8, DI
	SUBQ $8, R12
	JG   blvcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(R8*1), DI
	DECQ DX
	JNZ  blvrow

	VZEROUPPER
	RET

// func prepBilinHAVX2(tmp *int16, src *uint8, srcStride int, mx int32, w, h int)
TEXT ·prepBilinHAVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), R9
	MOVQ w+32(FP), BX
	MOVQ h+40(FP), DX

	BILCOEF(mx+24(FP), X14)

prblhrow:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

prblhcol:
	BILH
	STTMP(prblh4)
	JMP     prblhnext

prblh4:
	VMOVQ X0, (DI)

prblhnext:
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, R12
	JG   prblhcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(BX*2), DI
	DECQ DX
	JNZ  prblhrow

	VZEROUPPER
	RET

// func prepBilinVAVX2(tmp *int16, src *uint8, srcStride int, my int32, w, h int)
TEXT ·prepBilinVAVX2(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), R9
	MOVQ w+32(FP), BX
	MOVQ h+40(FP), DX

	BILCOEF(my+24(FP), X14)

prblvrow:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

prblvcol:
	BILV
	STTMP(prblv4)
	JMP     prblvnext

prblv4:
	VMOVQ X0, (DI)

prblvnext:
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, R12
	JG   prblvcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(BX*2), DI
	DECQ DX
	JNZ  prblvrow

	VZEROUPPER
	RET

#define BILVMID(RND, SHIFT)          \
	VMOVDQU    (R13), X0;        \
	VMOVDQU    (R13)(R15*1), X1; \
	VPUNPCKLWD X1, X0, X2;      \
	VPUNPCKHWD X1, X0, X3;      \
	VPMADDWD   X14, X2, X2;     \
	VPMADDWD   X14, X3, X3;     \
	VPADDD     RND, X2, X2;     \
	VPADDD     RND, X3, X3;     \
	VPSRAD     SHIFT, X2, X2;   \
	VPSRAD     SHIFT, X3, X3;   \
	VPACKSSDW  X3, X2, X0

// func putBilinHVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)
TEXT ·putBilinHVAVX2(SB), NOSPLIT, $0-64
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ mid+32(FP), R14
	MOVQ w+48(FP), BX
	MOVQ h+56(FP), DX

	BILCOEF(mx+40(FP), X14)

	MOVQ SI, R10
	MOVQ R14, R11
	MOVQ DX, R15
	INCQ R15

blhvh:
	MOVQ SI, R10
	MOVQ R11, R13
	MOVQ BX, R12

blhvhcol:
	BILH
	VMOVDQU X0, (R13)
	ADDQ    $8, SI
	ADDQ    $16, R13
	SUBQ    $8, R12
	JG      blhvhcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(BX*2), R11
	DECQ R15
	JNZ  blhvh

	LEAQ         (BX)(BX*1), R15
	MOVL         my+44(FP), AX
	MOVL         $16, CX
	SUBL         AX, CX
	SHLL         $16, AX
	ORL          CX, AX
	VMOVD        AX, X13
	VPBROADCASTD X13, X14
	VPBROADCASTD pd128<>(SB), X15
	MOVQ         R14, R11

blhvv:
	MOVQ R11, R13
	MOVQ DI, R10
	MOVQ BX, R12

blhvvcol:
	BILVMID(X15, $8)
	STPX(blhv4)
	JMP       blhvnext

blhv4:
	VMOVD X0, (DI)

blhvnext:
	ADDQ $16, R13
	ADDQ $8, DI
	SUBQ $8, R12
	JG   blhvvcol

	LEAQ (R11)(BX*2), R11
	LEAQ (R10)(R8*1), DI
	DECQ DX
	JNZ  blhvv

	VZEROUPPER
	RET

// func prepBilinHVAVX2(tmp *int16, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)
TEXT ·prepBilinHVAVX2(SB), NOSPLIT, $0-56
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ srcStride+16(FP), R9
	MOVQ mid+24(FP), R14
	MOVQ w+40(FP), BX
	MOVQ h+48(FP), DX

	BILCOEF(mx+32(FP), X14)

	MOVQ R14, R11
	MOVQ DX, R15
	INCQ R15

prblhvh:
	MOVQ SI, R10
	MOVQ R11, R13
	MOVQ BX, R12

prblhvhcol:
	BILH
	VMOVDQU X0, (R13)
	ADDQ    $8, SI
	ADDQ    $16, R13
	SUBQ    $8, R12
	JG      prblhvhcol

	LEAQ (R10)(R9*1), SI
	LEAQ (R11)(BX*2), R11
	DECQ R15
	JNZ  prblhvh

	LEAQ         (BX)(BX*1), R15
	MOVL         my+36(FP), AX
	MOVL         $16, CX
	SUBL         AX, CX
	SHLL         $16, AX
	ORL          CX, AX
	VMOVD        AX, X13
	VPBROADCASTD X13, X14
	VPBROADCASTD pd8<>(SB), X15
	MOVQ         R14, R11

prblhvv:
	MOVQ R11, R13
	MOVQ DI, R10
	MOVQ BX, R12

prblhvvcol:
	BILVMID(X15, $4)
	STTMP(prblhv4)
	JMP     prblhvnext

prblhv4:
	VMOVQ X0, (DI)

prblhvnext:
	ADDQ $16, R13
	ADDQ $16, DI
	SUBQ $8, R12
	JG   prblhvvcol

	LEAQ (R11)(BX*2), R11
	LEAQ (R10)(BX*2), DI
	DECQ DX
	JNZ  prblhvv

	VZEROUPPER
	RET
