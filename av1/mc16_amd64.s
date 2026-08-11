//go:build amd64 && !noasm

#include "textflag.h"

// At high bit depth a sample and a tap both fit a word, so two taps go through
// one VPMADDWD. The pair is one tap stride apart, which is what still lets a
// single kernel serve both directions.

// FSETUP broadcasts the four tap pairs the caller has packed.
#define FSETUP                    \
	MOVQ         f+40(FP), AX; \
	VPBROADCASTD (AX), Y8;    \
	VPBROADCASTD 4(AX), Y9;   \
	VPBROADCASTD 8(AX), Y10;  \
	VPBROADCASTD 12(AX), Y11; \
	MOVQ         dst+0(FP), DI; \
	MOVQ         src+16(FP), SI; \
	MOVQ         tapStride+32(FP), R10; \
	SHLQ         $1, R10;     \
	MOVQ         rnd+64(FP), AX; \
	VMOVD        AX, X7;      \
	VPBROADCASTD X7, Y7;      \
	MOVQ         shift+72(FP), AX; \
	VMOVD        AX, X6;      \
	MOVQ         maxv+80(FP), AX; \
	VMOVD        AX, X5;      \
	VPBROADCASTD X5, Y5;      \
	VPXOR        Y4, Y4, Y4;  \
	MOVQ         h+56(FP), R12

// TAPS8 leaves the low half of every lane in A0 and the high half in A1, which
// is the order VPACKUSDW puts back together.
#define TAPS8(A0, A1, L0, L1, T0, T1, F0, F1, F2, F3, RND) \
	VMOVDQU    (CX), L0;         \
	VMOVDQU    (CX)(R10*1), L1;  \
	VPUNPCKLWD L1, L0, A0;       \
	VPUNPCKHWD L1, L0, A1;       \
	VPMADDWD   F0, A0, A0;       \
	VPMADDWD   F0, A1, A1;       \
	LEAQ       (CX)(R10*2), CX;  \
	VMOVDQU    (CX), L0;         \
	VMOVDQU    (CX)(R10*1), L1;  \
	VPUNPCKLWD L1, L0, T0;       \
	VPUNPCKHWD L1, L0, T1;       \
	VPMADDWD   F1, T0, T0;       \
	VPMADDWD   F1, T1, T1;       \
	VPADDD     T0, A0, A0;       \
	VPADDD     T1, A1, A1;       \
	LEAQ       (CX)(R10*2), CX;  \
	VMOVDQU    (CX), L0;         \
	VMOVDQU    (CX)(R10*1), L1;  \
	VPUNPCKLWD L1, L0, T0;       \
	VPUNPCKHWD L1, L0, T1;       \
	VPMADDWD   F2, T0, T0;       \
	VPMADDWD   F2, T1, T1;       \
	VPADDD     T0, A0, A0;       \
	VPADDD     T1, A1, A1;       \
	LEAQ       (CX)(R10*2), CX;  \
	VMOVDQU    (CX), L0;         \
	VMOVDQU    (CX)(R10*1), L1;  \
	VPUNPCKLWD L1, L0, T0;       \
	VPUNPCKHWD L1, L0, T1;       \
	VPMADDWD   F3, T0, T0;       \
	VPMADDWD   F3, T1, T1;       \
	VPADDD     T0, A0, A0;       \
	VPADDD     T1, A1, A1;       \
	VPADDD     RND, A0, A0;      \
	VPADDD     RND, A1, A1;      \
	VPSRAD     X6, A0, A0;       \
	VPSRAD     X6, A1, A1

#define TAPSY TAPS8(Y0, Y3, Y1, Y2, Y13, Y14, Y8, Y9, Y10, Y11, Y7)
#define TAPSX TAPS8(X0, X3, X1, X2, X13, X14, X8, X9, X10, X11, X7)

#define CLIPY               \
	VPMAXSD Y4, Y0, Y0; \
	VPMAXSD Y4, Y3, Y3; \
	VPMINSD Y5, Y0, Y0; \
	VPMINSD Y5, Y3, Y3

#define CLIPX               \
	VPMAXSD X4, X0, X0; \
	VPMAXSD X4, X3, X3; \
	VPMINSD X5, X0, X0; \
	VPMINSD X5, X3, X3

