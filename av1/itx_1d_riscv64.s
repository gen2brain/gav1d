//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// The butterfly is add, subtract, multiply by a coefficient and a rounding
// shift, all of which RVV has. The coefficients ride in scalar registers.
#define MACA(D, A, CA, B, CB, SH, T) \
	VMULVX  CA, A, D; \
	VMULVX  CB, B, T; \
	VADDVV  T, D, D;  \
	VSSRAVI SH, D, D

#define MACS(D, A, CA, B, CB, SH, T) \
	VMULVX  CA, A, D; \
	VMULVX  CB, B, T; \
	VSUBVV  T, D, D;  \
	VSSRAVI SH, D, D

#define R181(D, A, B, OP) \
	OP(D, A, B);      \
	VMULVX  X6, D, D; \
	VSSRAVI $8, D, D

#define ADDV(D, A, B) VADDVV B, A, D
#define SUBV(D, A, B) VSUBVV B, A, D

#define CLIP(V)        \
	VMAXVX X28, V, V; \
	VMINVX X29, V, V

#define DCT4(A0, A1, A2, A3, T0, T1, T2, T3) \
	R181(T0, A0, A2, ADDV);   \
	R181(T1, A0, A2, SUBV);   \
	MACA(T2, A1, X7, A3, X8, $12, T3); \
	VSUBVV A3, T2, T2;        \
	MACS(T3, A3, X7, A1, X8, $12, A0); \
	VADDVV A1, T3, T3;        \
	VADDVV T3, T0, A0;        \
	CLIP(A0);                 \
	VADDVV T2, T1, A1;        \
	CLIP(A1);                 \
	VSUBVV T2, T1, A2;        \
	CLIP(A2);                 \
	VSUBVV T3, T0, A3;        \
	CLIP(A3)

#define KONST         \
	MOV $181, X6; \
	MOV $1567, X7; \
	MOV $312, X8; \
	MOV $799, X9; \
	MOV $79, X11; \
	MOV $1703, X12; \
	MOV $1138, X13; \
	MOV $401, X14; \
	MOV $20, X15; \
	MOV $1583, X16; \
	MOV $1299, X17; \
	MOV $1931, X18; \
	MOV $484, X19; \
	MOV $176, X20; \
	MOV $1189, X21

#define SETUP                   \
	MOV tmp+0(FP), X10;     \
	MOV lanes+8(FP), X22;   \
	MOV stride+16(FP), X23; \
	MOVW lo+24(FP), X28;    \
	MOVW hi+28(FP), X29;    \
	SLLI $2, X23, X23;      \
	KONST

// LOADN and STOREN walk the rows, so the butterfly never addresses memory.
#define LD2(V0, V1)  \
	VLE32V (X24), V0; \
	ADD    X23, X24;  \
	VLE32V (X24), V1; \
	ADD    X23, X24

#define ST2(V0, V1)  \
	VSE32V V0, (X24); \
	ADD    X23, X24;  \
	VSE32V V1, (X24); \
	ADD    X23, X24

// ODD8 is the four-point odd half, from A1, A3, A5 and A7, leaving t4, t5, t6
// and t7 in T4, T5, T6 and T7.
#define ODD8(A1, A3, A5, A7, T4, T5, T6, T7, S) \
	MACA(T4, A1, X9, A7, X11, $12, S); \
	VSUBVV A7, T4, T4;                 \
	MACS(T5, A5, X12, A3, X13, $11, S); \
	MACA(T6, A5, X13, A3, X12, $11, S); \
	MACS(T7, A7, X9, A1, X11, $12, S); \
	VADDVV A1, T7, T7;                 \
	VADDVV T5, T4, A1;                 \
	CLIP(A1);                          \
	VSUBVV T5, T4, A3;                 \
	CLIP(A3);                          \
	VADDVV T6, T7, A7;                 \
	CLIP(A7);                          \
	VSUBVV T6, T7, A5;                 \
	CLIP(A5);                          \
	R181(T5, A5, A3, SUBV);            \
	R181(T6, A5, A3, ADDV);            \
	VMVVV A1, T4;                      \
	VMVVV A7, T7

// BFLY writes the pair one butterfly produces, leaving the sum in A.
#define BFLY(A, B, S) \
	VSUBVV B, A, S; \
	CLIP(S);        \
	VADDVV B, A, A; \
	CLIP(A);        \
	VMVVV  S, B

// func itxColDct4RVV(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct4RVV(SB), NOSPLIT, $0-32
	SETUP

col4:
	VSETVLI X22, E32, M1, TA, MA, X5
	MOV     X10, X24
	LD2(V0, V1)
	LD2(V2, V3)

	DCT4(V0, V1, V2, V3, V4, V5, V6, V7)

	MOV X10, X24
	ST2(V0, V1)
	ST2(V2, V3)

	SLLI $2, X5, X25
	ADD  X25, X10
	SUB  X5, X22
	BNEZ X22, col4

	RET

// func itxColDct8RVV(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct8RVV(SB), NOSPLIT, $0-32
	SETUP

col8:
	VSETVLI X22, E32, M1, TA, MA, X5
	MOV     X10, X24
	LD2(V0, V1)
	LD2(V2, V3)
	LD2(V4, V5)
	LD2(V6, V7)

	DCT4(V0, V2, V4, V6, V16, V17, V18, V19)
	ODD8(V1, V3, V5, V7, V16, V17, V18, V19, V20)

	BFLY(V0, V19, V20)
	BFLY(V2, V18, V20)
	BFLY(V4, V17, V20)
	BFLY(V6, V16, V20)

	MOV X10, X24
	ST2(V0, V2)
	ST2(V4, V6)
	ST2(V16, V17)
	ST2(V18, V19)

	SLLI $2, X5, X25
	ADD  X25, X10
	SUB  X5, X22
	BNEZ X22, col8

	RET

