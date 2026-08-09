//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of these.
#define SMAXS(Vd, Vn, Vm) WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm) WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define CMGTS(Vd, Vn, Vm) WORD $(0x4EA03400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define ABSS(Vd, Vn)      WORD $(0x4EA0B800 | ((Vn) << 5) | (Vd))
#define SSHR(Vd, Vn, SH)  WORD $(0x4F000400 | (((64 - (SH)) & 0x7f) << 16) | ((Vn) << 5) | (Vd))
#define XTNS(Vd, Vn)      WORD $(0x0E612800 | ((Vn) << 5) | (Vd))
#define XTNH(Vd, Vn)      WORD $(0x0E212800 | ((Vn) << 5) | (Vd))

#define E      0(R1)
#define I      4(R1)
#define H      8(R1)
#define FTHR   12(R1)
#define LIM1   16(R1)
#define NEGLIM 20(R1)
#define PIXMAX 24(R1)
#define C4     28(R1)
#define C3     32(R1)
#define C1     36(R1)
#define WD     40(R1)

// R2 is the centre of the source array, which holds every sample widened to a
// lane and replicated past both ends for the sliding window. The output array
// sits 320 bytes above it and the packed bytes 496.
#define SP6 -112
#define SP5 -96
#define SP4 -80
#define SP3 -64
#define SP2 -48
#define SP1 -32
#define SP0 -16
#define SQ0 0
#define SQ1 16
#define SQ2 32
#define SQ3 48
#define SQ4 64
#define SQ5 80
#define SQ6 96

#define OP2 272
#define OP1 288
#define OP0 304
#define OQ0 320
#define OQ1 336
#define OQ2 352

#define BCAST(SRC, VD)   \
	MOVW SRC, R9;    \
	VDUP R9, VD.S4

#define LOAD(OFF, VD)      \
	ADD  $OFF, R2, R9; \
	VLD1 (R9), [VD.S4]

#define STORE(OFF, VS)     \
	ADD  $OFF, R2, R9; \
	VST1 [VS.S4], (R9)

// GTA leaves set the lanes where |A-B| exceeds the threshold in THRN. The
// hand-encoded forms take register numbers, so V14 is the fixed scratch.
#define GTA(A, B, THRN)          \
	VSUB B.S4, A.S4, V14.S4; \
	ABSS(14, 14);            \
	CMGTS(14, 14, THRN)

#define ACCGT(A, B, THRN, M)       \
	GTA(A, B, THRN);           \
	VORR V14.B16, M.B16, M.B16

// ACCGTM is the same with the second sample coming from the source array.
#define ACCGTM(A, OFF, THRN, M) \
	LOAD(OFF, V13);         \
	ACCGT(A, V13, THRN, M)

#define CLIPD(V, LO, HI) \
	SMAXS(V, V, LO); \
	SMINS(V, V, HI)

// SEL keeps the previous value wherever the mask in VM is set.
#define SEL(VM, OLD, NEW, VT)         \
	VMOV VM.B16, VT.B16;          \
	VBSL NEW.B16, OLD.B16, VT.B16

#define ADDS(OFF, ACC)              \
	LOAD(OFF, V13);             \
	VADD V13.S4, ACC.S4, ACC.S4

// WIDE8 and WIDE6 are one output each of the eight and six tap filters. D is
// the doubled sample, V29 carries the rounding term and V30 the mask that
// keeps whatever the narrow filter already wrote.
#define WIDE8(A, B, C, D, F, G, K, DST) \
	LOAD(A, V12);                   \
	ADDS(B, V12);                   \
	ADDS(C, V12);                   \
	ADDS(D, V12);                   \
	ADDS(D, V12);                   \
	ADDS(F, V12);                   \
	ADDS(G, V12);                   \
	ADDS(K, V12);                   \
	VADD V29.S4, V12.S4, V12.S4;    \
	SSHR(12, 12, 3);                \
	LOAD(DST, V14);                 \
	SEL(V30, V14, V12, V15);        \
	STORE(DST, V15)

