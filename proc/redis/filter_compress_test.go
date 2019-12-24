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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol/redis"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/proc/redis/compressor"
	_ "github.com/samaritan-proxy/samaritan/proc/redis/compressor/snappy"
)

func newMockWriteCloser(
	writerFunc func(p []byte) (n int, err error),
	closerFunc func() error,
) io.WriteCloser {
	return &mockWriteCloser{writerFunc: writerFunc, closerFunc: closerFunc}
}

type mockWriteCloser struct {
	writerFunc func(p []byte) (n int, err error)
	closerFunc func() error
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return m.writerFunc(p)
}

func (m *mockWriteCloser) Close() error {
	return m.closerFunc()
}

func newMockReader(readerFunc func(p []byte) (n int, err error)) io.Reader {
	return &mockReader{
		readerFunc: readerFunc,
	}
}

type mockReader struct {
	readerFunc func(p []byte) (n int, err error)
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return m.readerFunc(p)
}

func TestCompress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := compressor.NewMockcompressor(ctrl)
	compressor.Register(redis.Compression_MOCK.String(), c)
	defer compressor.UnRegister(redis.Compression_MOCK.String())
	cpsData := []byte("cps_data")
	c.EXPECT().NewWriter(gomock.Any()).DoAndReturn(func(w io.Writer) io.WriteCloser {
		return newMockWriteCloser(func(p []byte) (n int, err error) {
			assert.Equal(t, []byte("uncompressed_data"), p)
			return w.Write(cpsData)
		}, func() error {
			return nil
		})
	})
	dst := compress([]byte("uncompressed_data"), redis.Compression_MOCK)
	assert.Len(t, dst, cpsHdrLen+len(cpsData))
	assert.Equal(t, []byte{40, 80, 36, 0xff, 13, 10}, dst[:cpsHdrLen])
	assert.Equal(t, cpsData, dst[cpsHdrLen:])
}

func genCompressFilter(t *testing.T, enable bool, method redis.Compression_Method, threshold uint32) *compressFilter {
	cfg := newConfig(&service.Config{
		ProtocolOptions: &service.Config_RedisOption{
			RedisOption: &protocol.RedisOption{
				Compression: &redis.Compression{
					Enable:    enable,
					Method:    method,
					Threshold: threshold,
				},
			},
		},
	})
	f := newCompressFilter(cfg)
	return f.(*compressFilter)
}

func TestCompressResp(t *testing.T) {
	backup := compress
	compress = func(src []byte, typ redis.Compression_Method) []byte {
		return []byte(strings.ToUpper(string(src)))
	}
	defer func() {
		compress = backup
	}()
	cases := []struct {
		Input  string
		Expect string
	}{
		{
			Input:  "set key val",
			Expect: "set key val",
		},
		{
			Input:  "set key value",
			Expect: "set key VALUE",
		},
		{
			Input:  "mset key1 value1 key2 value2",
			Expect: "mset key1 VALUE1 key2 VALUE2",
		},
		{
			Input:  "mset key1 va1 key2 value2",
			Expect: "mset key1 va1 key2 VALUE2",
		},
		{
			Input:  "getset key1 value1",
			Expect: "getset key1 VALUE1",
		},
		{
			Input:  "getset key1 val",
			Expect: "getset key1 val",
		},
		{
			Input:  "hset key f1 value1",
			Expect: "hset key f1 VALUE1",
		},
		{
			Input:  "hset key f1 val",
			Expect: "hset key f1 val",
		},
		{
			Input:  "hmset key f1 value1 f2 value2",
			Expect: "hmset key f1 VALUE1 f2 VALUE2",
		},
		{
			Input:  "hmset key f1 val f2 value2",
			Expect: "hmset key f1 val f2 VALUE2",
		},
		{
			Input:  "hsetnx key f1 value",
			Expect: "hsetnx key f1 VALUE",
		},
		{
			Input:  "hsetnx key f1 val",
			Expect: "hsetnx key f1 val",
		},
		{
			Input:  "psetex key 1000 value",
			Expect: "psetex key 1000 VALUE",
		},
		{
			Input:  "psetex key 1000 val",
			Expect: "psetex key 1000 val",
		},
		{
			Input:  "setex key 1000 value",
			Expect: "setex key 1000 VALUE",
		},
		{
			Input:  "setex key 1000 val",
			Expect: "setex key 1000 val",
		},
	}
	filter := genCompressFilter(t, true, redis.Compression_MOCK, 4)
	for idx, c := range cases {
		t.Run(fmt.Sprintf("case %d", idx+1), func(t *testing.T) {
			parts := strings.Split(c.Input, " ")
			input := newStringArray(parts...)
			filter.compress(parts[0], input)
			assert.EqualValues(t, c.Expect, input.String())
		})
	}
}

