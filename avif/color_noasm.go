//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package avif

func rowFn() convertRow { return convertRowGo }

func rowFn16() convertRow16 { return convertRow16Go }
