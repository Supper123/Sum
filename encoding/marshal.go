package encoding

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
)

const (
	MaxObjectSize = 12e6 //12MB
	MaxSliceSize  = 5e6  //5MB
)

var (
	errBadPointer     = errors.New("cannot decode into invalid pointer")
	ErrObjectTooLarge = errors.New("encoded object exceeds size limit")
	ErrSliceTooLarge  = errors.New("encoded slice is too large")
)

type (
	SUMMarshaler interface {
		MarshalSUM(io.Writer) error
	}

	SUMUnmarshaler interface {
		UnmarshalSUM(io.Reader) error
	}

	Encoder struct {
		w io.Writer
	}
)

//编码接口
func (e *Encoder) Encode(v interface{}) error {
	return e.encode(reflect.ValueOf(v))
}

//批量编码接口
func (e *Encoder) EncodeAll(vs ...interface{}) error {
	for _, v := range vs {
		err := e.Encode(v)
		if err != nil {
			return err
		}
	}
	return nil
}

//捕获写入时少写的异常
func (e *Encoder) write(p []byte) error {
	n, err := e.w.Write(p)
	if n != len(p) && err == nil {
		return io.ErrShortWrite
	}
	return err
}

//各数据类型编码的具体实现
func (e *Encoder) encode(val reflect.Value) error {
	//判断是否是MarshalSUM接口
	if val.CanInterface() {
		if m, ok := val.Interface().(SUMMarshaler); ok {
			return m.MarshalSUM(e.w)
		}
	}

	switch val.Kind() {
	case reflect.Ptr:
		//写入1或者0
		if err := e.Encode(!val.IsNil()); err != nil {
			return err
		}
		if !val.IsNil() {
			return e.encode(val.Elem())
		}
	case reflect.Bool:
		if val.Bool() {
			return e.write([]byte{1})
		} else {
			return e.write([]byte{0})
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.write(EncInt64(val.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return WriteUint64(e.w, val.Uint())
	case reflect.String:
		return WritePrefix(e.w, []byte(val.String()))
	case reflect.Slice:
		//因为Slice是可变长的类型,所以先写入长度再进行数组处理
		if err := WriteInt(e.w, val.Len()); err != nil {
			return err
		}
		if val.Len() == 0 {
			return nil
		}
		fallthrough
	case reflect.Array:
		//特殊的位数组
		if val.Type().Elem().Kind() == reflect.Uint8 {
			if val.CanAddr() {
				return e.write(val.Slice(0, val.Len()).Bytes())
			}
			slice := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), val.Len(), val.Len())
			reflect.Copy(slice, val)
			return e.write(slice.Bytes())
		}

		//常规的数组和切片
		for i := 0; i < val.Len(); i++ {
			if err := e.encode(val.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			if err := e.encode(val.Field(i)); err != nil {
				return err
			}
		}
		return nil
	}
	panic("could not marshal type" + val.Type().String())
}

//新建编码器
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

//编码接口
func Marshal(v interface{}) []byte {
	b := new(bytes.Buffer)
	NewEncoder(b).Encode(v)
	return b.Bytes()
}

//批量编码接口
func MarshalAll(vs ...interface{}) []byte {
	b := new(bytes.Buffer)
	enc := NewEncoder(b)
	_ = enc.EncodeAll(vs...)
	return b.Bytes()
}

//将接口写入文件中
func WriteFile(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	err = NewEncoder(file).Encode(v)
	if err != nil {
		return errors.New("error while writing" + filename + ": " + err.Error())
	}
	return nil
}

type Decoder struct {
	r io.Reader
	n int
}

//捕获读取时多读取的异常
func (d *Decoder) Read(p []byte) (int, error) {
	n, err := d.r.Read(p)
	if d.n += n; d.n > MaxObjectSize {
		panic(ErrObjectTooLarge)
	}
	return n, err
}

//解码接口
func (d *Decoder) Decode(v interface{}) (err error) {
	//v是一个指针
	pval := reflect.ValueOf(v)
	if pval.Kind() != reflect.Ptr || pval.IsNil() {
		return errBadPointer
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not decode type %s: %v", pval.Elem().Type().String(), r)
		}
	}()

	//重置读取计数
	d.n = 0

	d.decode(pval.Elem())
	return
}

//批量解码接口
func (d *Decoder) DecodeAll(vs ...interface{}) error {
	for _, v := range vs {
		if err := d.Decode(v); err != nil {
			return err
		}
	}
	return nil
}

//读取n位数据
func (d *Decoder) readN(n int) []byte {
	if buf, ok := d.r.(*bytes.Buffer); ok {
		b := buf.Next(n)
		if len(b) != n {
			panic(io.ErrUnexpectedEOF)
		}
		if d.n += n; d.n > MaxObjectSize {
			panic(ErrObjectTooLarge)
		}
		return b
	}
	b := make([]byte, n)
	_, err := io.ReadFull(d, b)
	if err != nil {
		panic(err)
	}
	return b
}

//各类型数据的解码具体实现，从数据流中读取数据，解码至val
func (d *Decoder) decode(val reflect.Value) {
	if val.CanAddr() && val.Addr().CanInterface() {
		if u, ok := val.Addr().Interface().(SUMUnmarshaler); ok {
			err := u.UnmarshalSUM(d.r)
			if err != nil {
				panic(err)
			}
			return
		}
	}

	switch val.Kind() {
	case reflect.Ptr:
		var valid bool
		d.decode(reflect.ValueOf(&valid).Elem())
		//如果是空指针的话，不进行解码
		if !valid {
			return
		}
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		d.decode(val.Elem())
	case reflect.Bool:
		b := d.readN(1)
		if b[0] > 1 {
			panic("boolean value was not 0 or 1")
		}
		val.SetBool(b[0] == 1)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val.SetInt(DecInt64(d.readN(8)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val.SetUint(DecUint64(d.readN(8)))
	case reflect.String:
		strLen := DecUint64(d.readN(8))
		if strLen > MaxSliceSize {
			panic("string is too large")
		}
		val.SetString(string(d.readN(int(strLen))))
	case reflect.Slice:
		//slice是可变长的，首先分配地址空间，然后和数组一样处理
		sliceLen := DecUint64(d.readN(8))
		if sliceLen > 1<<31-1 || sliceLen*uint64(val.Type().Elem().Size()) > MaxSliceSize {
			panic(ErrSliceTooLarge)
		} else if sliceLen == 0 {
			return
		}
		val.Set(reflect.MakeSlice(val.Type(), int(sliceLen), int(sliceLen)))
		fallthrough
	case reflect.Array:
		//特殊的位数组
		if val.Type().Elem().Kind() == reflect.Uint8 {
			b := val.Slice(0, val.Len())
			_, err := io.ReadFull(d, b.Bytes())
			if err != nil {
				panic(err)
			}
			return
		}

		for i := 0; i < val.Len(); i++ {
			d.decode(val.Index(i))
		}
		return
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			d.decode(val.Field(i))
		}
		return
	default:
		panic("unknown type")
	}
}

//新建解码器
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r, 0}
}

//解码b至v接口中
func Unmarshal(b []byte, v interface{}) error {
	r := bytes.NewBuffer(b)
	return NewDecoder(r).Decode(v)
}

//批量解码b至vs接口中
func UnmarshalAll(b []byte, vs ...interface{}) error {
	dec := NewDecoder(bytes.NewBuffer(b))
	return dec.DecodeAll(vs...)
}

//从文件中读取并解码至v接口中
func ReadFile(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	err = NewDecoder(file).Decode(v)
	if err != nil {
		return errors.New("error while reading " + filename + ": " + err.Error())
	}
	return nil
}
