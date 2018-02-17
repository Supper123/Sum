package merkletree

import (
	"errors"
	"hash"
)

type CachedTree struct {
	cachedNodeHeight uint64
	trueProofIndex   uint64
	Tree
}

//新建缓存树
func NewCachedTree(h hash.Hash, cachedNodeHeight uint64) *CachedTree {
	return &CachedTree{
		cachedNodeHeight: cachedNodeHeight,
		Tree: Tree{
			hash:       h,
			cachedTree: true,
		},
	}
}

//证明建立索引的叶子节点是树中的元素
func (ct *CachedTree) Prove(cachedProofSet [][]byte) (merkleRoot []byte, proofSet [][]byte, proofIndex uint64, numLeaves uint64) {
	leavesPerCachedNode := uint64(1) << ct.cachedNodeHeight
	numLeaves = leavesPerCachedNode * ct.currentIndex

	merkleRoot, proofSetTail, _, _ := ct.Tree.Prove()
	if len(proofSetTail) < 1 {
		return merkleRoot, nil, ct.trueProofIndex, numLeaves
	}
	proofSet = append(cachedProofSet, proofSetTail[1:]...)
	return merkleRoot, proofSet, ct.trueProofIndex, numLeaves
}

//设置树的索引，新建树时调用
func (ct *CachedTree) SetIndex(i uint64) error {
	if ct.head != nil {
		return errors.New("cannot call SetIndex on Tree if Tree has not been reset")
	}
	ct.trueProofIndex = i
	return ct.Tree.SetIndex(i / (1 << ct.cachedNodeHeight))
}
