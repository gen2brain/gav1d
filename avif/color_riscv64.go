//go:build riscv64 && riscv64.rva23u64 && !noasm

package avif

func rowFn() convertRow { return convertRowGo }
