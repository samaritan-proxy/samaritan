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

package snappy

import (
	"io"
	"io/ioutil"
	"sync"

	"github.com/golang/snappy"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol/redis"
	"github.com/samaritan-proxy/samaritan/proc/redis/compressor"
)

const (
	Name = redis.Compression_SNAPPY
)

var (
	poolCompressor   sync.Pool
	poolDecompressor sync.Pool
)

func init() {
	poolCompressor.New = func() interface{} {
		return &writer{
			Writer: snappy.NewBufferedWriter(ioutil.Discard),
			pool:   &poolCompressor,
		}
	}
	compressor.Register(Name.String(), new(snappyCompressor))
}

// writer is wrapper of snappy.Writer
type writer struct {
	*snappy.Writer
	pool *sync.Pool
}

func (w *writer) Close() error {
	defer w.pool.Put(w)
	return w.Writer.Close()
}

// reader is wrapper of snappy.Reader
type reader struct {
	*snappy.Reader
	pool *sync.Pool
}

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == io.EOF {
		r.pool.Put(r)
	}
	return n, err
}

type snappyCompressor struct{}

func (snappyCompressor) NewWriter(w io.Writer) io.WriteCloser {
	tempC := poolCompressor.Get().(*writer)
	tempC.Writer.Reset(w)
	return tempC
}

func (snappyCompressor) NewReader(r io.Reader) io.Reader {
	tempDeC, inPool := poolDecompressor.Get().(*reader)
	if !inPool {
		return &reader{
			Reader: snappy.NewReader(r),
			pool:   &poolDecompressor,
		}
	}
	tempDeC.Reader.Reset(r)
	return tempDeC
}
