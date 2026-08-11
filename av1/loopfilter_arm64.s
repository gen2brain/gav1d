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

#define SMAXH(Vd, Vn, Vm) WORD $(0x4E606400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINH(Vd, Vn, Vm) WORD $(0x4E606C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define CMGTH(Vd, Vn, Vm) WORD $(0x4E603400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define ABSH(Vd, Vn)      WORD $(0x4E60B800 | ((Vn) << 5) | (Vd))
#define SSHRH(Vd, Vn, SH) WORD $(0x4F000400 | (((32 - (SH)) & 0x7f) << 16) | ((Vn) << 5) | (Vd))

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

#define MASKH 464

#define BCASTH(SRC, VD) \
	MOVW SRC, R9;   \
	VDUP R9, VD.H8

#define LOADH(OFF, VD)     \
	ADD  $OFF, R2, R9; \
	VLD1 (R9), [VD.H8]

#define STOREH(OFF, VS)    \
	ADD  $OFF, R2, R9; \
	VST1 [VS.H8], (R9)

#define GTAH(A, B, THRN)         \
	VSUB B.H8, A.H8, V14.H8; \
	ABSH(14, 14);            \
	CMGTH(14, 14, THRN)

#define ACCGTH(A, B, THRN, M)      \
	GTAH(A, B, THRN);          \
	VORR V14.B16, M.B16, M.B16

#define ACCGTMH(A, OFF, THRN, M) \
	LOADH(OFF, V13);         \
	ACCGTH(A, V13, THRN, M)

#define CLIPDH(V, LO, HI) \
	SMAXH(V, V, LO);  \
	SMINH(V, V, HI)

#define ADDSH(OFF, ACC)  \
	LOADH(OFF, V13); \
	VADD V13.H8, ACC.H8, ACC.H8

#define WIDE8H(A, B, C, D, F, G, K, DST) \
	LOADH(A, V12);                   \
	ADDSH(B, V12);                   \
	ADDSH(C, V12);                   \
	ADDSH(D, V12);                   \
	ADDSH(D, V12);                   \
	ADDSH(F, V12);                   \
	ADDSH(G, V12);                   \
	ADDSH(K, V12);                   \
	VADD V29.H8, V12.H8, V12.H8;     \
	SSHRH(12, 12, 3);                \
	LOADH(DST, V14);                 \
	SEL(V30, V14, V12, V15);         \
	STOREH(DST, V15)

#define WIDE6H(A, B, C, D, F, DST)   \
	LOADH(A, V12);               \
	ADDSH(B, V12);               \
	ADDSH(B, V12);               \
	ADDSH(C, V12);               \
	ADDSH(C, V12);               \
	ADDSH(D, V12);               \
	ADDSH(D, V12);               \
	ADDSH(F, V12);               \
	VADD V29.H8, V12.H8, V12.H8; \
	SSHRH(12, 12, 3);            \
	LOADH(DST, V14);             \
	SEL(V30, V14, V12, V15);     \
	STOREH(DST, V15)

// The eight rows in V0 to V7 come back transposed, so a byte lane that held
// one tap of one line ends up holding one line of one tap.
#define TRN8X8 \
	VTRN1 V1.B8, V0.B8, V8.B8;    \
	VTRN2 V1.B8, V0.B8, V9.B8;    \
	VTRN1 V3.B8, V2.B8, V10.B8;   \
	VTRN2 V3.B8, V2.B8, V11.B8;   \
	VTRN1 V5.B8, V4.B8, V12.B8;   \
	VTRN2 V5.B8, V4.B8, V13.B8;   \
	VTRN1 V7.B8, V6.B8, V14.B8;   \
	VTRN2 V7.B8, V6.B8, V15.B8;   \
	VTRN1 V10.H4, V8.H4, V16.H4;  \
	VTRN1 V11.H4, V9.H4, V17.H4;  \
	VTRN2 V10.H4, V8.H4, V18.H4;  \
	VTRN2 V11.H4, V9.H4, V19.H4;  \
	VTRN1 V14.H4, V12.H4, V20.H4; \
	VTRN1 V15.H4, V13.H4, V21.H4; \
	VTRN2 V14.H4, V12.H4, V22.H4; \
	VTRN2 V15.H4, V13.H4, V23.H4; \
	VTRN1 V20.S2, V16.S2, V0.S2;  \
	VTRN1 V21.S2, V17.S2, V1.S2;  \
	VTRN1 V22.S2, V18.S2, V2.S2;  \
	VTRN1 V23.S2, V19.S2, V3.S2;  \
	VTRN2 V20.S2, V16.S2, V4.S2;  \
	VTRN2 V21.S2, V17.S2, V5.S2;  \
	VTRN2 V22.S2, V18.S2, V6.S2;  \
	VTRN2 V23.S2, V19.S2, V7.S2

