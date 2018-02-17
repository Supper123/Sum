package fastrand

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"strconv"
	"sync/atomic"
	"unsafe"

	"golang.org/x/crypto/blake2b"
)

type randReader struct {
	counter      uint64   //计数器前64位
	counterExtra uint64   //计数器后64位
	entropy      [32]byte //初始化种子
}

//全局变量，伪随机数发生器
var Reader io.Reader

//初始化randReader的熵entropy
func init() {
	r := &randReader{}
	n, err := rand.Read(r.entropy[:])
	if err != nil || n != len(r.entropy) {
		panic("not enough entropy to fill fastrand reader at startup")
	}
	Reader = r
}

//生成随机位数组，返回数组b的长度
func (r *randReader) Read(b []byte) (int, error) {
	counter := atomic.AddUint64(&r.counter, 1)
	counterExtra := atomic.LoadUint64(&r.counterExtra)

	//计数器前64位溢出时，计数器后64位加一
	if counter == 1<<63 || counter == math.MaxUint64 {
		atomic.AddUint64(&r.counterExtra, 1)
	}

	//将randReader的数据保存至seed
	seed := make([]byte, 64)
	binary.LittleEndian.PutUint64(seed[0:8], counter)
	binary.LittleEndian.PutUint64(seed[8:16], counterExtra)
	copy(seed[32:], r.entropy[:])

	//内部计数器
	n := 0
	innerCounter := uint64(0)
	innerCounterExtra := uint64(0)
	for n < len(b) {
		binary.LittleEndian.PutUint64(seed[16:24], innerCounter)
		binary.LittleEndian.PutUint64(seed[24:32], innerCounterExtra)

		//对seed进行hash，以生成随机数组
		result := blake2b.Sum512(seed)
		n += copy(b[n:], result[:])

		innerCounter++
		if innerCounter == math.MaxUint64 {
			innerCounterExtra++
		}
	}
	return n, nil
}

//生成随机数组
func Read(b []byte) {
	Reader.Read(b)
}

//生成长度为n的随机位数组
func Bytes(n int) []byte {
	b := make([]byte, n)
	Read(b)
	return b
}

//[0,n)的uint64随机数
func Uint64n(n uint64) uint64 {
	if n == 0 {
		panic("fastrand: argument to Uint64n is 0")
	}
	max := math.MaxUint64 - math.MaxUint64%n
	b := Bytes(8)
	r := *(*uint64)(unsafe.Pointer(&b[0]))
	for r >= max {
		Read(b)
		r = *(*uint64)(unsafe.Pointer(&b[0]))
	}
	return r % n
}

//[0,n)的int随机数
func Intn(n int) int {
	if n <= 0 {
		panic("fastrand: argument to Intn is <= 0: " + strconv.Itoa(n))
	}
	return int(Uint64n(uint64(n)))
}

//[0,n)的big.int随机数
func BigIntn(n *big.Int) *big.Int {
	i, _ := rand.Int(Reader, n)
	return i
}

//一组整数的随机序列[0,n)
func Perm(n int) []int {
	m := make([]int, n)
	for i := 1; i < n; i++ {
		j := Intn(i + 1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}
