package encoding

import (
	"encoding/binary"
	"io"
)

//对整型进行小端序编码
//编码int64
func EncInt64(i int64) (b []byte) {
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	return
}

//解码int64
func DecInt64(b []byte) int64 {
	b2 := make([]byte, 8)
	copy(b2, b)
	return int64(binary.LittleEndian.Uint64(b2))
}

//编码uint64
func EncUint64(u uint64) (b []byte) {
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, u)
	return
}

//解码uint64
func DecUint64(b []byte) uint64 {
	b2 := make([]byte, 8)
	copy(b2, b)
	return binary.LittleEndian.Uint64(b2)
}

//将Uint64编码后写入io
func WriteUint64(w io.Writer, u uint64) error {
	_, err := w.Write(EncUint64(u))
	return err
}

//将int64编码后写入io
func WriteInt(w io.Writer, i int) error {
	return WriteUint64(w, uint64(i))
}