// R5 steps from the fourth row to the fifth, so a four line call sets it to
// walk back to the first and the top four lanes repeat the bottom four.
#define LOADROWS             \
	MOVD R8, R14;        \
	VLD1 (R14), [V0.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V1.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V2.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V3.B8]; \
	ADD  R5, R14, R14;   \
	VLD1 (R14), [V4.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V5.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V6.B8]; \
	ADD  R3, R14, R14;   \
	VLD1 (R14), [V7.B8]

#define STOREROWS            \
	MOVD R8, R14;        \
	VST1 [V0.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V1.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V2.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V3.B8], (R14); \
	ADD  R5, R14, R14;   \
	VST1 [V4.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V5.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V6.B8], (R14); \
	ADD  R3, R14, R14;   \
	VST1 [V7.B8], (R14)

// LPFBODY masks the lanes and layers the filters, working in 32-bit lanes
// throughout, which is why both bit depths share it.
#define LPFBODY \
	LOAD(SP1, V0); \
	LOAD(SP0, V1); \
	LOAD(SQ0, V2); \
	LOAD(SQ1, V3); \
	BCAST(I, V20); \
	BCAST(H, V21); \
	VSUB V1.S4, V0.S4, V13.S4; \
	ABSS(13, 13); \
	CMGTS(31, 13, 20); \
	CMGTS(30, 13, 21); \
	VSUB V2.S4, V3.S4, V13.S4; \
	ABSS(13, 13); \
	CMGTS(14, 13, 20); \
	VORR V14.B16, V31.B16, V31.B16; \
	CMGTS(14, 13, 21); \
	VORR V14.B16, V30.B16, V30.B16; \
	ADD  $480, R2, R9; \
	VST1 [V30.S4], (R9); \
	VSUB V2.S4, V1.S4, V13.S4; \
	ABSS(13, 13); \
	VADD V13.S4, V13.S4, V13.S4; \
	VSUB V3.S4, V0.S4, V14.S4; \
	ABSS(14, 14); \
	VUSHR $1, V14.S4, V14.S4; \
	VADD  V14.S4, V13.S4, V13.S4; \
	BCAST(E, V22); \
	CMGTS(14, 13, 22); \
	VORR V14.B16, V31.B16, V31.B16; \
	MOVW WD, R10; \
	CMP  $4, R10; \
	BLE  fmdone; \
	ACCGTM(V0, SP2, 20, V31); \
	ACCGTM(V3, SQ2, 20, V31); \
	CMP  $6, R10; \
	BLE  fmdone; \
	LOAD(SP3, V15); \
	ACCGTM(V15, SP2, 20, V31); \
	LOAD(SQ3, V15); \
	ACCGTM(V15, SQ2, 20, V31); \
	fmdone:; \
	VMOV V31.D[0], R9; \
	VMOV V31.D[1], R14; \
	AND  R14, R9, R9; \
	CMN  $1, R9; \
	BEQ  scatter; \
	ADD   $480, R2, R9; \
	VLD1  (R9), [V30.S4]; \
	BCAST(NEGLIM, V20); \
	BCAST(LIM1, V21); \
	VSUB  V3.S4, V0.S4, V13.S4; \
	CLIPD(13, 20, 21); \
	VAND  V30.B16, V13.B16, V13.B16; \
	VSUB  V1.S4, V2.S4, V14.S4; \
	VADD  V14.S4, V14.S4, V15.S4; \
	VADD  V14.S4, V15.S4, V15.S4; \
	VADD  V15.S4, V13.S4, V13.S4; \
	CLIPD(13, 20, 21); \
	BCAST(C4, V22); \
	VADD  V22.S4, V13.S4, V14.S4; \
	SMINS(14, 14, 21); \
	SSHR(14, 14, 3); \
	BCAST(C3, V22); \
	VADD  V22.S4, V13.S4, V15.S4; \
	SMINS(15, 15, 21); \
	SSHR(15, 15, 3); \
	VMOVI $0, V20.B16; \
	BCAST(PIXMAX, V21); \
	VADD  V15.S4, V1.S4, V16.S4; \
	CLIPD(16, 20, 21); \
	VSUB  V14.S4, V2.S4, V17.S4; \
	CLIPD(17, 20, 21); \
	BCAST(C1, V22); \
	VADD  V22.S4, V14.S4, V18.S4; \
	SSHR(18, 18, 1); \
	VADD  V18.S4, V0.S4, V13.S4; \
	CLIPD(13, 20, 21); \
	SEL(V30, V0, V13, V19); \
	VSUB  V18.S4, V3.S4, V13.S4; \
	CLIPD(13, 20, 21); \
	SEL(V30, V3, V13, V23); \
	SEL(V31, V0, V19, V13); \
	STORE(OP1, V13); \
	SEL(V31, V1, V16, V13); \
	STORE(OP0, V13); \
	SEL(V31, V2, V17, V13); \
	STORE(OQ0, V13); \
	SEL(V31, V3, V23, V13); \
	STORE(OQ1, V13); \
	MOVW WD, R10; \
	CMP  $4, R10; \
	BLE  scatter; \
	BCAST(FTHR, V20); \
	VMOV V31.B16, V30.B16; \
	ACCGTM(V1, SP2, 20, V30); \
	ACCGT(V0, V1, 20, V30); \
	ACCGT(V3, V2, 20, V30); \
	ACCGTM(V2, SQ2, 20, V30); \
	CMP  $8, R10; \
	BLT  flatindone; \
	ACCGTM(V1, SP3, 20, V30); \
	ACCGTM(V2, SQ3, 20, V30); \
	flatindone:; \
	VMOV V30.D[0], R9; \
	VMOV V30.D[1], R14; \
	AND  R14, R9, R9; \
	CMN  $1, R9; \
	BEQ  scatter; \
	BCAST(C4, V29); \
	CMP  $6, R10; \
	BGT  eight; \
	WIDE6(SP2, SP2, SP1, SP0, SQ0, OP1); \
	WIDE6(SP2, SP1, SP0, SQ0, SQ1, OP0); \
	WIDE6(SP1, SP0, SQ0, SQ1, SQ2, OQ0); \
	WIDE6(SP0, SQ0, SQ1, SQ2, SQ2, OQ1); \
	B     scatter; \
	eight:; \
	WIDE8(SP3, SP3, SP3, SP2, SP1, SP0, SQ0, OP2); \
	WIDE8(SP3, SP3, SP2, SP1, SP0, SQ0, SQ1, OP1); \
	WIDE8(SP3, SP2, SP1, SP0, SQ0, SQ1, SQ2, OP0); \
	WIDE8(SP2, SP1, SP0, SQ0, SQ1, SQ2, SQ3, OQ0); \
	WIDE8(SP1, SP0, SQ0, SQ1, SQ2, SQ3, SQ3, OQ1); \
	WIDE8(SP0, SQ0, SQ1, SQ2, SQ3, SQ3, SQ3, OQ2); \
	CMP $16, R10; \
	BLT scatter; \
	BCAST(FTHR, V20); \
	ACCGTM(V1, SP6, 20, V30); \
	ACCGTM(V1, SP5, 20, V30); \
	ACCGTM(V1, SP4, 20, V30); \
	ACCGTM(V2, SQ4, 20, V30); \
	ACCGTM(V2, SQ5, 20, V30); \
	ACCGTM(V2, SQ6, 20, V30); \
	VMOV V30.D[0], R9; \
	VMOV V30.D[1], R14; \
	AND  R14, R9, R9; \
	CMN  $1, R9; \
	BEQ  scatter; \
	LOAD(SP6, V16); \
	SUB  $208, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $192, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $176, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $160, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $144, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $128, R2, R9; \
	VST1 [V16.S4], (R9); \
	LOAD(SQ6, V16); \
	ADD  $112, R2, R9; \
	VST1 [V16.S4], (R9); \
	ADD  $128, R2, R9; \
	VST1 [V16.S4], (R9); \
	ADD  $144, R2, R9; \
	VST1 [V16.S4], (R9); \
	ADD  $160, R2, R9; \
	VST1 [V16.S4], (R9); \
	ADD  $176, R2, R9; \
	VST1 [V16.S4], (R9); \
	ADD  $192, R2, R9; \
	VST1 [V16.S4], (R9); \
	SUB  $192, R2, R8; \
	VLD1 (R8), [V17.S4]; \
	MOVD $12, R13; \
	sumwindow:; \
	ADD  $16, R8, R8; \
	VLD1 (R8), [V18.S4]; \
	VADD V18.S4, V17.S4, V17.S4; \
	SUBS $1, R13, R13; \
	BNE  sumwindow; \
	BCAST(C4, V29); \
	VADD V29.S4, V29.S4, V29.S4; \
	SUB  $96, R2, R8; \
	ADD  $224, R2, R14; \
	MOVD $12, R13; \
	fourteen:; \
	SUB  $16, R8, R9; \
	VLD1 (R9), [V19.S4]; \
	VLD1 (R8), [V20.S4]; \
	VADD V20.S4, V19.S4, V19.S4; \
	ADD  $16, R8, R9; \
	VLD1 (R9), [V20.S4]; \
	VADD V20.S4, V19.S4, V19.S4; \
	VADD V17.S4, V19.S4, V19.S4; \
	VADD V29.S4, V19.S4, V19.S4; \
	SSHR(19, 19, 4); \
	VLD1 (R14), [V21.S4]; \
	SEL(V30, V21, V19, V22); \
	VST1 [V22.S4], (R14); \
	ADD  $112, R8, R9; \
	VLD1 (R9), [V20.S4]; \
	VADD V20.S4, V17.S4, V17.S4; \
	SUB  $96, R8, R9; \
	VLD1 (R9), [V20.S4]; \
	VSUB V20.S4, V17.S4, V17.S4; \
	ADD  $16, R8, R8; \
	ADD  $16, R14, R14; \
	SUBS $1, R13, R13; \
	BNE  fourteen;

