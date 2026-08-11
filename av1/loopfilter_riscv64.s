//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// The four lanes are the four lines of one edge. A strided load reads them
// directly whichever way the edge runs, so unlike the other two architectures
// nothing has to be transposed. There are enough vector registers to hold
// every tap, so the filter itself touches no memory.

#define VLEW VLE16V
#define VSEW VSE16V
#define SGN  15

#define E      0(X11)
#define I      4(X11)
#define H      8(X11)
#define FTHR   12(X11)
#define LIM1   16(X11)
#define NEGLIM 20(X11)
#define PIXMAX 24(X11)
#define C4     28(X11)
#define C3     32(X11)
#define C1     36(X11)
#define WD     40(X11)

// V1 to V14 are p6 down to q6. X12 is the centre of the widened samples and
// X13 the centre of the outputs, both only used either side of the filter.
#define P6 V1
#define P5 V2
#define P4 V3
#define P3 V4
#define P2 V5
#define P1 V6
#define P0 V7
#define Q0 V8
#define Q1 V9
#define Q2 V10
#define Q3 V11
#define Q4 V12
#define Q5 V13
#define Q6 V14

#define STOREO(OFF, VS)       \
	ADD    $OFF, X13, X6; \
	VSEW VS, (X6)

#define LOADS(OFF, VD)        \
	ADD    $OFF, X12, X6; \
	VLEW (X6), VD

// GTA leaves set the lanes where |A-B| exceeds THRV. The sign bit of the
// difference is the mask, so no mask register is involved.
#define GTA(A, B, THRV, T)  \
	VSUBVV  B, A, T;    \
	VRSUBVX X0, T, V24; \
	VMAXVV  V24, T, T;  \
	VSUBVV  T, THRV, T; \
	VSRAVI  $SGN, T, T

#define ACCGT(A, B, THRV, T, M) \
	GTA(A, B, THRV, T);     \
	VORVV T, M, M

#define CLIPD(V, LO, HI) \
	VMAXVX LO, V, V; \
	VMINVX HI, V, V

// SEL keeps OLD wherever M is set.
#define SEL(M, OLD, NEW, T) \
	VXORVV NEW, OLD, T; \
	VANDVV M, T, T;     \
	VXORVV T, NEW, T

// WIDE8 and WIDE6 are one output each of the eight and six tap filters. D is
// the doubled sample, V25 carries the rounding term and V17 the mask that
// keeps whatever the narrow filter already wrote.
#define WIDE8(A, B, C, D, F, G, K, DST) \
	VADDVV B, A, V18;               \
	VADDVV C, V18, V18;             \
	VADDVV D, V18, V18;             \
	VADDVV D, V18, V18;             \
	VADDVV F, V18, V18;             \
	VADDVV G, V18, V18;             \
	VADDVV K, V18, V18;             \
	VADDVV V25, V18, V18;           \
	VSRAVI $3, V18, V18;            \
	ADD    $DST, X13, X6;           \
	VLEW (X6), V19;               \
	SEL(V17, V19, V18, V20);        \
	VSEW V20, (X6)

#define WIDE6(A, B, C, D, F, DST) \
	VADDVV B, A, V18;         \
	VADDVV B, V18, V18;       \
	VADDVV C, V18, V18;       \
	VADDVV C, V18, V18;       \
	VADDVV D, V18, V18;       \
	VADDVV D, V18, V18;       \
	VADDVV F, V18, V18;       \
	VADDVV V25, V18, V18;     \
	VSRAVI $3, V18, V18;      \
	ADD    $DST, X13, X6;     \
	VLEW (X6), V19;         \
	SEL(V17, V19, V18, V20);  \
	VSEW V20, (X6)