#define WIDE6(A, B, C, D, F, DST)    \
	LOAD(A, V12);                \
	ADDS(B, V12);                \
	ADDS(B, V12);                \
	ADDS(C, V12);                \
	ADDS(C, V12);                \
	ADDS(D, V12);                \
	ADDS(D, V12);                \
	ADDS(F, V12);                \
	VADD V29.S4, V12.S4, V12.S4; \
	SSHR(12, 12, 3);             \
	LOAD(DST, V14);              \
	SEL(V30, V14, V12, V15);     \
	STORE(DST, V15)

// func lpf8NEON(dst *uint8, stridea, strideb int, p *lpfParams)
TEXT ·lpf8NEON(SB), NOSPLIT, $768-32
	MOVD dst+0(FP), R0
	MOVD stridea+8(FP), R3
	MOVD strideb+16(FP), R4
	MOVD p+24(FP), R1
	ADD  $208, RSP, R2

	MOVW WD, R10
	MOVD $-2, R11
	MOVD $4, R12
	CMP  $4, R10
	BLE  spanset
	MOVD $-3, R11
	MOVD $6, R12
	CMP  $6, R10
	BLE  spanset
	MOVD $-4, R11
	MOVD $8, R12
	CMP  $16, R10
	BLT  spanset
	MOVD $-7, R11
	MOVD $14, R12

spanset:
	CMP $1, R3
	BEQ gathercols

	LSL  $1, R3, R7
	ADD  R3, R7, R7
	CMP  $16, R10
	BGE  gatherwide

	MOVD  $-4, R11
	MOVD  $8, R12
	SUB   $4, R0, R8
	VLD1  (R8), [V0.D1]
	ADD   R3, R8, R9
	VLD1  (R9), [V1.D1]
	ADD   R3, R9, R9
	VLD1  (R9), [V2.D1]
	ADD   R3, R9, R9
	VLD1  (R9), [V3.D1]
	VZIP1 V1.B16, V0.B16, V4.B16
	VZIP1 V3.B16, V2.B16, V6.B16
	VZIP1 V6.H8, V4.H8, V8.H8
	VZIP2 V6.H8, V4.H8, V9.H8
	ADD   $508, R2, R9
	VST1  [V8.B16, V9.B16], (R9)
	B     widen

gatherwide:
	MOVD  $-7, R11
	MOVD  $16, R12
	SUB   $7, R0, R8
	VLD1  (R8), [V0.B16]
	ADD   R3, R8, R9
	VLD1  (R9), [V1.B16]
	ADD   R3, R9, R9
	VLD1  (R9), [V2.B16]
	ADD   R3, R9, R9
	VLD1  (R9), [V3.B16]
	VZIP1 V1.B16, V0.B16, V4.B16
	VZIP2 V1.B16, V0.B16, V5.B16
	VZIP1 V3.B16, V2.B16, V6.B16
	VZIP2 V3.B16, V2.B16, V7.B16
	VZIP1 V6.H8, V4.H8, V8.H8
	VZIP2 V6.H8, V4.H8, V9.H8
	VZIP1 V7.H8, V5.H8, V10.H8
	VZIP2 V7.H8, V5.H8, V11.H8
	ADD   $496, R2, R9
	VST1  [V8.B16, V9.B16, V10.B16, V11.B16], (R9)
	B     widen

gathercols:
	MUL  R4, R11, R8
	ADD  R0, R8, R8
	ADD  $496, R2, R9
	ADD  R11<<2, R9, R9
	ADD  $28, R9, R9
	MOVD R12, R13

gathercolsloop:
	MOVWU (R8), R14
	MOVW  R14, (R9)
	ADD   R4, R8, R8
	ADD   $4, R9, R9
	SUBS  $1, R13, R13
	BNE   gathercolsloop

widen:
	ADD  $496, R2, R8
	ADD  R11<<2, R8, R8
	ADD  $28, R8, R8
	ADD  R11<<4, R2, R9
	MOVD R12, R13

