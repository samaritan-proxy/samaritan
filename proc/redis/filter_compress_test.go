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
	"strings"
	"testing"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol/redis"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/proc/redis/compressor"
	_ "github.com/samaritan-proxy/samaritan/proc/redis/compressor/snappy"
	"github.com/stretchr/testify/assert"
)

func makeCompressFilter(cpsCfg *redis.Compression) *compressFilter {
	cfg := makeDefaultConfig()
	cfg.GetRedisOption().Compression = cpsCfg
	f := newCompressFilter(cfg)
	return f.(*compressFilter)
}

func compress(algorithm redis.Compression_Algorithm, data string) string {
	var b bytes.Buffer

	// header
	b.WriteString(cpsMagicNumber)
	b.WriteByte(byte(algorithm))
	b.Write(CRLF)

	// compress
	w, _ := compressor.NewWriter(algorithm.String(), &b)
	w.Write([]byte(data)) //nolint:errcheck
	_ = w.Close()

	return b.String()
}

func TestCompressRequest(t *testing.T) {
	cfg := &redis.Compression{
		Enable:    true,
		Algorithm: redis.Compression_SNAPPY,
	}
	f := makeCompressFilter(cfg)

	tests := []struct {
		cmd       string
		req       *simpleRequest
		expectReq *simpleRequest
	}{}
	addTest := func(cmd string, req, expectReq *simpleRequest) {
		tests = append(tests, struct {
			cmd       string
			req       *simpleRequest
			expectReq *simpleRequest
		}{
			cmd:       cmd,
			req:       req,
			expectReq: expectReq,
		})
	}

	// supported commands
	// fist value position is 2
	for _, cmd := range []string{
		"set", "setnx", "getset",
	} {
		val := strings.Repeat("0", 500)
		compressedVal := compress(cfg.Algorithm, val)
		req := newSimpleRequest(newStringArray(cmd, "key", val))
		expect := newSimpleRequest(newStringArray(cmd, "key", compressedVal))
		addTest(cmd, req, expect)
	}
	// fist value position is 3
	for _, cmd := range []string{
		"hset", "hmset", "hsetnx", "psetex", "setex",
	} {
		val := strings.Repeat("0", 500)
		compressedVal := compress(cfg.Algorithm, val)
		req := newSimpleRequest(newStringArray(cmd, "key", "dummy", val))
		expect := newSimpleRequest(newStringArray(cmd, "key", "dummy", compressedVal))
		addTest(cmd, req, expect)
	}

	// unsupported commands
	// only used to test functionality, more compatibility tests will be in integration test.
	for _, cmd := range []string{
		"hget", "hexists", "sadd",
	} {
		val := strings.Repeat("0", 500)
		req := newSimpleRequest(newStringArray(cmd, "key", val))
		expect := newSimpleRequest(newStringArray(cmd, "key", val))
		addTest(cmd, req, expect)
	}

	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			f.Do(test.cmd, test.req)
			assert.Equal(t, test.expectReq.Body().String(), test.req.Body().String())
		})
	}
}

func TestDecompressNormalResponse(t *testing.T) {
	cfg := &redis.Compression{}
	f := makeCompressFilter(cfg)

	tests := []struct {
		typ        string
		resp       *RespValue
		expectResp *RespValue
	}{
		{
			typ:        "integer",
			resp:       newInteger(1),
			expectResp: newInteger(1),
		},
		{
			typ:        "error",
			resp:       newError("unknown"),
			expectResp: newError("unknown"),
		},
		{
			typ:        "simple string",
			resp:       newSimpleString(compress(redis.Compression_SNAPPY, strings.Repeat("x", 500))),
			expectResp: newSimpleString(strings.Repeat("x", 500)),
		},
		{
			typ:        "bulk string",
			resp:       newBulkString(compress(redis.Compression_SNAPPY, strings.Repeat("x", 500))),
			expectResp: newBulkString(strings.Repeat("x", 500)),
		},
		{
			typ: "array",
			resp: newStringArray(
				compress(redis.Compression_SNAPPY, strings.Repeat("x", 500)),
				compress(redis.Compression_SNAPPY, strings.Repeat("y", 500)),
			),
			expectResp: newStringArray(
				strings.Repeat("x", 500),
				strings.Repeat("y", 500),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.typ, func(t *testing.T) {
			// only want to test response, so it doesn't matter what the request is.
			req := newSimpleRequest(newStringArray("get", "key"))
			f.Do("get", req)
			req.SetResponse(test.resp)

			// assert
			req.Wait()
			assert.Equal(t, test.expectResp.String(), req.Response().String())
		})
	}
}

func TestDecompressInvalidResponse(t *testing.T) {
	cfg := &redis.Compression{}
	f := makeCompressFilter(cfg)

	tests := []struct {
		name    string
		spoilFn func(data *[]byte)
	}{
		{
			name: "missing header",
			spoilFn: func(data *[]byte) {
				*data = []byte("1")
			},
		},
		{
			name: "missing magic number",
			spoilFn: func(data *[]byte) {
				*data = (*data)[len(cpsMagicNumber):]
			},
		},
		{
			name: "invalid algorithm",
			spoilFn: func(data *[]byte) {
				(*data)[len(cpsMagicNumber)] = 128
			},
		},
		{
			name: "invalid body",
			spoilFn: func(data *[]byte) {
				copy((*data)[cpsHdrLen:], bytes.Repeat([]byte("na"), 10))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := newSimpleRequest(newStringArray("get", "a"))
			f.Do("get", req)

			val := []byte(compress(redis.Compression_SNAPPY, strings.Repeat("x", 500)))
			// spoil the value
			test.spoilFn(&val)
			req.SetResponse(newBulkBytes(val))

			// assert
			req.Wait()
			assert.Equal(t, newBulkBytes(val).String(), req.Response().String())
		})
	}
}

func TestCompressFilterDoWithCompressionOff(t *testing.T) {
	run := func(cfg *config) {
		f := newCompressFilter(cfg)
		status := f.Do("get", newSimpleRequest(newStringArray("get", "a")))
		assert.Equal(t, Continue, status)
	}

	// no protocol options
	cfg := makeDefaultConfig()
	cfg.ProtocolOptions = nil
	run(cfg)

	// no compression config
	cfg.ProtocolOptions = &service.Config_RedisOption{
		RedisOption: &protocol.RedisOption{},
	}
	run(cfg)

	// not enabled
	cfg.GetRedisOption().Compression = &redis.Compression{
		Enable: false,
	}
	run(cfg)
}

func TestCompressFilterRejectBannedCommands(t *testing.T) {
	cfg := &redis.Compression{
		Enable: true,
	}
	f := makeCompressFilter(cfg)

	for cmd := range bannedCmdsInCps {
		req := newSimpleRequest(newStringArray(cmd))
		status := f.Do(cmd, req)
		assert.Equal(t, Stop, status)
		assert.Equal(t, Error, req.Response().Type)
	}
}
