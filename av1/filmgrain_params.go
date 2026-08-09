package av1

// fguv16Params carries what the chroma row needs beyond its pointers.
type fguv16Params struct {
	n, shift, minValue, maxValue int
	csfl                         int
	lumaMult, mult, offset       int
	pixelMax                     int
	sx                           int
}
