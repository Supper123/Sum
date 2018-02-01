package encoding

import (
	"fmt"
	"io"
)

//读取带有前缀的数据
func ReadPrefix(r io.Reader, maxLen uint64) ([]byte, error) {
	prefix := make([]byte, 8)

	if _, err := io.ReadFull(r, prefix); err != nil {
		return nil, err
	}
	dataLen := DecUint64(prefix)
	if dataLen > maxLen {
		return nil, fmt.Errorf("length %d exceeds maxLen of %d", dataLen, maxLen)
	}

	data := make([]byte, dataLen)
	_, err := io.ReadFull(r, data)
	return data, err
}

//读取并解码对象数据
func ReadObject(r io.Reader, obj interface{}, maxLen uint64) error {
	data, err := ReadPrefix(r, maxLen)
	if err != nil {
		return err
	}
	return Unmarshal(data, obj)
}

//写入带前缀的数据
func WritePrefix(w io.Writer, data []byte) error {
	err := WriteInt(w, len(data))
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

//编码并写入对象数据
func WriteObject(w io.Writer, v interface{}) error {
	return WritePrefix(w, Marshal(v))
}
