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
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
)

const (
	maxArrayLen      = 1024 * 1024
	maxBulkStringLen = 1024 * 1024 * 512
)

var (
	// ErrBadRespType represents bad resp type
	ErrBadRespType = errors.New("bad resp type")

	// ErrBadCRLFEnd for invalid crlf
	ErrBadCRLFEnd = errors.New("bad CRLF end")

	// ErrBadArrayLen for invalid array len
	ErrBadArrayLen = errors.New("bad array len")
	// ErrBadArrayLenTooLong too long array len
	ErrBadArrayLenTooLong = errors.New("bad array len, too long")

	// ErrBadBulkStringLen for invalid bulk string len
	ErrBadBulkStringLen = errors.New("bad bulk string len")
	// ErrBadBulkStringLenTooLong for too long bulk string len
	ErrBadBulkStringLenTooLong = errors.New("bad bulk string len, too long")

	// ErrBadMultiBulkLen for invalid multi-bulk len
	ErrBadMultiBulkLen = errors.New("bad multi-bulk len")
	// ErrBadMultiBulkContent for invalid multi-bulk content
	ErrBadMultiBulkContent = errors.New("bad multi-bulk content, should be bulkbytes")
)

const (
	CR byte = '\r'
	LF byte = '\n'
)

// crlf is the delimter in redis protocol.
var CRLF = []byte{CR, LF}

type decoder struct {
	br  *Reader
	err error
}

func newDecoder(r io.Reader, bufSize int) *decoder {
	br := NewReaderSize(r, bufSize)
	return &decoder{br: br}
}

func (d *decoder) Decode() (*RespValue, error) {
	if d.err != nil {
		return nil, d.err
	}
	v, err := d.decode()
	if err != nil {
		d.err = err
	}
	return v, err
}

func (d *decoder) decode() (*RespValue, error) {
	b, err := d.br.PeekByte()
	if err != nil {
		return nil, err
	}

	switch typ := RespType(b); typ {
	case Integer, SimpleString, Error, BulkString, Array:
		d.br.ReadByte()
		return d.decodeResp(typ)
	default:
		return d.decodeInline()
	}
}

func (d *decoder) decodeResp(typ RespType) (*RespValue, error) {
	var (
		v   = &RespValue{Type: typ}
		err error
	)
	switch v.Type {
	case Integer:
		v.Int, err = d.decodeInt()
	case SimpleString, Error:
		v.Text, err = d.decodeTextBytes()
	case BulkString:
		v.Text, err = d.decodeBulkString()
	case Array:
		v.Array, err = d.decodeArray()
	default:
		v, err = nil, ErrBadRespType
	}
	return v, err
}

func (d *decoder) decodeInline() (*RespValue, error) {
	// TODO: limit the data size of inline command
	b, err := d.decodeTextBytes()
	if err != nil {
		return nil, err
	}
	// bytes.Split has more logic to handle separator is arrays
	// use more simple implement here
	multi := make([]RespValue, 0, 8)
	for l, r := 0, 0; r <= len(b); r++ {
		if r != len(b) && b[r] != ' ' {
			continue
		}
		if l < r {
			multi = append(multi, RespValue{
				Type: BulkString,
				Text: b[l:r],
			})
		}
		l = r + 1
	}
	if len(multi) == 0 {
		return nil, ErrBadMultiBulkLen
	}
	return newArray(multi), nil
}

func (d *decoder) decodeInt() (int64, error) {
	// TODO(kirk91): Maybe there are some illegal characters before CR, we
	// should find them as soon as possible.
	b, err := d.br.ReadSlice(LF)
	if err != nil {
		return 0, err
	}
	n := len(b) - 2
	if n < 0 || b[n] != CR {
		return 0, ErrBadCRLFEnd
	}
	return btoi64(b[:n])
}

// btoi64 parse bytes to int64
func btoi64(b []byte) (int64, error) {
	if len(b) != 0 && len(b) < 10 {
		// better performance and zero allocs
		var neg, i = false, 0
		switch b[0] {
		case '-':
			neg = true
			fallthrough
		case '+':
			i++
		}
		if len(b) != i {
			var n int64
			for ; i < len(b) && b[i] >= '0' && b[i] <= '9'; i++ {
				n = int64(b[i]-'0') + n*10
			}
			if len(b) == i {
				if neg {
					n = -n
				}
				return n, nil
			}
		}
	}
	return strconv.ParseInt(string(b), 10, 64)
}

