//go:build (!amd64 && !arm64) || noasm

package av1

func dspInit[P pixel](*dspContext[P]) {}