// TAP14 is one output of the fourteen tap filter: the sliding window plus the
// middle three samples again, then the window moves on.
#define TAP14(A, B, C, ADDV, SUBV, DST) \
	VADDVV B, A, V18;               \
	VADDVV C, V18, V18;             \
	VADDVV V26, V18, V18;           \
	VADDVV V25, V18, V18;           \
	VSRAVI $4, V18, V18;            \
	ADD    $DST, X13, X6;           \
	VLEW (X6), V19;               \
	SEL(V17, V19, V18, V20);        \
	VSEW V20, (X6);               \
	VADDVV ADDV, V26, V26;          \
	VSUBVV SUBV, V26, V26

#define TAP14L(A, B, C, DST)      \
	VADDVV B, A, V18;         \
	VADDVV C, V18, V18;       \
	VADDVV V26, V18, V18;     \
	VADDVV V25, V18, V18;     \
	VSRAVI $4, V18, V18;      \
	ADD    $DST, X13, X6;     \
	VLEW (X6), V19;         \
	SEL(V17, V19, V18, V20);  \
	VSEW V20, (X6)

#define ALLSET(M, LABEL)     \
	VREDANDVS M, M, V24; \
	VMVXS  V24, X6;      \
	ADD    $1, X6, X6;   \
	BEQ    X0, X6, LABEL

#define BCAST(SRC, VD) \
	MOVW  SRC, X6; \
	VMVVX X6, VD

#define TAPLOADS \
	LOADS(TP1, P1); \
	LOADS(TP0, P0); \
	LOADS(TQ0, Q0); \
	LOADS(TQ1, Q1); \
	MOV  $4, X6;    \
	BGE  X6, X18, taps; \
	LOADS(TP2, P2); \
	LOADS(TQ2, Q2); \
	MOV  $6, X6;    \
	BGE  X6, X18, taps; \
	LOADS(TP3, P3); \
	LOADS(TQ3, Q3); \
	MOV  $16, X6;   \
	BLT  X18, X6, taps; \
	LOADS(TP6, P6); \
	LOADS(TP5, P5); \
	LOADS(TP4, P4); \
	LOADS(TQ4, Q4); \
	LOADS(TQ5, Q5); \
	LOADS(TQ6, Q6)

#define TP6 -112
#define TP5 -96
#define TP4 -80
#define TP3 -64
#define TP2 -48
#define TP1 -32
#define TP0 -16
#define TQ0 0
#define TQ1 16
#define TQ2 32
#define TQ3 48
#define TQ4 64
#define TQ5 80
#define TQ6 96

#define OP5 -96
#define OP4 -80
#define OP3 -64
#define OP2 -48
#define OP1 -32
#define OP0 -16
#define OQ0 0
#define OQ1 16
#define OQ2 32
#define OQ3 48
#define OQ4 64
#define OQ5 80