// func itxColDct16RVV(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct16RVV(SB), NOSPLIT, $0-32
	SETUP

col16:
	VSETVLI X22, E32, M1, TA, MA, X5
	MOV     X10, X24
	LD2(V0, V1)
	LD2(V2, V3)
	LD2(V4, V5)
	LD2(V6, V7)
	LD2(V8, V9)
	LD2(V10, V11)
	LD2(V12, V13)
	LD2(V14, V15)

	DCT4(V0, V4, V8, V12, V16, V17, V18, V19)
	ODD8(V2, V6, V10, V14, V16, V17, V18, V19, V20)

	VADDVV V19, V0, V20
	CLIP(V20)
	VSUBVV V19, V0, V27
	CLIP(V27)
	VADDVV V18, V4, V21
	CLIP(V21)
	VSUBVV V18, V4, V26
	CLIP(V26)
	VADDVV V17, V8, V22
	CLIP(V22)
	VSUBVV V17, V8, V25
	CLIP(V25)
	VADDVV V16, V12, V23
	CLIP(V23)
	VSUBVV V16, V12, V24
	CLIP(V24)

	// The sixteen-point odd half, into the registers the even rows vacated.
	MACA(V0, V1, X14, V15, X15, $12, V16)
	VSUBVV V15, V0, V0
	MACS(V14, V15, X14, V1, X15, $12, V16)
	VADDVV V1, V14, V14

	MACS(V2, V9, X16, V7, X17, $11, V16)
	MACA(V12, V9, X17, V7, X16, $11, V16)

	MACA(V4, V5, X18, V11, X19, $12, V16)
	VSUBVV V11, V4, V4
	MACS(V10, V11, X18, V5, X19, $12, V16)
	VADDVV V5, V10, V10

	VMULVX X20, V13, V6
	VMULVX X21, V3, V16
	VADDVV V16, V6, V6
	VRSUBVX X0, V6, V6
	VSSRAVI $12, V6, V6
	VADDVV V13, V6, V6
	MACS(V8, V13, X21, V3, X20, $12, V16)
	VADDVV V3, V8, V8

	VADDVV V2, V0, V1
	CLIP(V1)
	VSUBVV V2, V0, V3
	CLIP(V3)
	VSUBVV V4, V6, V5
	CLIP(V5)
	VADDVV V4, V6, V7
	CLIP(V7)
	VADDVV V10, V8, V9
	CLIP(V9)
	VSUBVV V10, V8, V11
	CLIP(V11)
	VSUBVV V12, V14, V13
	CLIP(V13)
	VADDVV V12, V14, V15
	CLIP(V15)

	MACA(V0, V13, X7, V3, X8, $12, V16)
	VSUBVV V3, V0, V0
	MACS(V2, V3, X7, V13, X8, $12, V16)
	VADDVV V13, V2, V2
	MACS(V4, V11, X8, V5, X7, $12, V16)
	VSUBVV V11, V4, V4
	MACA(V6, V11, X7, V5, X8, $12, V16)
	VSUBVV V5, V6, V6

	VADDVV V7, V1, V8
	CLIP(V8)
	VSUBVV V7, V1, V10
	CLIP(V10)
	VADDVV V4, V0, V12
	CLIP(V12)
	VSUBVV V4, V0, V14
	CLIP(V14)
	VSUBVV V9, V15, V16
	CLIP(V16)
	VADDVV V9, V15, V1
	CLIP(V1)
	VSUBVV V6, V2, V3
	CLIP(V3)
	VADDVV V6, V2, V5
	CLIP(V5)

	R181(V7, V3, V14, SUBV)
	R181(V9, V3, V14, ADDV)
	R181(V11, V16, V10, SUBV)
	R181(V13, V16, V10, ADDV)

	BFLY(V20, V1, V0)
	BFLY(V21, V5, V0)
	BFLY(V22, V9, V0)
	BFLY(V23, V13, V0)
	BFLY(V24, V11, V0)
	BFLY(V25, V7, V0)
	BFLY(V26, V12, V0)
	BFLY(V27, V8, V0)

	MOV X10, X24
	ST2(V20, V21)
	ST2(V22, V23)
	ST2(V24, V25)
	ST2(V26, V27)
	ST2(V8, V12)
	ST2(V7, V11)
	ST2(V13, V9)
	ST2(V5, V1)

	SLLI $2, X5, X25
	ADD  X25, X10
	SUB  X5, X22
	BNEZ X22, col16

	RET

// func itxIdentityRVV(tmp *int32, count, stride, n int, p *[4]int32)
TEXT ·itxIdentityRVV(SB), NOSPLIT, $0-40
	MOV  tmp+0(FP), X10
	MOV  count+8(FP), X11
	MOV  stride+16(FP), X12
	MOV  n+24(FP), X13
	MOV  p+32(FP), X14
	MOVW (X14), X15
	MOVW 4(X14), X16
	MOVW 8(X14), X17
	MOVW 12(X14), X18
	SLLI $2, X12, X12

idtap:
	MOV X10, X24
	MOV X11, X25

idlane:
	VSETVLI X25, E32, M1, TA, MA, X5
	VLE32V  (X24), V0
	VMULVX  X15, V0, V1
	VMULVX  X16, V0, V0
	VADDVX  X17, V0, V0
	VSRAVX  X18, V0, V0
	VADDVV  V1, V0, V0
	VSE32V  V0, (X24)

	SLLI $2, X5, X26
	ADD  X26, X24
	SUB  X5, X25
	BNEZ X25, idlane

	ADD  X12, X10
	SUB  $1, X13
	BNEZ X13, idtap

	RET