func (d *decoder) decodeBulkString() ([]byte, error) {
	n, err := d.decodeInt()
	if err != nil {
		return nil, err
	}
	switch {
	case n < -1:
		return nil, ErrBadBulkStringLen
	case n > maxBulkStringLen:
		return nil, ErrBadBulkStringLenTooLong
	case n == -1:
		return nil, nil
	}
	b, err := d.br.ReadFull(int(n) + 2)
	if err != nil {
		return nil, err
	}
	if b[n] != CR || b[n+1] != LF {
		return nil, ErrBadCRLFEnd
	}
	return b[:n], nil
}

func (d *decoder) decodeTextBytes() ([]byte, error) {
	b, err := d.br.ReadBytes(LF)
	if err != nil {
		return nil, err
	}
	n := len(b) - 2
	if n < 0 || b[n] != CR {
		return nil, ErrBadCRLFEnd
	}
	return b[:n], nil
}

func (d *decoder) decodeArray() ([]RespValue, error) {
	n, err := d.decodeInt()
	if err != nil {
		return nil, err
	}
	switch {
	case n < -1:
		return nil, ErrBadArrayLen
	case n > maxArrayLen:
		return nil, ErrBadArrayLenTooLong
	case n == -1:
		return nil, nil
	}
	array := make([]RespValue, n)
	for i := range array {
		r, err := d.decode()
		if err != nil {
			return nil, err
		}
		array[i] = *r
	}
	return array, nil
}

type encoder struct {
	bw  *bufio.Writer
	err error
}

func newEncoder(w io.Writer, bufSize int) *encoder {
	bw := bufio.NewWriterSize(w, bufSize)
	return &encoder{bw: bw}
}

func (e *encoder) Encode(v *RespValue) error {
	if e.err != nil {
		return e.err
	}
	err := e.encode(v)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *encoder) encode(v *RespValue) error {
	err := e.bw.WriteByte(byte(v.Type))
	if err != nil {
		return err
	}
	switch v.Type {
	case Integer:
		return e.encodeInt(v.Int)
	case Error, SimpleString:
		return e.encodeTextBytes(v.Text)
	case BulkString:
		return e.encodeBulkBytes(v.Text)
	case Array:
		return e.encodeArray(v.Array)
	default:
		return ErrBadRespType
	}
}

func (e *encoder) encodeInt(i int64) error {
	return e.encodeTextString(itoa(i))
}

const (
	minItoa = -128
	maxItoa = 32768
)

var (
	itoaOffset [maxItoa - minItoa + 1]uint32
	itoaBuffer string
)

func init() {
	// make iota buffer to speed up conversion
	var b bytes.Buffer
	for i := range itoaOffset {
		itoaOffset[i] = uint32(b.Len())
		b.WriteString(strconv.Itoa(i + minItoa))
	}
	itoaBuffer = b.String()
}

func itoa(i int64) string {
	if i >= minItoa && i <= maxItoa {
		beg := itoaOffset[i-minItoa]
		if i == maxItoa {
			return itoaBuffer[beg:]
		}
		end := itoaOffset[i-minItoa+1]
		return itoaBuffer[beg:end]
	}
	return strconv.FormatInt(i, 10)
}

func (e *encoder) encodeTextBytes(b []byte) error {
	if _, err := e.bw.Write(b); err != nil {
		return err
	}
	return e.writeCRLF()
}

func (e *encoder) encodeTextString(s string) error {
	if _, err := e.bw.WriteString(s); err != nil {
		return err
	}
	return e.writeCRLF()
}

func (e *encoder) writeCRLF() (err error) {
	_, err = e.bw.Write(CRLF)
	return err
}

func (e *encoder) encodeBulkBytes(b []byte) error {
	if b == nil {
		return e.encodeInt(-1)
	}
	if err := e.encodeInt(int64(len(b))); err != nil {
		return err
	}
	return e.encodeTextBytes(b)
}

func (e *encoder) encodeArray(array []RespValue) error {
	if array == nil {
		return e.encodeInt(-1)
	}
	if err := e.encodeInt(int64(len(array))); err != nil {
		return err
	}
	for _, v := range array {
		if err := e.encode(&v); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	if err := e.bw.Flush(); err != nil {
		e.err = err
	}
	return e.err
}