// LPFBODY masks the lanes and layers the filters, working in 32-bit lanes
// throughout, which is why both bit depths share it.
#define LPFBODY \
	BCAST(I, V21); \
	BCAST(H, V22); \
	GTA(P1, P0, V21, V15); \
	GTA(P1, P0, V22, V16); \
	ACCGT(Q1, Q0, V21, V18, V15); \
	ACCGT(Q1, Q0, V22, V18, V16); \
	VSUBVV  Q0, P0, V18; \
	VRSUBVX X0, V18, V19; \
	VMAXVV  V19, V18, V18; \
	VADDVV  V18, V18, V18; \
	VSUBVV  Q1, P1, V19; \
	VRSUBVX X0, V19, V20; \
	VMAXVV  V20, V19, V19; \
	VSRLVI  $1, V19, V19; \
	VADDVV  V19, V18, V18; \
	BCAST(E, V23); \
	VSUBVV  V18, V23, V18; \
	VSRAVI  $SGN, V18, V18; \
	VORVV   V18, V15, V15; \
	MOV  $4, X6; \
	BGE  X6, X18, fmdone; \
	ACCGT(P1, P2, V21, V18, V15); \
	ACCGT(Q1, Q2, V21, V18, V15); \
	MOV  $6, X6; \
	BGE  X6, X18, fmdone; \
	ACCGT(P3, P2, V21, V18, V15); \
	ACCGT(Q3, Q2, V21, V18, V15); \
	fmdone:; \
	ALLSET(V15, done); \
	MOVW   NEGLIM, X20; \
	MOVW   LIM1, X21; \
	VSUBVV Q1, P1, V18; \
	CLIPD(V18, X20, X21); \
	VANDVV V16, V18, V18; \
	VSUBVV P0, Q0, V19; \
	VADDVV V19, V19, V20; \
	VADDVV V19, V20, V20; \
	VADDVV V20, V18, V18; \
	CLIPD(V18, X20, X21); \
	MOVW   C4, X6; \
	VADDVX X6, V18, V19; \
	VMINVX X21, V19, V19; \
	VSRAVI $3, V19, V19; \
	MOVW   C3, X6; \
	VADDVX X6, V18, V20; \
	VMINVX X21, V20, V20; \
	VSRAVI $3, V20, V20; \
	MOVW   PIXMAX, X22; \
	VADDVV V20, P0, V21; \
	CLIPD(V21, X0, X22); \
	VSUBVV V19, Q0, V22; \
	CLIPD(V22, X0, X22); \
	MOVW   C1, X6; \
	VADDVX X6, V19, V23; \
	VSRAVI $1, V23, V23; \
	VADDVV V23, P1, V18; \
	CLIPD(V18, X0, X22); \
	SEL(V16, P1, V18, V20); \
	VSUBVV V23, Q1, V18; \
	CLIPD(V18, X0, X22); \
	SEL(V16, Q1, V18, V19); \
	SEL(V15, P1, V20, V18); \
	STOREO(OP1, V18); \
	SEL(V15, P0, V21, V18); \
	STOREO(OP0, V18); \
	SEL(V15, Q0, V22, V18); \
	STOREO(OQ0, V18); \
	SEL(V15, Q1, V19, V18); \
	STOREO(OQ1, V18); \
	MOV $-2, X19; \
	MOV $4, X16; \
	MOV $4, X6; \
	BGE X6, X18, scatter; \
	BCAST(FTHR, V21); \
	VORVV V15, V15, V17; \
	ACCGT(P0, P2, V21, V18, V17); \
	ACCGT(P1, P0, V21, V18, V17); \
	ACCGT(Q1, Q0, V21, V18, V17); \
	ACCGT(Q0, Q2, V21, V18, V17); \
	MOV   $8, X6; \
	BLT   X18, X6, flatindone; \
	ACCGT(P0, P3, V21, V18, V17); \
	ACCGT(Q0, Q3, V21, V18, V17); \
	flatindone:; \
	ALLSET(V17, scatter); \
	BCAST(C4, V25); \
	MOV  $6, X6; \
	BLT  X6, X18, eight; \
	WIDE6(P2, P2, P1, P0, Q0, OP1); \
	WIDE6(P2, P1, P0, Q0, Q1, OP0); \
	WIDE6(P1, P0, Q0, Q1, Q2, OQ0); \
	WIDE6(P0, Q0, Q1, Q2, Q2, OQ1); \
	JMP  scatter; \
	eight:; \
	MOV $-3, X19; \
	MOV $6, X16; \
	WIDE8(P3, P3, P3, P2, P1, P0, Q0, OP2); \
	WIDE8(P3, P3, P2, P1, P0, Q0, Q1, OP1); \
	WIDE8(P3, P2, P1, P0, Q0, Q1, Q2, OP0); \
	WIDE8(P2, P1, P0, Q0, Q1, Q2, Q3, OQ0); \
	WIDE8(P1, P0, Q0, Q1, Q2, Q3, Q3, OQ1); \
	WIDE8(P0, Q0, Q1, Q2, Q3, Q3, Q3, OQ2); \
	MOV $16, X6; \
	BLT X18, X6, scatter; \
	BCAST(FTHR, V21); \
	ACCGT(P0, P6, V21, V18, V17); \
	ACCGT(P0, P5, V21, V18, V17); \
	ACCGT(P0, P4, V21, V18, V17); \
	ACCGT(Q0, Q4, V21, V18, V17); \
	ACCGT(Q0, Q5, V21, V18, V17); \
	ACCGT(Q0, Q6, V21, V18, V17); \
	ALLSET(V17, scatter); \
	VADDVV P6, P6, V26; \
	VADDVV V26, V26, V26; \
	VADDVV P6, V26, V26; \
	VADDVV P6, V26, V26; \
	VADDVV P5, V26, V26; \
	VADDVV P4, V26, V26; \
	VADDVV P3, V26, V26; \
	VADDVV P2, V26, V26; \
	VADDVV P1, V26, V26; \
	VADDVV P0, V26, V26; \
	VADDVV Q0, V26, V26; \
	BCAST(C4, V25); \
	VADDVV V25, V25, V25; \
	TAP14(P6, P5, P4, Q1, P6, OP5); \
	TAP14(P5, P4, P3, Q2, P6, OP4); \
	TAP14(P4, P3, P2, Q3, P6, OP3); \
	TAP14(P3, P2, P1, Q4, P6, OP2); \
	TAP14(P2, P1, P0, Q5, P6, OP1); \
	TAP14(P1, P0, Q0, Q6, P6, OP0); \
	TAP14(P0, Q0, Q1, Q6, P5, OQ0); \
	TAP14(Q0, Q1, Q2, Q6, P4, OQ1); \
	TAP14(Q1, Q2, Q3, Q6, P3, OQ2); \
	TAP14(Q2, Q3, Q4, Q6, P2, OQ3); \
	TAP14(Q3, Q4, Q5, Q6, P1, OQ4); \
	TAP14L(Q4, Q5, Q6, OQ5); \
	MOV $-6, X19; \
	MOV $12, X16;

