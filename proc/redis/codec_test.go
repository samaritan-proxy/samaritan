// Copyright 2019 Samaritan Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var tmap = make(map[int64][]byte)

func init() {
	var n = len(itoaOffset)*2 + 100000
	for i := -n; i <= n; i++ {
		tmap[int64(i)] = []byte(strconv.Itoa(int(i)))
	}
	for i := math.MinInt64; i != 0; i = int(float64(i) / 1.1) {
		tmap[int64(i)] = []byte(strconv.Itoa(int(i)))
	}
	for i := math.MaxInt64; i != 0; i = int(float64(i) / 1.1) {
		tmap[int64(i)] = []byte(strconv.Itoa(int(i)))
	}
}

func TestItoa(t *testing.T) {
	for i, b := range tmap {
		assert.Equal(t, itoa(i), string(b))
	}
	for i := int64(minItoa); i <= maxItoa; i++ {
		assert.Equal(t, itoa(i), strconv.Itoa(int(i)))
	}
}

func TestEncodeSimpleString(t *testing.T) {
	v := newSimpleString("OK")
	testEncodeAndCheck(t, v, []byte("+OK\r\n"))
}

func TestEncodeError(t *testing.T) {
	v := newError("Error")
	testEncodeAndCheck(t, v, []byte("-Error\r\n"))
}

func TestEncodeInteger(t *testing.T) {
	for _, i := range []int{-1, 0, 1024 * 1024} {
		v := newInteger(int64(i))
		testEncodeAndCheck(t, v, []byte(fmt.Sprintf(":%d\r\n", i)))
	}
}

func TestEncodeBulkString(t *testing.T) {
	v := &RespValue{Type: BulkString}
	testEncodeAndCheck(t, v, []byte("$-1\r\n"))
	v.Text = []byte{}
	testEncodeAndCheck(t, v, []byte("$0\r\n\r\n"))
	v.Text = []byte("helloworld!!")
	testEncodeAndCheck(t, v, []byte("$12\r\nhelloworld!!\r\n"))
}

func TestEncodeArray(t *testing.T) {
	v := newArray(nil)
	testEncodeAndCheck(t, v, []byte("*-1\r\n"))
	v.Array = []RespValue{}
	testEncodeAndCheck(t, v, []byte("*0\r\n"))
	v.Array = append(v.Array, *newInteger(0))
	testEncodeAndCheck(t, v, []byte("*1\r\n:0\r\n"))
	v.Array = append(v.Array, *newNullBulkString())
	testEncodeAndCheck(t, v, []byte("*2\r\n:0\r\n$-1\r\n"))
	v.Array = append(v.Array, *newBulkString("test"))
	testEncodeAndCheck(t, v, []byte("*3\r\n:0\r\n$-1\r\n$4\r\ntest\r\n"))
}

func encode(v *RespValue) []byte {
	var b bytes.Buffer
	enc := newEncoder(&b, 8192)
	enc.Encode(v)
	enc.Flush()
	return b.Bytes()
}

func testEncodeAndCheck(t *testing.T, v *RespValue, expect []byte) {
	t.Helper()
	b := encode(v)
	assert.Equal(t, expect, b)
}

func newBenchmarkEncoder(n int) *encoder {
	return newEncoder(ioutil.Discard, 1024*128)
}

func benchmarkEncode(b *testing.B, n int) {
	v := &RespValue{
		Type: BulkString,
		Text: make([]byte, n),
	}
	e := newBenchmarkEncoder(n)
	for i := 0; i < b.N; i++ {
		assert.Nil(b, e.encode(v))
	}
	assert.Nil(b, e.Flush())
}

func BenchmarkEncode16B(b *testing.B)  { benchmarkEncode(b, 16) }
func BenchmarkEncode64B(b *testing.B)  { benchmarkEncode(b, 64) }
func BenchmarkEncode512B(b *testing.B) { benchmarkEncode(b, 512) }
func BenchmarkEncode1K(b *testing.B)   { benchmarkEncode(b, 1024) }
func BenchmarkEncode2K(b *testing.B)   { benchmarkEncode(b, 1024*2) }
func BenchmarkEncode4K(b *testing.B)   { benchmarkEncode(b, 1024*4) }
func BenchmarkEncode16K(b *testing.B)  { benchmarkEncode(b, 1024*16) }
func BenchmarkEncode32K(b *testing.B)  { benchmarkEncode(b, 1024*32) }
func BenchmarkEncode128K(b *testing.B) { benchmarkEncode(b, 1024*128) }

func TestBtoi64(t *testing.T) {
	assert := assert.New(t)
	for i, b := range tmap {
		v, err := btoi64(b)
		assert.Nil(err)
		assert.Equal(v, i)
	}
}

func TestDecodeInvalidData(t *testing.T) {
	test := []string{
		"*hello\r\n",
		"*-100\r\n",
		"*3\r\nhi",
		"*3\r\nhi\r\n",
		"*4\r\n$1",
		"*4\r\n$1\r",
		"*4\r\n$1\n",
		"*2\r\n$3\r\nget\r\n$what?\r\nx\r\n",
		"*4\r\n$3\r\nget\r\n$1\r\nx\r\n",
		"*2\r\n$3\r\nget\r\n$1\r\nx",
		"*2\r\n$3\r\nget\r\n$1\r\nx\r",
		"*2\r\n$3\r\nget\r\n$100\r\nx\r\n",
		"$6\r\nfoobar\r",
		"$0\rn\r\n",
		"$-1\n",
		"*0",
		"*2n$3\r\nfoo\r\n$3\r\nbar\r\n",
		"*-\r\n",
		"+OK\n",
		"-Error message\r",
	}
	for i, s := range test {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			r := bytes.NewReader([]byte(s))
			dec := newDecoder(r, 8192)
			_, err := dec.Decode()
			assert.NotEqual(t, nil, err)
		})
	}
}

