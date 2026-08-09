//go:build arm64 && !noasm

package avif

func rowFn() convertRow { return convertRowGo }