TEXT lpfbody8<>(SB), NOSPLIT|NOFRAME, $0
	LOADH(SP1, V0)
	LOADH(SP0, V1)
	LOADH(SQ0, V2)
	LOADH(SQ1, V3)
	BCASTH(I, V20)
	BCASTH(H, V21)
	VSUB V1.H8, V0.H8, V13.H8
	ABSH(13, 13)
	CMGTH(31, 13, 20)
	CMGTH(30, 13, 21)
	VSUB V2.H8, V3.H8, V13.H8
	ABSH(13, 13)
	CMGTH(14, 13, 20)
	VORR V14.B16, V31.B16, V31.B16
	CMGTH(14, 13, 21)
	VORR V14.B16, V30.B16, V30.B16
	ADD  $MASKH, R2, R9
	VST1 [V30.H8], (R9)
	VSUB  V2.H8, V1.H8, V13.H8
	ABSH(13, 13)
	VADD  V13.H8, V13.H8, V13.H8
	VSUB  V3.H8, V0.H8, V14.H8
	ABSH(14, 14)
	VUSHR $1, V14.H8, V14.H8
	VADD  V14.H8, V13.H8, V13.H8
	BCASTH(E, V22)
	CMGTH(14, 13, 22)
	VORR V14.B16, V31.B16, V31.B16
	MOVW WD, R10
	CMP  $4, R10
	BLE  fmdone
	ACCGTMH(V0, SP2, 20, V31)
	ACCGTMH(V3, SQ2, 20, V31)
	CMP  $6, R10
	BLE  fmdone
	LOADH(SP3, V15)
	ACCGTMH(V15, SP2, 20, V31)
	LOADH(SQ3, V15)
	ACCGTMH(V15, SQ2, 20, V31)