// func lpfVL16RVV() int
TEXT ·lpfVL16RVV(SB), NOSPLIT, $0-8
	MOV     $16, X5
	VSETVLI X5, E16, M1, TA, MA, X6
	MOV     X6, ret+0(FP)
	RET

// func lpf8RVV(dst *uint8, stridea, strideb int, p *lpfParams, n int)
TEXT ·lpf8RVV(SB), NOSPLIT, $688-40
	MOV  dst+0(FP), X10
	MOV  stridea+8(FP), X14
	MOV  strideb+16(FP), X15
	MOV  p+24(FP), X11
	MOV  n+32(FP), X24
	ADD  $128, X2, X12
	ADD  $448, X2, X13
	ADD  $568, X2, X23

	MOV     $8, X17
	VSETVLI X17, E16, M1, TA, MA, X7

	MOVW WD, X18
	MOV  $-2, X19
	MOV  $4, X16
	MOV  $4, X6
	BGE  X6, X18, spanset
	MOV  $-3, X19
	MOV  $6, X16
	MOV  $6, X6
	BGE  X6, X18, spanset
	MOV  $-4, X19
	MOV  $8, X16
	MOV  $16, X6
	BLT  X18, X6, spanset
	MOV  $-7, X19
	MOV  $14, X16

spanset:
	SLLI $3, X19, X6
	ADD  X23, X6, X22
	ADD  $56, X22, X22
	MOV  $1, X6
	BNE  X6, X14, gatherrows

	// The lines are contiguous, so one strided load takes every tap. Their
	// elements are only four byte aligned, so eight lines take two.
	MUL     X15, X19, X8
	ADD     X10, X8, X8
	VSETVLI X16, E32, M2, TA, MA, X7
	VLSE32V (X8), X15, V8
	MOV     $8, X6
	VSSE32V V8, X6, (X22)
	ADD     $4, X22, X9
	MOV     $8, X20
	BNE     X20, X24, gatherdup
	ADD     $4, X8, X8
	VLSE32V (X8), X15, V8

gatherdup:
	VSSE32V V8, X6, (X9)
	JMP     widen

