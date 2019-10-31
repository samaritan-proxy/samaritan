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
	"io"
)

const defaultBufferSize = 4096

type sliceAlloc struct {
	allocs int // alloc times from runtime
	buf    []byte
}

// Make creates a slice from buffer if we can
func (d *sliceAlloc) Make(n int) (ss []byte) {
	switch {
	case n == 0:
		return []byte{}
	case n >= 512:
		return d.alloc(n)
	default:
		// TODO(kirk91): reduce memory fragmentation
		if len(d.buf) < n {
			d.buf = d.alloc(8192)
		}
		ss, d.buf = d.buf[:n:n], d.buf[n:]
		return ss
	}
}

func (d *sliceAlloc) alloc(n int) []byte {
	d.allocs++
	return make([]byte, n)
}

// Reader implements Reader for buffer
type Reader struct {
	buf   []byte
	rd    io.Reader
	err   error
	r     int
	w     int
	slice sliceAlloc
}

// NewReader creates Reader
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufferSize)
}

// NewReaderSize creates Reader with buffer size
func NewReaderSize(rd io.Reader, size int) *Reader {
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufferSize
	}
	return &Reader{rd: rd, buf: make([]byte, size)}
}

func (b *Reader) fill() error {
	if b.err != nil {
		return b.err
	}
	if b.r > 0 {
		n := copy(b.buf, b.buf[b.r:b.w])
		b.r = 0
		b.w = n
	}
	n, err := b.rd.Read(b.buf[b.w:])
	if err != nil {
		b.err = err
	} else if n == 0 {
		b.err = io.ErrNoProgress
	} else {
		b.w += n
	}
	return b.err
}

func (b *Reader) buffered() int {
	return b.w - b.r
}

func (b *Reader) Read(p []byte) (int, error) {
	if b.err != nil || len(p) == 0 {
		return 0, b.err
	}
	if b.buffered() == 0 {
		if len(p) >= len(b.buf) {
			n, err := b.rd.Read(p)
			if err != nil {
				b.err = err
			}
			return n, b.err
		}
		if b.fill() != nil {
			return 0, b.err
		}
	}
	n := copy(p, b.buf[b.r:b.w])
	b.r += n
	return n, nil
}

// ReadByte reads a single byte
func (b *Reader) ReadByte() (byte, error) {
	if b.err != nil {
		return 0, b.err
	}
	if b.buffered() == 0 {
		if b.fill() != nil {
			return 0, b.err
		}
	}
	c := b.buf[b.r]
	b.r++
	return c, nil
}

// PeekByte returns a single byte but does not consume
func (b *Reader) PeekByte() (byte, error) {
	if b.err != nil {
		return 0, b.err
	}
	if b.buffered() == 0 {
		if b.fill() != nil {
			return 0, b.err
		}
	}
	c := b.buf[b.r]
	return c, nil
}

// ReadSlice reads a slice until a delimiter
func (b *Reader) ReadSlice(delim byte) ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}
	for {
		var index = bytes.IndexByte(b.buf[b.r:b.w], delim)
		if index >= 0 {
			limit := b.r + index + 1
			slice := b.buf[b.r:limit]
			b.r = limit
			return slice, nil
		}
		if b.buffered() == len(b.buf) {
			b.r = b.w
			return b.buf, bufio.ErrBufferFull
		}
		if b.fill() != nil {
			return nil, b.err
		}
	}
}

// ReadBytes reads bytes until a delimiter
func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	var full [][]byte
	var last []byte
	var size int
	for last == nil {
		f, err := b.ReadSlice(delim)
		if err != nil {
			if err != bufio.ErrBufferFull {
				return nil, b.err
			}
			// NOTE: use customize slice allocator to reduce allocs.
			dup := b.slice.Make(len(f))
			copy(dup, f)
			full = append(full, dup)
		} else {
			last = f
		}
		size += len(f)
	}
	var n int
	var buf = b.slice.Make(size)
	for _, frag := range full {
		n += copy(buf[n:], frag)
	}
	copy(buf[n:], last)
	return buf, nil
}

// ReadFull reads n bytes
func (b *Reader) ReadFull(n int) ([]byte, error) {
	if b.err != nil || n == 0 {
		return nil, b.err
	}
	// NOTE: use customize slice allocator to reduce allocs.
	var buf = b.slice.Make(n)
	if _, err := io.ReadFull(b, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