fmdone:
	VMOV V31.D[0], R9
	VMOV V31.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  bodydone

	ADD  $MASKH, R2, R9
	VLD1 (R9), [V30.H8]
	BCASTH(NEGLIM, V20)
	BCASTH(LIM1, V21)
	VSUB V3.H8, V0.H8, V13.H8
	CLIPDH(13, 20, 21)
	VAND V30.B16, V13.B16, V13.B16
	VSUB V1.H8, V2.H8, V14.H8
	VADD V14.H8, V14.H8, V15.H8
	VADD V14.H8, V15.H8, V15.H8
	VADD V15.H8, V13.H8, V13.H8
	CLIPDH(13, 20, 21)
	BCASTH(C4, V22)
	VADD V22.H8, V13.H8, V14.H8
	SMINH(14, 14, 21)
	SSHRH(14, 14, 3)
	BCASTH(C3, V22)
	VADD  V22.H8, V13.H8, V15.H8
	SMINH(15, 15, 21)
	SSHRH(15, 15, 3)
	VMOVI $0, V20.B16
	BCASTH(PIXMAX, V21)
	VADD V15.H8, V1.H8, V16.H8
	CLIPDH(16, 20, 21)
	VSUB V14.H8, V2.H8, V17.H8
	CLIPDH(17, 20, 21)
	BCASTH(C1, V22)
	VADD V22.H8, V14.H8, V18.H8
	SSHRH(18, 18, 1)
	VADD V18.H8, V0.H8, V13.H8
	CLIPDH(13, 20, 21)
	SEL(V30, V0, V13, V19)
	VSUB V18.H8, V3.H8, V13.H8
	CLIPDH(13, 20, 21)
	SEL(V30, V3, V13, V23)
	SEL(V31, V0, V19, V13)
	STOREH(OP1, V13)
	SEL(V31, V1, V16, V13)
	STOREH(OP0, V13)
	SEL(V31, V2, V17, V13)
	STOREH(OQ0, V13)
	SEL(V31, V3, V23, V13)
	STOREH(OQ1, V13)
	MOVW WD, R10
	CMP  $4, R10
	BLE  bodydone
	BCASTH(FTHR, V20)
	VMOV V31.B16, V30.B16
	ACCGTMH(V1, SP2, 20, V30)
	ACCGTH(V0, V1, 20, V30)
	ACCGTH(V3, V2, 20, V30)
	ACCGTMH(V2, SQ2, 20, V30)
	CMP  $8, R10
	BLT  flatindone
	ACCGTMH(V1, SP3, 20, V30)
	ACCGTMH(V2, SQ3, 20, V30)

