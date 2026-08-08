package av1

import (
	"io"
	"strconv"
)

type hdrDumper struct {
	w      io.Writer
	prefix string
	buf    []byte
}

func (d *hdrDumper) v(name string, val int64) {
	d.buf = d.buf[:0]
	d.buf = append(d.buf, d.prefix...)
	d.buf = append(d.buf, name...)
	d.buf = append(d.buf, ' ')
	d.buf = strconv.AppendInt(d.buf, val, 10)
	d.buf = append(d.buf, '\n')
	d.w.Write(d.buf)
}

func (d *hdrDumper) i(name string, idx int, val int64) {
	d.v(name+"["+strconv.Itoa(idx)+"]", val)
}

func (d *hdrDumper) i2(name string, a, b int, val int64) {
	d.v(name+"["+strconv.Itoa(a)+"]["+strconv.Itoa(b)+"]", val)
}

func dumpSeqHdr(w io.Writer, s *sequenceHeader) {
	d := &hdrDumper{w: w, prefix: "SEQ "}

	d.v("profile", int64(s.profile))
	d.v("max_width", int64(s.maxWidth))
	d.v("max_height", int64(s.maxHeight))
	d.v("layout", int64(s.layout))
	d.v("pri", int64(s.pri))
	d.v("trc", int64(s.trc))
	d.v("mtrx", int64(s.mtrx))
	d.v("chr", int64(s.chr))
	d.v("hbd", int64(s.hbd))
	d.v("color_range", int64(s.colorRange))
	d.v("num_operating_points", int64(s.numOperatingPoints))
	d.v("still_picture", int64(s.stillPicture))
	d.v("reduced_still_picture_header", int64(s.reducedStillPictureHeader))
	d.v("timing_info_present", int64(s.timingInfoPresent))
	d.v("num_units_in_tick", int64(s.numUnitsInTick))
	d.v("time_scale", int64(s.timeScale))
	d.v("equal_picture_interval", int64(s.equalPictureInterval))
	d.v("num_ticks_per_picture", int64(s.numTicksPerPicture))
	d.v("decoder_model_info_present", int64(s.decoderModelInfoPresent))
	d.v("encoder_decoder_buffer_delay_length", int64(s.encoderDecoderBufferDelayLength))
	d.v("num_units_in_decoding_tick", int64(s.numUnitsInDecodingTick))
	d.v("buffer_removal_delay_length", int64(s.bufferRemovalDelayLength))
	d.v("frame_presentation_delay_length", int64(s.framePresentationDelayLength))
	d.v("display_model_info_present", int64(s.displayModelInfoPresent))
	d.v("width_n_bits", int64(s.widthNBits))
	d.v("height_n_bits", int64(s.heightNBits))
	d.v("frame_id_numbers_present", int64(s.frameIDNumbersPresent))
	d.v("delta_frame_id_n_bits", int64(s.deltaFrameIDNBits))
	d.v("frame_id_n_bits", int64(s.frameIDNBits))
	d.v("sb128", int64(s.sb128))
	d.v("filter_intra", int64(s.filterIntra))
	d.v("intra_edge_filter", int64(s.intraEdgeFilter))
	d.v("inter_intra", int64(s.interIntra))
	d.v("masked_compound", int64(s.maskedCompound))
	d.v("warped_motion", int64(s.warpedMotion))
	d.v("dual_filter", int64(s.dualFilter))
	d.v("order_hint", int64(s.orderHint))
	d.v("jnt_comp", int64(s.jntComp))
	d.v("ref_frame_mvs", int64(s.refFrameMvs))
	d.v("screen_content_tools", int64(s.screenContentTools))
	d.v("force_integer_mv", int64(s.forceIntegerMv))
	d.v("order_hint_n_bits", int64(s.orderHintNBits))
	d.v("super_res", int64(s.superRes))
	d.v("cdef", int64(s.cdef))
	d.v("restoration", int64(s.restoration))
	d.v("ss_hor", int64(s.ssHor))
	d.v("ss_ver", int64(s.ssVer))
	d.v("monochrome", int64(s.monochrome))
	d.v("color_description_present", int64(s.colorDescriptionPresent))
	d.v("separate_uv_delta_q", int64(s.separateUvDeltaQ))
	d.v("film_grain_present", int64(s.filmGrainPresent))

	for i := range int(s.numOperatingPoints) {
		op := &s.operatingPoints[i]
		d.v("op["+strconv.Itoa(i)+"].major_level", int64(op.majorLevel))
		d.v("op["+strconv.Itoa(i)+"].minor_level", int64(op.minorLevel))
		d.v("op["+strconv.Itoa(i)+"].initial_display_delay", int64(op.initialDisplayDelay))
		d.v("op["+strconv.Itoa(i)+"].idc", int64(op.idc))
		d.v("op["+strconv.Itoa(i)+"].tier", int64(op.tier))
		d.v("op["+strconv.Itoa(i)+"].decoder_model_param_present",
			int64(op.decoderModelParamPresent))
		d.v("op["+strconv.Itoa(i)+"].display_model_param_present",
			int64(op.displayModelParamPresent))
		opi := &s.operatingParameterInfo[i]
		d.v("opi["+strconv.Itoa(i)+"].decoder_buffer_delay", int64(opi.decoderBufferDelay))
		d.v("opi["+strconv.Itoa(i)+"].encoder_buffer_delay", int64(opi.encoderBufferDelay))
		d.v("opi["+strconv.Itoa(i)+"].low_delay_mode", int64(opi.lowDelayMode))
	}
}

