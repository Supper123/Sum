package merkletree

import (
	"errors"
	"hash"
)

type Tree struct {
	head *subTree
	hash hash.Hash

	currentIndex uint64
	proofIndex   uint64
	proofSet     [][]byte

	cachedTree bool
}

type subTree struct {
	next   *subTree
	height int
	sum    []byte
}

//求和 data
func sum(h hash.Hash, data ...[]byte) []byte {
	h.Reset()
	for _, d := range data {
		_, _ = h.Write(d)
	}
	return h.Sum(nil)
}

//计算叶子节点的hash， Hash(0x00 || data)
func leafSum(h hash.Hash, data []byte) []byte {
	return sum(h, []byte{0}, data)
}

//计算两个子叶的父hash， Hash(0x01 || left sibling sum || right sibling sum)
func nodeSum(h hash.Hash, a, b []byte) []byte {
	return sum(h, []byte{1}, a, b)
}

//将两个叶子节点合并为一个大子树
func joinSubTrees(h hash.Hash, a, b *subTree) *subTree {
	return &subTree{
		next:   a.next,
		height: a.height + 1,
		sum:    nodeSum(h, a.sum, b.sum),
	}
}

//新建一个树，树根的hash为h
func New(h hash.Hash) *Tree {
	return &Tree{
		hash: h,
	}
}

//证明建立索引的叶子节点是树中的元素
func (t *Tree) Prove() (merkleRoot []byte, proofSet [][]byte, proofIndex uint64, numLeaves uint64) {
	//如何头指针为空，元素集合返回为nil
	if t.head == nil || len(t.proofSet) == 0 {
		return t.Root(), nil, t.proofIndex, t.currentIndex
	}
	proofSet = t.proofSet

	current := t.head
	for current.next != nil && current.next.height < len(proofSet)-1 {
		current = joinSubTrees(t.hash, current.next, current)
	}

	if current.next != nil && current.next.height == len(proofSet)-1 {
		proofSet = append(proofSet, current.sum)
		current = current.next
	}

	current = current.next

	for current != nil {
		proofSet = append(proofSet, current.sum)
		current = current.next
	}
	return t.Root(), proofSet, t.proofIndex, t.currentIndex
}

//添加data到集合中
func (t *Tree) Push(data []byte) {
	if t.currentIndex == t.proofIndex {
		t.proofSet = append(t.proofSet, data)
	}

	t.head = &subTree{
		next:   t.head,
		height: 0,
	}
	if t.cachedTree {
		t.head.sum = data
	} else {
		t.head.sum = leafSum(t.hash, data)
	}

	for t.head.next != nil && t.head.height == t.head.next.height {
		if t.head.height == len(t.proofSet)-1 {
			leaves := uint64(1 << uint(t.head.height))
			mid := (t.currentIndex / leaves) * leaves
			if t.proofIndex < mid {
				t.proofSet = append(t.proofSet, t.head.sum)
			} else {
				t.proofSet = append(t.proofSet, t.head.next.sum)
			}
		}
		t.head = joinSubTrees(t.hash, t.head.next, t.head)
	}
	t.currentIndex++
}

//返回树根
func (t *Tree) Root() []byte {
	if t.head == nil {
		return nil
	}

	current := t.head
	for current.next != nil {
		current = joinSubTrees(t.hash, current.next, current)
	}
	return current.sum
}

//设置树的索引，新建树时调用
func (t *Tree) SetIndex(i uint64) error {
	if t.head != nil {
		return errors.New("cannot call SetIndex on Tree if Tree has not been reset")
	}
	t.proofIndex = i
	return nil
}