#define ROWSTEP(NAME)              \
	MOVQ srcStride+24(FP), AX; \
	SHLQ $1, AX;               \
	ADDQ AX, SI;               \
	MOVQ dstStride+8(FP), AX;  \
	SHLQ $1, AX;               \
	ADDQ AX, DI;               \
	DECQ R12;                  \
	JNZ  NAME

// The bilinear filter is two taps, so one VPMADDWD takes the whole filter. Its
// horizontal put rounds twice at twelve bits, which is the second shift the
// eight tap kernels do not carry.
#define BSETUP                              \
	MOVL         mxy+40(FP), AX;        \
	MOVL         $16, BX;               \
	SUBL         AX, BX;                \
	SHLL         $16, AX;               \
	ORL          AX, BX;                \
	VMOVD        BX, X8;                \
	VPBROADCASTD X8, Y8;                \
	MOVQ         dst+0(FP), DI;         \
	MOVQ         src+16(FP), SI;        \
	MOVQ         tapStride+32(FP), R10; \
	SHLQ         $1, R10;               \
	MOVQ         rnd+64(FP), AX;        \
	VMOVD        AX, X7;                \
	VPBROADCASTD X7, Y7;                \
	MOVQ         shift+72(FP), AX;      \
	VMOVD        AX, X6;                \
	MOVQ         rnd2+80(FP), AX;       \
	VMOVD        AX, X3;                \
	VPBROADCASTD X3, Y3;                \
	MOVQ         shift2+88(FP), AX;     \
	VMOVD        AX, X2;                \
	MOVQ         maxv+96(FP), AX;       \
	VMOVD        AX, X5;                \
	VPBROADCASTD X5, Y5;                \
	VPXOR        Y4, Y4, Y4;            \
	MOVQ         h+56(FP), R12

#define TAPS2(A0, A1, L0, L1, F, RND, RND2) \
	VMOVDQU    (CX), L0;         \
	VMOVDQU    (CX)(R10*1), L1;  \
	VPUNPCKLWD L1, L0, A0;       \
	VPUNPCKHWD L1, L0, A1;       \
	VPMADDWD   F, A0, A0;        \
	VPMADDWD   F, A1, A1;        \
	VPADDD     RND, A0, A0;      \
	VPADDD     RND, A1, A1;      \
	VPSRAD     X6, A0, A0;       \
	VPSRAD     X6, A1, A1;       \
	VPADDD     RND2, A0, A0;     \
	VPADDD     RND2, A1, A1;     \
	VPSRAD     X2, A0, A0;       \
	VPSRAD     X2, A1, A1

#define TAPS2Y TAPS2(Y0, Y13, Y1, Y14, Y8, Y7, Y3)
#define TAPS2X TAPS2(X0, X13, X1, X14, X8, X7, X3)

#define BCLIPY                 \
	VPMAXSD Y4, Y0, Y0;    \
	VPMAXSD Y4, Y13, Y13;  \
	VPMINSD Y5, Y0, Y0;    \
	VPMINSD Y5, Y13, Y13

#define BCLIPX                 \
	VPMAXSD X4, X0, X0;    \
	VPMAXSD X4, X13, X13;  \
	VPMINSD X5, X0, X0;    \
	VPMINSD X5, X13, X13

#define BROWSTEP(NAME)             \
	MOVQ srcStride+24(FP), AX; \
	SHLQ $1, AX;               \
	ADDQ AX, SI;               \
	MOVQ dstStride+8(FP), AX;  \
	SHLQ $1, AX;               \
	ADDQ AX, DI;               \
	DECQ R12;                  \
	JNZ  NAME

// func mcTap16AVX2(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int, f *[4]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTap16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows1:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  narrow1

cols1:
	MOVQ R11, CX
	TAPSY
	CLIPY
	VPACKUSDW Y3, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       cols1
	TESTQ     DX, DX
	JZ        next1

narrow1:
	MOVQ R11, CX
	TAPSX
	CLIPX
	VPACKUSDW X3, X0, X0
	VMOVDQU   X0, (R13)

next1:
	ROWSTEP(rows1)
	VZEROUPPER
	RET

// func mcTapMid16AVX2(dst *int16, dstStride int, src *uint16, srcStride, tapStride int, f *[4]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows2:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  narrow2

cols2:
	MOVQ R11, CX
	TAPSY
	VPACKSSDW Y3, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       cols2
	TESTQ     DX, DX
	JZ        next2

