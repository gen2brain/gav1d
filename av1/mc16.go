package av1

// mcTaps widens the filter, since every tap is a 32-bit multiply here.
func mcTaps(f *[8]int8) [8]int32 {
	var t [8]int32
	for i, v := range f {
		t[i] = int32(v)
	}

	return t
}