gatherrows:
	// The taps run along each line instead, so the rows are read whole and
	// the segment store interleaves them into tap order.
	ADD     X19, X10, X8
	VSETVLI X16, E8, M1, TA, MA, X7
	VLE8V   (X8), V8
	ADD     X14, X8, X8
	VLE8V   (X8), V9
	ADD     X14, X8, X8
	VLE8V   (X8), V10
	ADD     X14, X8, X8
	VLE8V   (X8), V11
	MOV     $8, X6
	BNE     X6, X24, gatherrows4

	ADD   X14, X8, X8
	VLE8V (X8), V12
	ADD   X14, X8, X8
	VLE8V (X8), V13
	ADD   X14, X8, X8
	VLE8V (X8), V14
	ADD   X14, X8, X8
	VLE8V (X8), V15
	JMP   rowstore

gatherrows4:
	VMVVV V8, V12
	VMVVV V9, V13
	VMVVV V10, V14
	VMVVV V11, V15

rowstore:
	VSSEG8E8V V8, (X22)

widen:
	SLLI     $3, X16, X20
	VSETVLI  X20, E8, M4, TA, MA, X7
	VLE8V    (X22), V8
	VSETVLI  X20, E16, M8, TA, MA, X7
	VZEXTVF2 V8, V16
	SLLI     $4, X19, X6
	ADD      X12, X6, X9
	VSE16V   V16, (X9)
	ADD      $320, X9, X6
	VSE16V   V16, (X6)
	VSETVLI  X17, E16, M1, TA, MA, X7

	TAPLOADS

taps:
	LPFBODY

scatter:
	SLLI    $4, X19, X6
	ADD     X13, X6, X9
	SLLI    $3, X16, X20
	VSETVLI X20, E16, M8, TA, MA, X7
	VLE16V  (X9), V16
	VSETVLI X20, E8, M4, TA, MA, X7
	VNSRLWI $0, V16, V8
	SLLI    $3, X19, X6
	ADD     X23, X6, X22
	ADD     $56, X22, X22
	VSE8V   V8, (X22)

	MOV  $1, X6
	BNE  X6, X14, scatterrows

	MUL     X15, X19, X8
	ADD     X10, X8, X8
	VSETVLI X16, E32, M2, TA, MA, X7
	MOV     $8, X6
	VLSE32V (X22), X6, V8
	VSSE32V V8, X15, (X8)
	MOV     $8, X20
	BNE     X20, X24, done
	ADD     $4, X22, X9
	VLSE32V (X9), X6, V8
	ADD     $4, X8, X8
	VSSE32V V8, X15, (X8)
	RET

scatterrows:
	VSETVLI   X16, E8, M1, TA, MA, X7
	VLSEG8E8V (X22), V8
	ADD       X19, X10, X8
	VSE8V     V8, (X8)
	ADD       X14, X8, X8
	VSE8V     V9, (X8)
	ADD       X14, X8, X8
	VSE8V     V10, (X8)
	ADD       X14, X8, X8
	VSE8V     V11, (X8)
	MOV       $8, X6
	BNE       X6, X24, done

	ADD   X14, X8, X8
	VSE8V V12, (X8)
	ADD   X14, X8, X8
	VSE8V V13, (X8)
	ADD   X14, X8, X8
	VSE8V V14, (X8)
	ADD   X14, X8, X8
	VSE8V V15, (X8)

done:
	RET

#undef TP6
#undef TP5
#undef TP4
#undef TP3
#undef TP2
#undef TP1
#undef TP0
#undef TQ0
#undef TQ1
#undef TQ2
#undef TQ3
#undef TQ4
#undef TQ5
#undef TQ6
#undef OP5
#undef OP4
#undef OP3
#undef OP2
#undef OP1
#undef OP0
#undef OQ0
#undef OQ1
#undef OQ2
#undef OQ3
#undef OQ4
#undef OQ5

// Sixteen lines fill a 256 bit register at sixteen bit elements, so every slot
// and every output doubles.
#define TP6 -224
#define TP5 -192
#define TP4 -160
#define TP3 -128
#define TP2 -96
#define TP1 -64
#define TP0 -32
#define TQ0 0
#define TQ1 32
#define TQ2 64
#define TQ3 96
#define TQ4 128
#define TQ5 160
#define TQ6 192

