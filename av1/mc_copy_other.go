//go:build !amd64 || noasm

package av1

func mcPutFast[P pixel](_ []P, _, _ int, _ []P, _, _, _, _ int) bool {
	return false
}
