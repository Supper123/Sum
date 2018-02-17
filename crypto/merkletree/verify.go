package merkletree

import (
	"bytes"
	"hash"
)

//如果proofSet第一个元素是叶子节点的话返回真
func VerifyProof(h hash.Hash, merkleRoot []byte, proofSet [][]byte, proofIndex uint64, numLeaves uint64) bool {
	if merkleRoot == nil {
		return false
	}
	if proofIndex >= numLeaves {
		return false
	}

	height := 0
	if len(proofSet) <= height {
		return false
	}
	sum := leafSum(h, proofSet[height])
	height++

	stableEnd := proofIndex
	for {
		subTreeStartIndex := (proofIndex / (1 << uint(height))) * (1 << uint(height))
		subTreeEndIndex := subTreeStartIndex + (1 << (uint(height))) - 1
		if subTreeEndIndex >= numLeaves {
			break
		}
		stableEnd = subTreeEndIndex

		if len(proofSet) <= height {
			return false
		}
		if proofIndex-subTreeStartIndex < 1<<uint(height-1) {
			sum = nodeSum(h, sum, proofSet[height])
		} else {
			sum = nodeSum(h, proofSet[height], sum)
		}
		height++
	}

	if stableEnd != numLeaves-1 {
		if len(proofSet) <= height {
			return false
		}
		sum = nodeSum(h, sum, proofSet[height])
		height++
	}

	for height < len(proofSet) {
		sum = nodeSum(h, proofSet[height], sum)
		height++
	}

	if bytes.Compare(sum, merkleRoot) == 0 {
		return true
	}
	return false
}