#define OP5 -192
#define OP4 -160
#define OP3 -128
#define OP2 -96
#define OP1 -64
#define OP0 -32
#define OQ0 0
#define OQ1 32
#define OQ2 64
#define OQ3 96
#define OQ4 128
#define OQ5 160

// func lpf8WideRVV(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8WideRVV(SB), $1424-32
	MOV  dst+0(FP), X10
	MOV  stridea+8(FP), X14
	MOV  strideb+16(FP), X15
	MOV  p+24(FP), X11
	ADD  $240, X2, X12
	ADD  $880, X2, X13
	ADD  $1168, X2, X23

	MOV     $16, X17
	VSETVLI X17, E16, M1, TA, MA, X7

	MOVW WD, X18
	MOV  $-2, X19
	MOV  $4, X16
	MOV  $4, X6
	BGE  X6, X18, wspanset
	MOV  $-3, X19
	MOV  $6, X16
	MOV  $6, X6
	BGE  X6, X18, wspanset
	MOV  $-4, X19
	MOV  $8, X16
	MOV  $16, X6
	BLT  X18, X6, wspanset
	MOV  $-7, X19
	MOV  $14, X16

wspanset:
	SLLI $4, X19, X6
	ADD  X23, X6, X22
	ADD  $112, X22, X22
	MOV  $16, X21
	MOV  $1, X6
	BNE  X6, X14, wgatherrows

	MUL     X15, X19, X8
	ADD     X10, X8, X8
	VSETVLI X16, E32, M2, TA, MA, X7
	VLSE32V (X8), X15, V8
	VSSE32V V8, X21, (X22)
	ADD     $4, X8, X9
	ADD     $4, X22, X6
	VLSE32V (X9), X15, V8
	VSSE32V V8, X21, (X6)
	ADD     $8, X8, X9
	ADD     $8, X22, X6
	VLSE32V (X9), X15, V8
	VSSE32V V8, X21, (X6)
	ADD     $12, X8, X9
	ADD     $12, X22, X6
	VLSE32V (X9), X15, V8
	VSSE32V V8, X21, (X6)
	JMP     wwiden

wgatherrows:
	ADD     X19, X10, X8
	VSETVLI X16, E8, M1, TA, MA, X7
	VLE8V   (X8), V8
	ADD     X14, X8, X8
	VLE8V   (X8), V9
	ADD     X14, X8, X8
	VLE8V   (X8), V10
	ADD     X14, X8, X8
	VLE8V   (X8), V11
	ADD     X14, X8, X8
	VLE8V   (X8), V12
	ADD     X14, X8, X8
	VLE8V   (X8), V13
	ADD     X14, X8, X8
	VLE8V   (X8), V14
	ADD     X14, X8, X8
	VLE8V   (X8), V15
	VSSSEG8E8V V8, X21, (X22)
	ADD     X14, X8, X8
	VLE8V   (X8), V8
	ADD     X14, X8, X8
	VLE8V   (X8), V9
	ADD     X14, X8, X8
	VLE8V   (X8), V10
	ADD     X14, X8, X8
	VLE8V   (X8), V11
	ADD     X14, X8, X8
	VLE8V   (X8), V12
	ADD     X14, X8, X8
	VLE8V   (X8), V13
	ADD     X14, X8, X8
	VLE8V   (X8), V14
	ADD     X14, X8, X8
	VLE8V   (X8), V15
	ADD     $8, X22, X6
	VSSSEG8E8V V8, X21, (X6)

	// Sixteen lines of more than eight taps overrun a single eight register
	// group, so the widening walks the taps eight at a time.
wwiden:
	SLLI $4, X19, X6
	ADD  X23, X6, X22
	ADD  $112, X22, X22
	SLLI $5, X19, X6
	ADD  X12, X6, X9
	MOV  X16, X21