flatindone:
	VMOV V30.D[0], R9
	VMOV V30.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  bodydone
	BCASTH(C4, V29)
	CMP  $6, R10
	BGT  eight
	WIDE6H(SP2, SP2, SP1, SP0, SQ0, OP1)
	WIDE6H(SP2, SP1, SP0, SQ0, SQ1, OP0)
	WIDE6H(SP1, SP0, SQ0, SQ1, SQ2, OQ0)
	WIDE6H(SP0, SQ0, SQ1, SQ2, SQ2, OQ1)
	B     bodydone

eight:
	WIDE8H(SP3, SP3, SP3, SP2, SP1, SP0, SQ0, OP2)
	WIDE8H(SP3, SP3, SP2, SP1, SP0, SQ0, SQ1, OP1)
	WIDE8H(SP3, SP2, SP1, SP0, SQ0, SQ1, SQ2, OP0)
	WIDE8H(SP2, SP1, SP0, SQ0, SQ1, SQ2, SQ3, OQ0)
	WIDE8H(SP1, SP0, SQ0, SQ1, SQ2, SQ3, SQ3, OQ1)
	WIDE8H(SP0, SQ0, SQ1, SQ2, SQ3, SQ3, SQ3, OQ2)
	CMP $16, R10
	BLT bodydone
	BCASTH(FTHR, V20)
	ACCGTMH(V1, SP6, 20, V30)
	ACCGTMH(V1, SP5, 20, V30)
	ACCGTMH(V1, SP4, 20, V30)
	ACCGTMH(V2, SQ4, 20, V30)
	ACCGTMH(V2, SQ5, 20, V30)
	ACCGTMH(V2, SQ6, 20, V30)
	VMOV V30.D[0], R9
	VMOV V30.D[1], R14
	AND  R14, R9, R9
	CMN  $1, R9
	BEQ  bodydone

	LOADH(SP6, V16)
	SUB  $208, R2, R9
	VST1 [V16.H8], (R9)
	SUB  $192, R2, R9
	VST1 [V16.H8], (R9)
	SUB  $176, R2, R9
	VST1 [V16.H8], (R9)
	SUB  $160, R2, R9
	VST1 [V16.H8], (R9)
	SUB  $144, R2, R9
	VST1 [V16.H8], (R9)
	SUB  $128, R2, R9
	VST1 [V16.H8], (R9)
	LOADH(SQ6, V16)
	ADD  $112, R2, R9
	VST1 [V16.H8], (R9)
	ADD  $128, R2, R9
	VST1 [V16.H8], (R9)
	ADD  $144, R2, R9
	VST1 [V16.H8], (R9)
	ADD  $160, R2, R9
	VST1 [V16.H8], (R9)
	ADD  $176, R2, R9
	VST1 [V16.H8], (R9)
	ADD  $192, R2, R9
	VST1 [V16.H8], (R9)

	SUB  $192, R2, R8
	VLD1 (R8), [V17.H8]
	MOVD $12, R13

sumwindow:
	ADD  $16, R8, R8
	VLD1 (R8), [V18.H8]
	VADD V18.H8, V17.H8, V17.H8
	SUBS $1, R13, R13
	BNE  sumwindow

	BCASTH(C4, V29)
	VADD V29.H8, V29.H8, V29.H8
	SUB  $96, R2, R8
	ADD  $224, R2, R14
	MOVD $12, R13