widenloop:
	MOVWU  (R8), R14
	VMOV   R14, V16.S[0]
	VUSHLL $0, V16.B8, V17.H8
	VUSHLL $0, V17.H4, V18.S4
	VST1   [V18.S4], (R9)
	ADD    $320, R9, R14
	VST1   [V18.S4], (R14)
	ADD    $4, R8, R8
	ADD    $16, R9, R9
	SUBS   $1, R13, R13
	BNE    widenloop

	LOAD(SP1, V0)
	LOAD(SP0, V1)
	LOAD(SQ0, V2)
	LOAD(SQ1, V3)
	BCAST(I, V20)
	BCAST(H, V21)
	VSUB V1.S4, V0.S4, V13.S4
	ABSS(13, 13)
	CMGTS(31, 13, 20)
	CMGTS(30, 13, 21)
	VSUB V2.S4, V3.S4, V13.S4
	ABSS(13, 13)
	CMGTS(14, 13, 20)
	VORR V14.B16, V31.B16, V31.B16
	CMGTS(14, 13, 21)
	VORR V14.B16, V30.B16, V30.B16
	ADD  $480, R2, R9
	VST1 [V30.S4], (R9)
	VSUB V2.S4, V1.S4, V13.S4
	ABSS(13, 13)
	VADD V13.S4, V13.S4, V13.S4
	VSUB V3.S4, V0.S4, V14.S4
	ABSS(14, 14)
	VUSHR $1, V14.S4, V14.S4
	VADD  V14.S4, V13.S4, V13.S4
	BCAST(E, V22)
	CMGTS(14, 13, 22)
	VORR V14.B16, V31.B16, V31.B16

	MOVW WD, R10
	CMP  $4, R10
	BLE  fmdone
	ACCGTM(V0, SP2, 20, V31)
	ACCGTM(V3, SQ2, 20, V31)
	CMP  $6, R10
	BLE  fmdone
	LOAD(SP3, V15)
	ACCGTM(V15, SP2, 20, V31)
	LOAD(SQ3, V15)
	ACCGTM(V15, SQ2, 20, V31)

fmdone:
	VMOV V31.D[0], R9
	VMOV V31.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  scatter

	// The narrow filter runs on every lane that got this far.
	ADD   $480, R2, R9
	VLD1  (R9), [V30.S4]
	BCAST(NEGLIM, V20)
	BCAST(LIM1, V21)
	VSUB  V3.S4, V0.S4, V13.S4
	CLIPD(13, 20, 21)
	VAND  V30.B16, V13.B16, V13.B16
	VSUB  V1.S4, V2.S4, V14.S4
	VADD  V14.S4, V14.S4, V15.S4
	VADD  V14.S4, V15.S4, V15.S4
	VADD  V15.S4, V13.S4, V13.S4
	CLIPD(13, 20, 21)
	BCAST(C4, V22)
	VADD  V22.S4, V13.S4, V14.S4
	SMINS(14, 14, 21)
	SSHR(14, 14, 3)
	BCAST(C3, V22)
	VADD  V22.S4, V13.S4, V15.S4
	SMINS(15, 15, 21)
	SSHR(15, 15, 3)
	VMOVI $0, V20.B16
	BCAST(PIXMAX, V21)
	VADD  V15.S4, V1.S4, V16.S4
	CLIPD(16, 20, 21)
	VSUB  V14.S4, V2.S4, V17.S4
	CLIPD(17, 20, 21)
	BCAST(C1, V22)
	VADD  V22.S4, V14.S4, V18.S4
	SSHR(18, 18, 1)
	VADD  V18.S4, V0.S4, V13.S4
	CLIPD(13, 20, 21)
	SEL(V30, V0, V13, V19)
	VSUB  V18.S4, V3.S4, V13.S4
	CLIPD(13, 20, 21)
	SEL(V30, V3, V13, V23)
	SEL(V31, V0, V19, V13)
	STORE(OP1, V13)
	SEL(V31, V1, V16, V13)
	STORE(OP0, V13)
	SEL(V31, V2, V17, V13)
	STORE(OQ0, V13)
	SEL(V31, V3, V23, V13)
	STORE(OQ1, V13)

	MOVW WD, R10
	CMP  $4, R10
	BLE  scatter

	// flat8in decides whether the six or eight tap filter replaces it.
	BCAST(FTHR, V20)
	VMOV V31.B16, V30.B16
	ACCGTM(V1, SP2, 20, V30)
	ACCGT(V0, V1, 20, V30)
	ACCGT(V3, V2, 20, V30)
	ACCGTM(V2, SQ2, 20, V30)
	CMP  $8, R10
	BLT  flatindone
	ACCGTM(V1, SP3, 20, V30)
	ACCGTM(V2, SQ3, 20, V30)