wchunk:
	MOV  $8, X20
	BGE  X21, X20, wfull
	MOV  X21, X20

wfull:
	SLLI     $4, X20, X6
	VSETVLI  X6, E8, M4, TA, MA, X7
	VLE8V    (X22), V8
	VSETVLI  X6, E16, M8, TA, MA, X7
	VZEXTVF2 V8, V16
	VSE16V   V16, (X9)
	ADD      $640, X9, X6
	VSE16V   V16, (X6)
	SLLI     $4, X20, X6
	ADD      X6, X22, X22
	SLLI     $5, X20, X6
	ADD      X6, X9, X9
	SUB      X20, X21, X21
	BNEZ     X21, wchunk

	VSETVLI X17, E16, M1, TA, MA, X7
	TAPLOADS

taps:
	LPFBODY

scatter:
	SLLI $5, X19, X6
	ADD  X13, X6, X9
	SLLI $4, X19, X6
	ADD  X23, X6, X22
	ADD  $112, X22, X22
	MOV  X16, X21
	MOV  X22, X8

wschunk:
	MOV  $8, X20
	BGE  X21, X20, wsfull
	MOV  X21, X20

wsfull:
	SLLI    $4, X20, X6
	VSETVLI X6, E16, M8, TA, MA, X7
	VLE16V  (X9), V16
	VSETVLI X6, E8, M4, TA, MA, X7
	VNSRLWI $0, V16, V8
	VSE8V   V8, (X8)
	SLLI    $4, X20, X6
	ADD     X6, X8, X8
	SLLI    $5, X20, X6
	ADD     X6, X9, X9
	SUB     X20, X21, X21
	BNEZ    X21, wschunk

	MOV $16, X21
	MOV $1, X6
	BNE X6, X14, wscatterrows

	MUL     X15, X19, X8
	ADD     X10, X8, X8
	VSETVLI X16, E32, M2, TA, MA, X7
	VLSE32V (X22), X21, V8
	VSSE32V V8, X15, (X8)
	ADD     $4, X22, X6
	ADD     $4, X8, X9
	VLSE32V (X6), X21, V8
	VSSE32V V8, X15, (X9)
	ADD     $8, X22, X6
	ADD     $8, X8, X9
	VLSE32V (X6), X21, V8
	VSSE32V V8, X15, (X9)
	ADD     $12, X22, X6
	ADD     $12, X8, X9
	VLSE32V (X6), X21, V8
	VSSE32V V8, X15, (X9)
	RET

wscatterrows:
	VSETVLI    X16, E8, M1, TA, MA, X7
	VLSSEG8E8V (X22), X21, V8
	ADD        X19, X10, X8
	VSE8V      V8, (X8)
	ADD        X14, X8, X8
	VSE8V      V9, (X8)
	ADD        X14, X8, X8
	VSE8V      V10, (X8)
	ADD        X14, X8, X8
	VSE8V      V11, (X8)
	ADD        X14, X8, X8
	VSE8V      V12, (X8)
	ADD        X14, X8, X8
	VSE8V      V13, (X8)
	ADD        X14, X8, X8
	VSE8V      V14, (X8)
	ADD        X14, X8, X8
	VSE8V      V15, (X8)
	ADD        $8, X22, X6
	VLSSEG8E8V (X6), X21, V8
	ADD        X14, X8, X8
	VSE8V      V8, (X8)
	ADD        X14, X8, X8
	VSE8V      V9, (X8)
	ADD        X14, X8, X8
	VSE8V      V10, (X8)
	ADD        X14, X8, X8
	VSE8V      V11, (X8)
	ADD        X14, X8, X8
	VSE8V      V12, (X8)
	ADD        X14, X8, X8
	VSE8V      V13, (X8)
	ADD        X14, X8, X8
	VSE8V      V14, (X8)
	ADD        X14, X8, X8
	VSE8V      V15, (X8)

done:
	RET

