package av1

type edgeNode struct {
	o     uint8
	h, v  [2]uint8
	split [3]uint8
	h4    uint8
	v4    uint8
	kids  [4]*edgeNode
}

var intraEdgeTree [2]*edgeNode

func initEdges(node *edgeNode, bl int, edgeFlags uint8) {
	node.o = edgeFlags
	node.h[0] = edgeFlags | edgeAllLeftHasBottom
	node.v[0] = edgeFlags | edgeAllTopHasRight

	if bl == bl8x8 {
		node.h[1] = edgeFlags & (edgeAllLeftHasBottom | edgeI420TopHasRight)
		node.v[1] = edgeFlags & (edgeAllTopHasRight | edgeI420LeftHasBottom |
			edgeI422LeftHasBottom)

		node.split[0] = edgeFlags&edgeAllTopHasRight | edgeI422LeftHasBottom
		node.split[1] = edgeFlags | edgeI444TopHasRight
		node.split[2] = edgeFlags & (edgeI420TopHasRight | edgeI420LeftHasBottom |
			edgeI422LeftHasBottom)

		return
	}

	node.h[1] = edgeFlags & edgeAllLeftHasBottom
	node.v[1] = edgeFlags & edgeAllTopHasRight

	node.h4 = edgeAllLeftHasBottom
	node.v4 = edgeAllTopHasRight
	if bl == bl16x16 {
		node.h4 |= edgeFlags & edgeI420TopHasRight
		node.v4 |= edgeFlags & (edgeI420LeftHasBottom | edgeI422LeftHasBottom)
	}
}

func initModeNode(node *edgeNode, bl int, topHasRight, leftHasBottom bool) {
	var flags uint8
	if topHasRight {
		flags |= edgeAllTopHasRight
	}
	if leftHasBottom {
		flags |= edgeAllLeftHasBottom
	}
	initEdges(node, bl, flags)

	if bl == bl16x16 {
		for n := range 4 {
			kid := new(edgeNode)
			node.kids[n] = kid
			var f uint8
			if !(n == 3 || (n == 1 && !topHasRight)) {
				f |= edgeAllTopHasRight
			}
			if n == 0 || (n == 2 && leftHasBottom) {
				f |= edgeAllLeftHasBottom
			}
			initEdges(kid, bl+1, f)
		}

		return
	}

	for n := range 4 {
		kid := new(edgeNode)
		node.kids[n] = kid
		initModeNode(kid, bl+1,
			!(n == 3 || (n == 1 && !topHasRight)),
			n == 0 || (n == 2 && leftHasBottom))
	}
}

func init() {
	sb128 := new(edgeNode)
	initModeNode(sb128, bl128x128, true, false)
	sb64 := new(edgeNode)
	initModeNode(sb64, bl64x64, true, false)

	intraEdgeTree[0] = sb128
	intraEdgeTree[1] = sb64
}