flatindone:
	VMOV V30.D[0], R9
	VMOV V30.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  scatter

	BCAST(C4, V29)
	CMP  $6, R10
	BGT  eight

	WIDE6(SP2, SP2, SP1, SP0, SQ0, OP1)
	WIDE6(SP2, SP1, SP0, SQ0, SQ1, OP0)
	WIDE6(SP1, SP0, SQ0, SQ1, SQ2, OQ0)
	WIDE6(SP0, SQ0, SQ1, SQ2, SQ2, OQ1)
	B     scatter

eight:
	WIDE8(SP3, SP3, SP3, SP2, SP1, SP0, SQ0, OP2)
	WIDE8(SP3, SP3, SP2, SP1, SP0, SQ0, SQ1, OP1)
	WIDE8(SP3, SP2, SP1, SP0, SQ0, SQ1, SQ2, OP0)
	WIDE8(SP2, SP1, SP0, SQ0, SQ1, SQ2, SQ3, OQ0)
	WIDE8(SP1, SP0, SQ0, SQ1, SQ2, SQ3, SQ3, OQ1)
	WIDE8(SP0, SQ0, SQ1, SQ2, SQ3, SQ3, SQ3, OQ2)

	CMP $16, R10
	BLT scatter

	// flat8out narrows it once more, to the fourteen tap filter.
	BCAST(FTHR, V20)
	ACCGTM(V1, SP6, 20, V30)
	ACCGTM(V1, SP5, 20, V30)
	ACCGTM(V1, SP4, 20, V30)
	ACCGTM(V2, SQ4, 20, V30)
	ACCGTM(V2, SQ5, 20, V30)
	ACCGTM(V2, SQ6, 20, V30)
	VMOV V30.D[0], R9
	VMOV V30.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  scatter

	// Replicating the ends removes every bounds test from the window.
	LOAD(SP6, V16)
	SUB  $208, R2, R9
	VST1 [V16.S4], (R9)
	SUB  $192, R2, R9
	VST1 [V16.S4], (R9)
	SUB  $176, R2, R9
	VST1 [V16.S4], (R9)
	SUB  $160, R2, R9
	VST1 [V16.S4], (R9)
	SUB  $144, R2, R9
	VST1 [V16.S4], (R9)
	SUB  $128, R2, R9
	VST1 [V16.S4], (R9)
	LOAD(SQ6, V16)
	ADD  $112, R2, R9
	VST1 [V16.S4], (R9)
	ADD  $128, R2, R9
	VST1 [V16.S4], (R9)
	ADD  $144, R2, R9
	VST1 [V16.S4], (R9)
	ADD  $160, R2, R9
	VST1 [V16.S4], (R9)
	ADD  $176, R2, R9
	VST1 [V16.S4], (R9)
	ADD  $192, R2, R9
	VST1 [V16.S4], (R9)

	SUB  $192, R2, R8
	VLD1 (R8), [V17.S4]
	MOVD $12, R13

sumwindow:
	ADD  $16, R8, R8
	VLD1 (R8), [V18.S4]
	VADD V18.S4, V17.S4, V17.S4
	SUBS $1, R13, R13
	BNE  sumwindow

	BCAST(C4, V29)
	VADD V29.S4, V29.S4, V29.S4
	SUB  $96, R2, R8
	ADD  $224, R2, R14
	MOVD $12, R13