#undef OP5
#undef OP4
#undef OP3
#undef OP2
#undef OP1
#undef OP0
#undef OQ0
#undef OQ1
#undef OQ2
#undef OQ3
#undef OQ4
#undef OQ5
#define OP5 -96
#define OP4 -80
#define OP3 -64
#define OP2 -48
#define OP1 -32
#define OP0 -16
#define OQ0 0
#define OQ1 16
#define OQ2 32
#define OQ3 48
#define OQ4 64
#define OQ5 80

#undef VLEW
#undef VSEW
#undef SGN
#define VLEW VLE32V
#define VSEW VSE32V
#define SGN  31

// func lpf16RVV(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16RVV(SB), NOSPLIT, $736-32
	MOV  dst+0(FP), X10
	MOV  stridea+8(FP), X14
	MOV  strideb+16(FP), X15
	MOV  p+24(FP), X11
	ADD  $216, X2, X12
	ADD  $536, X2, X13
	SLLI $1, X14, X28

	MOV     $4, X17
	VSETVLI X17, E32, M1, TA, MA, X7

	MOVW WD, X18
	MOV  $-2, X19
	MOV  $4, X16
	MOV  $4, X6
	BGE  X6, X18, spanset
	MOV  $-3, X19
	MOV  $6, X16
	MOV  $6, X6
	BGE  X6, X18, spanset
	MOV  $-4, X19
	MOV  $8, X16
	MOV  $16, X6
	BLT  X18, X6, spanset
	MOV  $-7, X19
	MOV  $14, X16

spanset:
	MUL  X15, X19, X8
	SLLI $1, X8, X8
	ADD  X10, X8, X8
	SLLI $4, X19, X9
	ADD  X12, X9, X9

gather16:
	VSETVLI  X17, E16, MF2, TA, MA, X7
	VLSE16V  (X8), X28, V8
	VSETVLI  X17, E32, M1, TA, MA, X7
	VZEXTVF2 V8, V9
	VSE32V   V9, (X9)
	ADD      $320, X9, X6
	VSE32V   V9, (X6)
	SLLI     $1, X15, X6
	ADD      X6, X8, X8
	ADD      $16, X9, X9
	SUB      $1, X16, X16
	BNE      X0, X16, gather16

	LOADS(-32, P1)
	LOADS(-16, P0)
	LOADS(0, Q0)
	LOADS(16, Q1)
	MOV  $4, X6
	BGE  X6, X18, taps16
	LOADS(-48, P2)
	LOADS(32, Q2)
	MOV  $6, X6
	BGE  X6, X18, taps16
	LOADS(-64, P3)
	LOADS(48, Q3)
	MOV  $16, X6
	BLT  X18, X6, taps16
	LOADS(-112, P6)
	LOADS(-96, P5)
	LOADS(-80, P4)
	LOADS(64, Q4)
	LOADS(80, Q5)
	LOADS(96, Q6)

taps16:
	LPFBODY

scatter:
	MOVW WD, X18
	MOV  $-2, X19
	MOV  $4, X16
	MOV  $6, X6
	BGE  X6, X18, scattersetx
	MOV  $-3, X19
	MOV  $6, X16
	MOV  $16, X6
	BLT  X18, X6, scattersetx
	MOV  $-6, X19
	MOV  $12, X16

scattersetx:
	MUL  X15, X19, X8
	SLLI $1, X8, X8
	ADD  X10, X8, X8
	SLLI $4, X19, X9
	ADD  X13, X9, X9

scatterxloop:
	VSETVLI X17, E32, M1, TA, MA, X7
	VLE32V  (X9), V8
	VSETVLI X17, E16, MF2, TA, MA, X7
	VNSRLWI $0, V8, V9
	VSSE16V V9, X28, (X8)
	VSETVLI X17, E32, M1, TA, MA, X7
	SLLI    $1, X15, X6
	ADD     X6, X8, X8
	ADD     $16, X9, X9
	SUB     $1, X16, X16
	BNE     X0, X16, scatterxloop

done:
	RET