narrow2:
	MOVQ R11, CX
	TAPSX
	VPACKSSDW X3, X0, X0
	VMOVDQU   X0, (R13)

next2:
	ROWSTEP(rows2)
	VZEROUPPER
	RET

// func mcTapFromMid16AVX2(dst *uint16, dstStride int, src *int16, srcStride, tapStride int, f *[4]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapFromMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows3:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  narrow3

cols3:
	MOVQ R11, CX
	TAPSY
	CLIPY
	VPACKUSDW Y3, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       cols3
	TESTQ     DX, DX
	JZ        next3

narrow3:
	MOVQ R11, CX
	TAPSX
	CLIPX
	VPACKUSDW X3, X0, X0
	VMOVDQU   X0, (R13)

next3:
	ROWSTEP(rows3)
	VZEROUPPER
	RET

// func mcTapPrepMid16AVX2(dst *int16, dstStride int, src *int16, srcStride, tapStride int, f *[4]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapPrepMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows4:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  narrow4

cols4:
	MOVQ R11, CX
	TAPSY
	VPACKSSDW Y3, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       cols4
	TESTQ     DX, DX
	JZ        next4

narrow4:
	MOVQ R11, CX
	TAPSX
	VPACKSSDW X3, X0, X0
	VMOVDQU   X0, (R13)

next4:
	ROWSTEP(rows4)
	VZEROUPPER
	RET

// func mcBilin16AVX2(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int, mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)
TEXT ·mcBilin16AVX2(SB), NOSPLIT, $0-100
	BSETUP

brows1:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  bnarrow1

bcols1:
	MOVQ R11, CX
	TAPS2Y
	BCLIPY
	VPACKUSDW Y13, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       bcols1
	TESTQ     DX, DX
	JZ        bnext1

bnarrow1:
	MOVQ R11, CX
	TAPS2X
	BCLIPX
	VPACKUSDW X13, X0, X0
	VMOVDQU   X0, (R13)

bnext1:
	BROWSTEP(brows1)
	VZEROUPPER
	RET

// func mcBilinMid16AVX2(dst *int16, dstStride int, src *uint16, srcStride, tapStride int, mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)
TEXT ·mcBilinMid16AVX2(SB), NOSPLIT, $0-100
	BSETUP

brows2:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  bnarrow2

bcols2:
	MOVQ R11, CX
	TAPS2Y
	VPACKSSDW Y13, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       bcols2
	TESTQ     DX, DX
	JZ        bnext2

bnarrow2:
	MOVQ R11, CX
	TAPS2X
	VPACKSSDW X13, X0, X0
	VMOVDQU   X0, (R13)

bnext2:
	BROWSTEP(brows2)
	VZEROUPPER
	RET

// func mcBilinFromMid16AVX2(dst *uint16, dstStride int, src *int16, srcStride, tapStride int, mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)
TEXT ·mcBilinFromMid16AVX2(SB), NOSPLIT, $0-100
	BSETUP

brows3:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  bnarrow3

bcols3:
	MOVQ R11, CX
	TAPS2Y
	BCLIPY
	VPACKUSDW Y13, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       bcols3
	TESTQ     DX, DX
	JZ        bnext3

bnarrow3:
	MOVQ R11, CX
	TAPS2X
	BCLIPX
	VPACKUSDW X13, X0, X0
	VMOVDQU   X0, (R13)

bnext3:
	BROWSTEP(brows3)
	VZEROUPPER
	RET

// func mcBilinPrepMid16AVX2(dst *int16, dstStride int, src *int16, srcStride, tapStride int, mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)
TEXT ·mcBilinPrepMid16AVX2(SB), NOSPLIT, $0-100
	BSETUP

brows4:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13
	CMPQ DX, $16
	JLT  bnarrow4

bcols4:
	MOVQ R11, CX
	TAPS2Y
	VPACKSSDW Y13, Y0, Y0
	VMOVDQU   Y0, (R13)
	ADDQ      $32, R11
	ADDQ      $32, R13
	SUBQ      $16, DX
	CMPQ      DX, $16
	JGE       bcols4
	TESTQ     DX, DX
	JZ        bnext4

bnarrow4:
	MOVQ R11, CX
	TAPS2X
	VPACKSSDW X13, X0, X0
	VMOVDQU   X0, (R13)

bnext4:
	BROWSTEP(brows4)
	VZEROUPPER
	RET