fourteen:
	SUB  $16, R8, R9
	VLD1 (R9), [V19.S4]
	VLD1 (R8), [V20.S4]
	VADD V20.S4, V19.S4, V19.S4
	ADD  $16, R8, R9
	VLD1 (R9), [V20.S4]
	VADD V20.S4, V19.S4, V19.S4
	VADD V17.S4, V19.S4, V19.S4
	VADD V29.S4, V19.S4, V19.S4
	SSHR(19, 19, 4)
	VLD1 (R14), [V21.S4]
	SEL(V30, V21, V19, V22)
	VST1 [V22.S4], (R14)
	ADD  $112, R8, R9
	VLD1 (R9), [V20.S4]
	VADD V20.S4, V17.S4, V17.S4
	SUB  $96, R8, R9
	VLD1 (R9), [V20.S4]
	VSUB V20.S4, V17.S4, V17.S4
	ADD  $16, R8, R8
	ADD  $16, R14, R14
	SUBS $1, R13, R13
	BNE  fourteen

scatter:
	MOVW WD, R10
	MOVD $-2, R11
	MOVD $4, R12
	CMP  $6, R10
	BLE  scatterset
	MOVD $-3, R11
	MOVD $6, R12
	CMP  $16, R10
	BLT  scatterset
	MOVD $-6, R11
	MOVD $12, R12

scatterset:
	CMP $1, R3
	BEQ scattercols

	ADD  $496, R2, R8
	ADD  R11<<2, R8, R8
	ADD  $28, R8, R8
	MOVD $4, R15
	B    packloop

scattercols:
	MUL  R4, R11, R8
	ADD  R0, R8, R8
	MOVD R4, R15

packloop:
	ADD   R11<<4, R2, R9
	ADD   $320, R9, R9
	VLD1  (R9), [V16.S4]
	XTNS(17, 16)
	XTNH(17, 17)
	VMOV  V17.S[0], R14
	MOVW  R14, (R8)
	ADD   $1, R11, R11
	ADD   R15, R8, R8
	SUBS  $1, R12, R12
	BNE   packloop

	CMP $1, R3
	BEQ done

	LSL   $1, R3, R7
	ADD   R3, R7, R7
	MOVW  WD, R10
	CMP   $16, R10
	BGE   scatterwide

	ADD   $508, R2, R9
	VLD1  (R9), [V8.B16, V9.B16]
	VUZP1 V9.H8, V8.H8, V4.H8
	VUZP2 V9.H8, V8.H8, V5.H8
	VUZP1 V4.B16, V4.B16, V0.B16
	VUZP2 V4.B16, V4.B16, V1.B16
	VUZP1 V5.B16, V5.B16, V2.B16
	VUZP2 V5.B16, V5.B16, V3.B16
	SUB   $4, R0, R8
	VST1  [V0.D1], (R8)
	ADD   R3, R8, R9
	VST1  [V1.D1], (R9)
	ADD   R3, R9, R9
	VST1  [V2.D1], (R9)
	ADD   R3, R9, R9
	VST1  [V3.D1], (R9)
	B     done

scatterwide:
	ADD   $496, R2, R9
	VLD1  (R9), [V8.B16, V9.B16, V10.B16, V11.B16]
	VUZP1 V9.H8, V8.H8, V4.H8
	VUZP2 V9.H8, V8.H8, V5.H8
	VUZP1 V11.H8, V10.H8, V6.H8
	VUZP2 V11.H8, V10.H8, V7.H8
	VUZP1 V6.B16, V4.B16, V0.B16
	VUZP2 V6.B16, V4.B16, V1.B16
	VUZP1 V7.B16, V5.B16, V2.B16
	VUZP2 V7.B16, V5.B16, V3.B16
	SUB   $7, R0, R8
	VST1  [V0.B16], (R8)
	ADD   R3, R8, R9
	VST1  [V1.B16], (R9)
	ADD   R3, R9, R9
	VST1  [V2.B16], (R9)
	ADD   R3, R9, R9
	VST1  [V3.B16], (R9)

done:
	RET
