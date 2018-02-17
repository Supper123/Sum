package merkletree

import (
	"errors"
	"hash"
	"io"
)

//读取segmentSize大小的片段，然后将它压入树中
func (t *Tree) ReadAll(r io.Reader, segmentSize int) error {
	for {
		segment := make([]byte, segmentSize)
		n, readErr := io.ReadFull(r, segment)
		if readErr == io.EOF {
			break
		} else if readErr == io.ErrUnexpectedEOF {
			segment = segment[:n]
		} else if readErr != nil {
			return readErr
		}
		t.Push(segment)
	}
	return nil
}

//返回merkle树根的数据
func ReaderRoot(r io.Reader, h hash.Hash, segmentSize int) (root []byte, err error) {
	tree := New(h)
	err = tree.ReadAll(r, segmentSize)
	if err != nil {
		return
	}
	root = tree.Root()
	return
}

//证明确定的数据是merkle树中的数据
func BuildReaderProof(r io.Reader, h hash.Hash, segmentSize int, index uint64) (root []byte, proofSet [][]byte, numLeaves uint64, err error) {
	tree := New(h)
	err = tree.SetIndex(index)
	if err != nil {
		panic(err)
	}
	err = tree.ReadAll(r, segmentSize)
	if err != nil {
		return
	}
	root, proofSet, _, numLeaves = tree.Prove()
	if len(proofSet) == 0 {
		err = errors.New("index was not reached while creating proof")
		return
	}
	return
}