func TestDecompress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := compressor.NewMockcompressor(ctrl)
	compressor.Register(redis.Compression_MOCK.String(), c)
	defer compressor.UnRegister(redis.Compression_MOCK.String())
	rawData := []byte("compressed_data")
	c.EXPECT().NewReader(gomock.Any()).DoAndReturn(func(r io.Reader) io.Reader {
		return newMockReader(func(p []byte) (n int, err error) {
			_, _ = ioutil.ReadAll(r)
			n = copy(p, rawData)
			err = io.EOF
			return
		})
	}).AnyTimes()

	_, err := decompress(nil)
	assert.Equal(t, errMissingCpsHdr, err)

	buf := newBuffer()
	writeCpsHeader(redis.Compression_MOCK, buf)
	buf.WriteString("compressed_data")
	dst, err := decompress(buf.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, rawData, dst)
}

func genCompressedData(t *testing.T, method interface{}, data interface{}) []byte {
	buf := bytes.NewBuffer(nil)

	buf.WriteString(MagicNumber)

	switch m := method.(type) {
	case byte:
		buf.WriteByte(m)
	case redis.Compression_Method:
		buf.WriteByte(byte(m))
	default:
		t.Fatalf("unexpected type of method: %s", reflect.TypeOf(method).String())
	}

	buf.WriteString(Separator)

	switch d := data.(type) {
	case string:
		buf.WriteString(d)
	case []byte:
		buf.Write(d)
	default:
		t.Fatalf("unexpected type of data")
	}

	return buf.Bytes()
}

func TestDecompressResp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		Input  *RespValue
		Expect *RespValue
	}{
		/*
			Array
		*/
		{
			Input: newByteArray(
				genCompressedData(t, redis.Compression_MOCK, "hello"),
				genCompressedData(t, redis.Compression_MOCK, "hello"),
			),
			Expect: newByteArray([]byte{0, 0, 0, 1}, []byte{0, 0, 0, 1}),
		},
		{
			Input: newByteArray(
				// unknown compress method
				genCompressedData(t, byte(0xfd), "hello"),
				genCompressedData(t, redis.Compression_MOCK, "hello"),
			),
			Expect: newByteArray(
				genCompressedData(t, byte(0xfd), "hello"),
				[]byte{0, 0, 0, 1},
			),
		},
		{
			Input: newByteArray(
				// uncompress error
				genCompressedData(t, redis.Compression_MOCK, "hell"),
				genCompressedData(t, redis.Compression_MOCK, "hello"),
			),
			Expect: newByteArray(
				genCompressedData(t, redis.Compression_MOCK, "hell"),
				[]byte{0, 0, 0, 1},
			),
		},
		/*
			Simple String
		*/
		{
			Input:  newSimpleBytes(genCompressedData(t, redis.Compression_MOCK, "hello")),
			Expect: newSimpleBytes([]byte{0, 0, 0, 1}),
		},
		{
			// unknown compress method
			Input:  newSimpleBytes(genCompressedData(t, byte(0xfd), "hello")),
			Expect: newSimpleBytes(genCompressedData(t, byte(0xfd), "hello")),
		},
		{
			// uncompress error
			Input:  newSimpleBytes(genCompressedData(t, redis.Compression_MOCK, "hell")),
			Expect: newSimpleBytes(genCompressedData(t, redis.Compression_MOCK, "hell")),
		},
		/*
			Others
		*/
		{
			Input:  nil,
			Expect: nil,
		},
		{
			Input:  newInteger(17),
			Expect: newInteger(17),
		},
		{
			Input:  newError("foo"),
			Expect: newError("foo"),
		},
		{
			// the error msg should not be decompress
			Input:  newError(string(genCompressedData(t, redis.Compression_MOCK, "hello"))),
			Expect: newError(string(genCompressedData(t, redis.Compression_MOCK, "hello"))),
		},
	}
	c := compressor.NewMockcompressor(ctrl)
	compressor.Register(redis.Compression_MOCK.String(), c)
	defer compressor.UnRegister(redis.Compression_MOCK.String())
	c.EXPECT().NewReader(gomock.Any()).DoAndReturn(func(r io.Reader) io.Reader {
		return newMockReader(func(p []byte) (n int, err error) {
			b, _ := ioutil.ReadAll(r)
			if !bytes.Equal(b, []byte("hello")) {
				return 0, errors.New("error")
			}
			n = copy(p, []byte{0, 0, 0, 1})
			err = io.EOF
			return
		})
	}).AnyTimes()
	filter := genCompressFilter(t, true, redis.Compression_MOCK, 1)

	for idx, c := range cases {
		t.Run(fmt.Sprintf("case %d", idx+1), func(t *testing.T) {
			filter.decompress(c.Input)
			assert.True(t, c.Expect.Equal(c.Input))
		})
	}
}