func TestDecodeBulkString(t *testing.T) {
	assert := assert.New(t)
	test := "*2\r\n$4\r\nLLEN\r\n$6\r\nmylist\r\n"
	dec := newDecoder(bytes.NewReader([]byte(test)), 8192)
	v, err := dec.Decode()
	assert.Nil(err)
	assert.Equal(2, len(v.Array))
	s1 := v.Array[0]
	assert.Equal(true, bytes.Equal(s1.Text, []byte("LLEN")))
	s2 := v.Array[1]
	assert.Equal(true, bytes.Equal(s2.Text, []byte("mylist")))
}

func TestDecodeInlineString(t *testing.T) {
	test := []string{
		"\r\n", "\r", "\n", " \n",
	}
	for _, s := range test {
		dec := newDecoder(bytes.NewReader([]byte(s)), 8192)
		_, err := dec.Decode()
		assert.Error(t, err)
	}

	test = []string{
		"hello world\r\n",
		"hello world    \r\n",
		"    hello world    \r\n",
		"    hello     world\r\n",
		"    hello     world    \r\n",
	}
	for i, s := range test {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			dec := newDecoder(bytes.NewReader([]byte(s)), 8192)
			a, err := dec.Decode()
			assert.NoError(t, err)
			assert.Len(t, a.Array, 2)
			assert.Equal(t, a.Array[0].Text, []byte("hello"))
			assert.Equal(t, a.Array[1].Text, []byte("world"))
		})
	}
}

func TestDecoder(t *testing.T) {
	test := []string{
		"$6\r\nfoobar\r\n",
		"$0\r\n\r\n",
		"$-1\r\n",
		"*0\r\n",
		"*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
		"*3\r\n:1\r\n:2\r\n:3\r\n",
		"*-1\r\n",
		"+OK\r\n",
		"-Error message\r\n",
		"*2\r\n$1\r\n0\r\n*0\r\n",
		"*3\r\n$4\r\nEVAL\r\n$31\r\nreturn {1,2,{3,'Hello World!'}}\r\n$1\r\n0\r\n",
	}
	for i, s := range test {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			dec := newDecoder(bytes.NewReader([]byte(s)), 8192)
			_, err := dec.Decode()
			assert.Nil(t, err)
		})
	}
}

type loopReader struct {
	buf []byte
	pos int
}

func (b *loopReader) Read(p []byte) (int, error) {
	if b.pos == len(b.buf) {
		b.pos = 0
	}
	n := copy(p, b.buf[b.pos:])
	b.pos += n
	return n, nil
}

func newBenchmarkDecoder(t *testing.B, n int) *decoder {
	v := newArray([]RespValue{
		*newBulkString(string(make([]byte, n))),
	})
	var buf bytes.Buffer
	enc := newEncoder(&buf, 8192)
	err := enc.Encode(v)
	assert.Nil(t, err)
	enc.Flush()
	p := buf.Bytes()
	var b bytes.Buffer
	for i := 0; i < 128 && b.Len() < 1024*1024; i++ {
		_, err := b.Write(p)
		assert.Nil(t, err)
	}
	return newDecoder(bytes.NewReader(b.Bytes()), 1024*128)
}

func benchmarkDecode(b *testing.B, n int) {
	d := newBenchmarkDecoder(b, n)
	for i := 0; i < b.N; i++ {
		v, err := d.Decode()
		assert.Nil(b, err)
		assert.Equal(b, true, len(v.Array) == 1 && len(v.Array[0].Text) == n)
	}
}

func BenchmarkDecode16B(b *testing.B)  { benchmarkDecode(b, 16) }
func BenchmarkDecode64B(b *testing.B)  { benchmarkDecode(b, 64) }
func BenchmarkDecode512B(b *testing.B) { benchmarkDecode(b, 512) }
func BenchmarkDecode1K(b *testing.B)   { benchmarkDecode(b, 1024) }
func BenchmarkDecode2K(b *testing.B)   { benchmarkDecode(b, 1024*2) }
func BenchmarkDecode4K(b *testing.B)   { benchmarkDecode(b, 1024*4) }
func BenchmarkDecode16K(b *testing.B)  { benchmarkDecode(b, 1024*16) }
func BenchmarkDecode32K(b *testing.B)  { benchmarkDecode(b, 1024*32) }
func BenchmarkDecode128K(b *testing.B) { benchmarkDecode(b, 1024*128) }

var testData = []byte("aaaaa bbbbbbb ccccccc ddddd eeeee ffffffffff gg hhhhhhhhhhhhhhhhh")
var sep = []byte(" ")

func BenchmarkBytesSplit(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes.Split(testData, sep)
	}
}

func split(s []byte) [][]byte {
	res := make([][]byte, 0, 8)
	for l, r := 0, 0; r <= len(s); r++ {
		if r == len(s) || s[r] == ' ' {
			if l < r {
				res = append(res, s[l:r])
			}
			l = r + 1
		}
	}
	return res
}

func BenchmarkBytesSplitV2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		split(testData)
	}
}
