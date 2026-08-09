//go:build amd64 && !noasm

package avif

var hasAVX2 = cpuidAVX2()

func cpuidAVX2() bool