func TestCompressFilterDo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := compressor.NewMockcompressor(ctrl)
	compressor.Register(redis.Compression_MOCK.String(), c)
	defer compressor.UnRegister(redis.Compression_MOCK.String())
	c.EXPECT().NewWriter(gomock.Any()).DoAndReturn(func(w io.Writer) io.WriteCloser {
		return newMockWriteCloser(func(p []byte) (n int, err error) {
			assert.Equal(t, []byte("00000000000000000000"), p)
			return w.Write([]byte{0, 0, 0, 1})
		}, func() error {
			return nil
		})
	}).Times(1)
	c.EXPECT().NewReader(gomock.Any()).DoAndReturn(func(r io.Reader) io.Reader {
		return newMockReader(func(p []byte) (n int, err error) {
			b, _ := ioutil.ReadAll(r)
			assert.Equal(t, []byte{0, 0, 0, 1}, b)
			n = copy(p, make([]byte, 10))
			err = io.EOF
			return
		})
	}).Times(1)

	cpsFilter := genCompressFilter(t, true, redis.Compression_MOCK, 1)

	t.Run("set", func(t *testing.T) {
		req := newSimpleRequest(newStringArray("set", "A", "00000000000000000000"))
		cpsFilter.Do("set", req)
		assert.Equal(t, genCompressedData(t, redis.Compression_MOCK, []byte{0, 0, 0, 1}), req.Body().Array[2].Text)
	})

	t.Run("get", func(t *testing.T) {
		req := newSimpleRequest(newStringArray("get", "A"))
		cpsFilter.Do("get", req)
		req.SetResponse(newStringArray(string(genCompressedData(t, redis.Compression_MOCK, []byte{0, 0, 0, 1}))))
		req.Wait()
		assert.Equal(t, make([]byte, 10), req.resp.Array[0].Text)
	})

	t.Run("cfg", func(t *testing.T) {
		for idx, c := range []func(){
			func() {
				cpsFilter.cfg = nil
			},
			func() {
				cpsFilter.cfg.ProtocolOptions = nil
			},
			func() {
				cpsFilter.cfg.GetRedisOption().Compression = nil
			},
			func() {
				cpsFilter.cfg.GetRedisOption().Compression.Enable = false
			},
		} {
			cpsFilter.cfg = genCompressFilter(t, true, redis.Compression_MOCK, 1).cfg
			t.Run(fmt.Sprintf("case %d", idx+1), func(t *testing.T) {
				c()
				req := newSimpleRequest(newStringArray("set", "A", "00000000000000000000"))
				cpsFilter.Do("set", req)
				assert.True(t, req.Body().Equal(newStringArray("set", "A", "00000000000000000000")))
			})

		}
	})
}

func BenchmarkSnappyCompress1K(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := make([]byte, 1024)
		compress(raw, redis.Compression_SNAPPY)
	}
}

func BenchmarkSnappyCompress1M(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := make([]byte, 1024*1024)
		compress(raw, redis.Compression_SNAPPY)
	}
}