fourteen:
	SUB  $16, R8, R9
	VLD1 (R9), [V19.H8]
	VLD1 (R8), [V20.H8]
	VADD V20.H8, V19.H8, V19.H8
	ADD  $16, R8, R9
	VLD1 (R9), [V20.H8]
	VADD V20.H8, V19.H8, V19.H8
	VADD V17.H8, V19.H8, V19.H8
	VADD V29.H8, V19.H8, V19.H8
	SSHRH(19, 19, 4)
	VLD1 (R14), [V21.H8]
	SEL(V30, V21, V19, V22)
	VST1 [V22.H8], (R14)
	ADD  $112, R8, R9
	VLD1 (R9), [V20.H8]
	VADD V20.H8, V17.H8, V17.H8
	SUB  $96, R8, R9
	VLD1 (R9), [V20.H8]
	VSUB V20.H8, V17.H8, V17.H8
	ADD  $16, R8, R8
	ADD  $16, R14, R14
	SUBS $1, R13, R13
	BNE  fourteen

bodydone:
	RET

// func lpf8ColNEON(dst *uint8, stridea, strideb int, p *lpfParams, n int)
TEXT ·lpf8ColNEON(SB), NOSPLIT, $704-40
	MOVD dst+0(FP), R0
	MOVD strideb+16(FP), R4
	MOVD p+24(FP), R1
	MOVD n+32(FP), R5
	ADD  $224, RSP, R2

	MOVW WD, R10
	MOVD $-2, R11
	MOVD $4, R12
	CMP  $4, R10
	BLE  cspan
	MOVD $-3, R11
	MOVD $6, R12
	CMP  $6, R10
	BLE  cspan
	MOVD $-4, R11
	MOVD $8, R12
	CMP  $16, R10
	BLT  cspan
	MOVD $-7, R11
	MOVD $14, R12

cspan:
	MUL  R4, R11, R8
	ADD  R0, R8, R8
	ADD  R11<<4, R2, R9
	MOVD R12, R13
	CMP  $8, R5
	BEQ  cgather8

cgather4:
	MOVWU  (R8), R14
	ORR    R14<<32, R14, R14
	VMOV   R14, V16.D[0]
	VUSHLL $0, V16.B8, V17.H8
	VST1   [V17.H8], (R9)
	ADD    $320, R9, R14
	VST1   [V17.H8], (R14)
	ADD    R4, R8, R8
	ADD    $16, R9, R9
	SUBS   $1, R13, R13
	BNE    cgather4
	B      cbody

cgather8:
	VLD1   (R8), [V16.B8]
	VUSHLL $0, V16.B8, V17.H8
	VST1   [V17.H8], (R9)
	ADD    $320, R9, R14
	VST1   [V17.H8], (R14)
	ADD    R4, R8, R8
	ADD    $16, R9, R9
	SUBS   $1, R13, R13
	BNE    cgather8

cbody:
	BL lpfbody8<>(SB)

	MOVW WD, R10
	MOVD $-2, R11
	MOVD $4, R12
	CMP  $6, R10
	BLE  cout
	MOVD $-3, R11
	MOVD $6, R12
	CMP  $16, R10
	BLT  cout
	MOVD $-6, R11
	MOVD $12, R12

cout:
	MUL  R4, R11, R8
	ADD  R0, R8, R8
	ADD  R11<<4, R2, R9
	ADD  $320, R9, R9
	CMP  $8, R5
	BEQ  cscatter8

cscatter4:
	VLD1 (R9), [V16.H8]
	XTNH(17, 16)
	VMOV V17.S[0], R14
	MOVW R14, (R8)
	ADD  $16, R9, R9
	ADD  R4, R8, R8
	SUBS $1, R12, R12
	BNE  cscatter4
	RET

cscatter8:
	VLD1 (R9), [V16.H8]
	XTNH(17, 16)
	VST1 [V17.B8], (R8)
	ADD  $16, R9, R9
	ADD  R4, R8, R8
	SUBS $1, R12, R12
	BNE  cscatter8
	RET

// func lpf8RowNEON(dst *uint8, stridea, strideb int, p *lpfParams, back int)
TEXT ·lpf8RowNEON(SB), NOSPLIT, $704-40
	MOVD dst+0(FP), R0
	MOVD stridea+8(FP), R3
	MOVD p+24(FP), R1
	MOVD back+32(FP), R5
	ADD  $224, RSP, R2

	MOVW WD, R10
	MOVD $-4, R11
	MOVD $8, R12
	CMP  $16, R10
	BLT  rspan
	MOVD $-7, R11
	MOVD $16, R12

rspan:
	ADD R0, R11, R8
	ADD R11<<4, R2, R9

