package av1

// lpfParams carries the thresholds already scaled to the bit depth, plus the
// small constants the kernel would otherwise have to build.
type lpfParams struct {
	e, i, h, fThr int32
	limM1, negLim int32
	pixMax        int32
	c4, c3, c1    int32
	wd            int32
}

func lpfParamsFor(e, i, h int32, wd int, bitdepthMax int32) lpfParams {
	bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
	lim := int32(128) << bitdepthMin8

	return lpfParams{
		e:      e << bitdepthMin8,
		i:      i << bitdepthMin8,
		h:      h << bitdepthMin8,
		fThr:   int32(1) << bitdepthMin8,
		limM1:  lim - 1,
		negLim: -lim,
		pixMax: bitdepthMax,
		c4:     4,
		c3:     3,
		c1:     1,
		wd:     int32(wd),
	}
}