func dumpFrameHdr(w io.Writer, h *frameHeader) {
	d := &hdrDumper{w: w, prefix: "FRM "}

	d.v("frame_type", int64(h.frameType))
	for i := range 2 {
		d.i("width", i, int64(h.width[i]))
	}
	d.v("height", int64(h.height))
	d.v("frame_offset", int64(h.frameOffset))
	d.v("temporal_id", int64(h.temporalID))
	d.v("spatial_id", int64(h.spatialID))
	d.v("show_existing_frame", int64(h.showExistingFrame))
	d.v("existing_frame_idx", int64(h.existingFrameIdx))
	d.v("frame_id", int64(h.frameID))
	d.v("frame_presentation_delay", int64(h.framePresentationDelay))
	d.v("show_frame", int64(h.showFrame))
	d.v("showable_frame", int64(h.showableFrame))
	d.v("error_resilient_mode", int64(h.errorResilientMode))
	d.v("disable_cdf_update", int64(h.disableCdfUpdate))
	d.v("allow_screen_content_tools", int64(h.allowScreenContentTools))
	d.v("force_integer_mv", int64(h.forceIntegerMv))
	d.v("frame_size_override", int64(h.frameSizeOverride))
	d.v("primary_ref_frame", int64(h.primaryRefFrame))
	d.v("buffer_removal_time_present", int64(h.bufferRemovalTimePresent))
	d.v("refresh_frame_flags", int64(h.refreshFrameFlags))
	d.v("render_width", int64(h.renderWidth))
	d.v("render_height", int64(h.renderHeight))
	d.v("super_res.width_scale_denominator", int64(h.superRes.widthScaleDenominator))
	d.v("super_res.enabled", int64(h.superRes.enabled))
	d.v("have_render_size", int64(h.haveRenderSize))
	d.v("allow_intrabc", int64(h.allowIntrabc))
	d.v("frame_ref_short_signaling", int64(h.frameRefShortSignaling))
	for i := range refsPerFrame {
		d.i("refidx", i, int64(h.refidx[i]))
	}
	d.v("hp", int64(h.hp))
	d.v("subpel_filter_mode", int64(h.subpelFilterMode))
	d.v("switchable_motion_mode", int64(h.switchableMotionMode))
	d.v("use_ref_frame_mvs", int64(h.useRefFrameMvs))
	d.v("refresh_context", int64(h.refreshContext))

	t := &h.tiling
	d.v("tiling.uniform", int64(t.uniform))
	d.v("tiling.n_bytes", int64(t.nBytes))
	d.v("tiling.min_log2_cols", int64(t.minLog2Cols))
	d.v("tiling.max_log2_cols", int64(t.maxLog2Cols))
	d.v("tiling.log2_cols", int64(t.log2Cols))
	d.v("tiling.cols", int64(t.cols))
	d.v("tiling.min_log2_rows", int64(t.minLog2Rows))
	d.v("tiling.max_log2_rows", int64(t.maxLog2Rows))
	d.v("tiling.log2_rows", int64(t.log2Rows))
	d.v("tiling.rows", int64(t.rows))
	d.v("tiling.update", int64(t.update))
	for i := range int(t.cols) + 1 {
		d.i("tiling.col_start_sb", i, int64(t.colStartSb[i]))
	}
	for i := range int(t.rows) + 1 {
		d.i("tiling.row_start_sb", i, int64(t.rowStartSb[i]))
	}

	q := &h.quant
	d.v("quant.yac", int64(q.yac))
	d.v("quant.ydc_delta", int64(q.ydcDelta))
	d.v("quant.udc_delta", int64(q.udcDelta))
	d.v("quant.uac_delta", int64(q.uacDelta))
	d.v("quant.vdc_delta", int64(q.vdcDelta))
	d.v("quant.vac_delta", int64(q.vacDelta))
	d.v("quant.qm", int64(q.qm))
	d.v("quant.qm_y", int64(q.qmY))
	d.v("quant.qm_u", int64(q.qmU))
	d.v("quant.qm_v", int64(q.qmV))

	sg := &h.segmentation
	d.v("segmentation.enabled", int64(sg.enabled))
	d.v("segmentation.update_map", int64(sg.updateMap))
	d.v("segmentation.temporal", int64(sg.temporal))
	d.v("segmentation.update_data", int64(sg.updateData))
	d.v("segmentation.seg_data.preskip", int64(sg.segData.preskip))
	d.v("segmentation.seg_data.last_active_segid", int64(sg.segData.lastActiveSegid))
	for i := range maxSegments {
		sd := &sg.segData.d[i]
		n := strconv.Itoa(i)
		d.v("seg["+n+"].delta_q", int64(sd.deltaQ))
		d.v("seg["+n+"].delta_lf_y_v", int64(sd.deltaLfYV))
		d.v("seg["+n+"].delta_lf_y_h", int64(sd.deltaLfYH))
		d.v("seg["+n+"].delta_lf_u", int64(sd.deltaLfU))
		d.v("seg["+n+"].delta_lf_v", int64(sd.deltaLfV))
		d.v("seg["+n+"].ref", int64(sd.ref))
		d.v("seg["+n+"].skip", int64(sd.skip))
		d.v("seg["+n+"].globalmv", int64(sd.globalmv))
	}
	for i := range maxSegments {
		d.i("segmentation.lossless", i, int64(sg.lossless[i]))
	}
	for i := range maxSegments {
		d.i("segmentation.qidx", i, int64(sg.qidx[i]))
	}

	d.v("delta.q.present", int64(h.delta.q.present))
	d.v("delta.q.res_log2", int64(h.delta.q.resLog2))
	d.v("delta.lf.present", int64(h.delta.lf.present))
	d.v("delta.lf.res_log2", int64(h.delta.lf.resLog2))
	d.v("delta.lf.multi", int64(h.delta.lf.multi))
	d.v("all_lossless", int64(h.allLossless))

	lf := &h.loopfilter
	for i := range 2 {
		d.i("loopfilter.level_y", i, int64(lf.levelY[i]))
	}
	d.v("loopfilter.level_u", int64(lf.levelU))
	d.v("loopfilter.level_v", int64(lf.levelV))
	d.v("loopfilter.mode_ref_delta_enabled", int64(lf.modeRefDeltaEnabled))
	d.v("loopfilter.mode_ref_delta_update", int64(lf.modeRefDeltaUpdate))
	d.v("loopfilter.sharpness", int64(lf.sharpness))
	for i := range 2 {
		d.i("loopfilter.mode_ref_deltas.mode_delta", i, int64(lf.modeRefDeltas.modeDelta[i]))
	}
	for i := range totalRefsPerFrame {
		d.i("loopfilter.mode_ref_deltas.ref_delta", i, int64(lf.modeRefDeltas.refDelta[i]))
	}

	d.v("cdef.damping", int64(h.cdef.damping))
	d.v("cdef.n_bits", int64(h.cdef.nBits))
	for i := range maxCdefStrengths {
		d.i("cdef.y_strength", i, int64(h.cdef.yStrength[i]))
	}
	for i := range maxCdefStrengths {
		d.i("cdef.uv_strength", i, int64(h.cdef.uvStrength[i]))
	}

	for i := range 3 {
		d.i("restoration.type", i, int64(h.restoration.typ[i]))
	}
	for i := range 2 {
		d.i("restoration.unit_size", i, int64(h.restoration.unitSize[i]))
	}

	d.v("txfm_mode", int64(h.txfmMode))
	d.v("switchable_comp_refs", int64(h.switchableCompRefs))
	d.v("skip_mode_allowed", int64(h.skipModeAllowed))
	d.v("skip_mode_enabled", int64(h.skipModeEnabled))
	for i := range 2 {
		d.i("skip_mode_refs", i, int64(h.skipModeRefs[i]))
	}
	d.v("warp_motion", int64(h.warpMotion))
	d.v("reduced_txtp_set", int64(h.reducedTxtpSet))

	for i := range refsPerFrame {
		n := strconv.Itoa(i)
		d.v("gmv["+n+"].type", int64(h.gmv[i].typ))
		for j := range 6 {
			d.v("gmv["+n+"].matrix["+strconv.Itoa(j)+"]", int64(h.gmv[i].matrix[j]))
		}
	}

	d.v("film_grain.present", int64(h.filmGrain.present))
	d.v("film_grain.update", int64(h.filmGrain.update))
	g := &h.filmGrain.data
	d.v("fg.seed", int64(g.seed))
	d.v("fg.num_y_points", int64(g.numYPoints))
	for i := range g.numYPoints {
		d.i2("fg.y_points", i, 0, int64(g.yPoints[i][0]))
		d.i2("fg.y_points", i, 1, int64(g.yPoints[i][1]))
	}
	d.v("fg.chroma_scaling_from_luma", int64(g.chromaScalingFromLuma))
	for pl := range 2 {
		d.i("fg.num_uv_points", pl, int64(g.numUvPoints[pl]))
		for i := range g.numUvPoints[pl] {
			d.v("fg.uv_points["+strconv.Itoa(pl)+"]["+strconv.Itoa(i)+"][0]",
				int64(g.uvPoints[pl][i][0]))
			d.v("fg.uv_points["+strconv.Itoa(pl)+"]["+strconv.Itoa(i)+"][1]",
				int64(g.uvPoints[pl][i][1]))
		}
	}
	d.v("fg.scaling_shift", int64(g.scalingShift))
	d.v("fg.ar_coeff_lag", int64(g.arCoeffLag))
	for i := range 24 {
		d.i("fg.ar_coeffs_y", i, int64(g.arCoeffsY[i]))
	}
	for pl := range 2 {
		for i := range 25 {
			d.i2("fg.ar_coeffs_uv", pl, i, int64(g.arCoeffsUv[pl][i]))
		}
	}
	d.v("fg.ar_coeff_shift", int64(g.arCoeffShift))
	d.v("fg.grain_scale_shift", int64(g.grainScaleShift))
	for pl := range 2 {
		d.i("fg.uv_mult", pl, int64(g.uvMult[pl]))
		d.i("fg.uv_luma_mult", pl, int64(g.uvLumaMult[pl]))
		d.i("fg.uv_offset", pl, int64(g.uvOffset[pl]))
	}
	d.v("fg.overlap_flag", int64(g.overlapFlag))
	d.v("fg.clip_to_restricted_range", int64(g.clipToRestrictedRange))
}