rgather:
	LOADROWS
	TRN8X8
	VUSHLL $0, V0.B8, V8.H8
	VUSHLL $0, V1.B8, V9.H8
	VUSHLL $0, V2.B8, V10.H8
	VUSHLL $0, V3.B8, V11.H8
	VUSHLL $0, V4.B8, V12.H8
	VUSHLL $0, V5.B8, V13.H8
	VUSHLL $0, V6.B8, V14.H8
	VUSHLL $0, V7.B8, V15.H8
	VST1   [V8.H8, V9.H8, V10.H8, V11.H8], (R9)
	ADD    $64, R9, R14
	VST1   [V12.H8, V13.H8, V14.H8, V15.H8], (R14)
	ADD    $320, R9, R14
	VST1   [V8.H8, V9.H8, V10.H8, V11.H8], (R14)
	ADD    $384, R9, R14
	VST1   [V12.H8, V13.H8, V14.H8, V15.H8], (R14)
	ADD    $8, R8, R8
	ADD    $128, R9, R9
	SUBS   $8, R12, R12
	BNE    rgather

	BL lpfbody8<>(SB)

	MOVW WD, R10
	MOVD $-4, R11
	MOVD $8, R12
	CMP  $16, R10
	BLT  rout
	MOVD $-7, R11
	MOVD $16, R12

rout:
	ADD R0, R11, R8
	ADD R11<<4, R2, R9
	ADD $320, R9, R9

rscatter:
	VLD1 (R9), [V8.H8, V9.H8, V10.H8, V11.H8]
	ADD  $64, R9, R14
	VLD1 (R14), [V12.H8, V13.H8, V14.H8, V15.H8]
	XTNH(0, 8)
	XTNH(1, 9)
	XTNH(2, 10)
	XTNH(3, 11)
	XTNH(4, 12)
	XTNH(5, 13)
	XTNH(6, 14)
	XTNH(7, 15)
	TRN8X8
	STOREROWS
	ADD  $8, R8, R8
	ADD  $128, R9, R9
	SUBS $8, R12, R12
	BNE  rscatter
	RET

// func lpf16NEON(dst *uint16, stridea, strideb int, p *lpfParams)
TEXT ·lpf16NEON(SB), 0, $848-32
	MOVD dst+0(FP), R0
	MOVD stridea+8(FP), R3
	MOVD strideb+16(FP), R4
	MOVD p+24(FP), R1
	ADD  $216, RSP, R2
	LSL  $1, R3, R7
	ADD  R3, R7, R7

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
	MOVD $16, R12

spanset:
	CMP $1, R3
	BEQ gathercols

	// The four lines are rows here, so read the span from each and transpose
	// it word-wise into tap order.
	CMP   $16, R10
	BGE   gatherwide
	MOVD  $-4, R11
	MOVD  $8, R12
	SUB   $8, R0, R8
	VLD1  (R8), [V0.H8]
	ADD   R3<<1, R8, R9
	VLD1  (R9), [V1.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V2.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V3.H8]
	VZIP1 V1.H8, V0.H8, V4.H8
	VZIP2 V1.H8, V0.H8, V5.H8
	VZIP1 V3.H8, V2.H8, V6.H8
	VZIP2 V3.H8, V2.H8, V7.H8
	VZIP1 V6.S4, V4.S4, V8.S4
	VZIP2 V6.S4, V4.S4, V9.S4
	VZIP1 V7.S4, V5.S4, V10.S4
	VZIP2 V7.S4, V5.S4, V11.S4
	ADD   $520, R2, R9
	VST1  [V8.B16, V9.B16, V10.B16, V11.B16], (R9)
	B     widen

gatherwide:
	MOVD  $-7, R11
	MOVD  $16, R12
	SUB   $14, R0, R8
	VLD1  (R8), [V0.H8]
	ADD   R3<<1, R8, R9
	VLD1  (R9), [V1.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V2.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V3.H8]
	VZIP1 V1.H8, V0.H8, V4.H8
	VZIP2 V1.H8, V0.H8, V5.H8
	VZIP1 V3.H8, V2.H8, V6.H8
	VZIP2 V3.H8, V2.H8, V7.H8
	VZIP1 V6.S4, V4.S4, V8.S4
	VZIP2 V6.S4, V4.S4, V9.S4
	VZIP1 V7.S4, V5.S4, V10.S4
	VZIP2 V7.S4, V5.S4, V11.S4
	ADD   $496, R2, R9
	VST1  [V8.B16, V9.B16, V10.B16, V11.B16], (R9)
	ADD   $2, R0, R8
	VLD1  (R8), [V0.H8]
	ADD   R3<<1, R8, R9
	VLD1  (R9), [V1.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V2.H8]
	ADD   R3<<1, R9, R9
	VLD1  (R9), [V3.H8]
	VZIP1 V1.H8, V0.H8, V4.H8
	VZIP2 V1.H8, V0.H8, V5.H8
	VZIP1 V3.H8, V2.H8, V6.H8
	VZIP2 V3.H8, V2.H8, V7.H8
	VZIP1 V6.S4, V4.S4, V8.S4
	VZIP2 V6.S4, V4.S4, V9.S4
	VZIP1 V7.S4, V5.S4, V10.S4
	VZIP2 V7.S4, V5.S4, V11.S4
	ADD   $560, R2, R9
	VST1  [V8.B16, V9.B16, V10.B16, V11.B16], (R9)
	B     widen

gathercols:
	MUL  R4, R11, R8
	ADD  R8<<1, R0, R8
	ADD  $552, R2, R9
	ADD  R11<<3, R9, R9
	MOVD R12, R13

gathercolsloop:
	MOVD (R8), R14
	MOVD R14, (R9)
	ADD  R4<<1, R8, R8
	ADD  $8, R9, R9
	SUBS $1, R13, R13
	BNE  gathercolsloop

widen:
	ADD  $552, R2, R8
	ADD  R11<<3, R8, R8
	ADD  R11<<4, R2, R9
	MOVD R12, R13

widenloop:
	VLD1   (R8), [V16.H4]
	VUSHLL $0, V16.H4, V18.S4
	VST1   [V18.S4], (R9)
	ADD    $320, R9, R14
	VST1   [V18.S4], (R14)
	ADD    $8, R8, R8
	ADD    $16, R9, R9
	SUBS   $1, R13, R13
	BNE    widenloop

	LPFBODY

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

	ADD  $552, R2, R8
	ADD  R11<<3, R8, R8
	MOVD $8, R15
	B    packloop

scattercols:
	MUL  R4, R11, R8
	ADD  R8<<1, R0, R8
	LSL  $1, R4, R15

packloop:
	ADD   R11<<4, R2, R9
	ADD   $320, R9, R9
	VLD1  (R9), [V16.S4]
	XTNS(17, 16)
	VST1  [V17.D1], (R8)
	ADD   $1, R11, R11
	ADD   R15, R8, R8
	SUBS  $1, R12, R12
	BNE   packloop

	CMP $1, R3
	BEQ done

	MOVW  WD, R10
	CMP   $16, R10
	BGE   scatterwide
	ADD   $520, R2, R9
	VLD1  (R9), [V8.B16, V9.B16, V10.B16, V11.B16]
	VUZP1 V9.S4, V8.S4, V4.S4
	VUZP2 V9.S4, V8.S4, V5.S4
	VUZP1 V11.S4, V10.S4, V6.S4
	VUZP2 V11.S4, V10.S4, V7.S4
	VUZP1 V6.H8, V4.H8, V0.H8
	VUZP2 V6.H8, V4.H8, V1.H8
	VUZP1 V7.H8, V5.H8, V2.H8
	VUZP2 V7.H8, V5.H8, V3.H8
	SUB   $8, R0, R8
	VST1  [V0.H8], (R8)
	ADD   R3<<1, R8, R9
	VST1  [V1.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V2.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V3.H8], (R9)
	B     done

scatterwide:
	ADD   $496, R2, R9
	VLD1  (R9), [V8.B16, V9.B16, V10.B16, V11.B16]
	VUZP1 V9.S4, V8.S4, V4.S4
	VUZP2 V9.S4, V8.S4, V5.S4
	VUZP1 V11.S4, V10.S4, V6.S4
	VUZP2 V11.S4, V10.S4, V7.S4
	VUZP1 V6.H8, V4.H8, V0.H8
	VUZP2 V6.H8, V4.H8, V1.H8
	VUZP1 V7.H8, V5.H8, V2.H8
	VUZP2 V7.H8, V5.H8, V3.H8
	SUB   $14, R0, R8
	VST1  [V0.H8], (R8)
	ADD   R3<<1, R8, R9
	VST1  [V1.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V2.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V3.H8], (R9)
	ADD   $560, R2, R9
	VLD1  (R9), [V8.B16, V9.B16, V10.B16, V11.B16]
	VUZP1 V9.S4, V8.S4, V4.S4
	VUZP2 V9.S4, V8.S4, V5.S4
	VUZP1 V11.S4, V10.S4, V6.S4
	VUZP2 V11.S4, V10.S4, V7.S4
	VUZP1 V6.H8, V4.H8, V0.H8
	VUZP2 V6.H8, V4.H8, V1.H8
	VUZP1 V7.H8, V5.H8, V2.H8
	VUZP2 V7.H8, V5.H8, V3.H8
	ADD   $2, R0, R8
	VST1  [V0.H8], (R8)
	ADD   R3<<1, R8, R9
	VST1  [V1.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V2.H8], (R9)
	ADD   R3<<1, R9, R9
	VST1  [V3.H8], (R9)

done:
	RET
