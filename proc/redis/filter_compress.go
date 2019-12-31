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

	"github.com/samaritan-proxy/samaritan/pb/config/protocol/redis"
	"github.com/samaritan-proxy/samaritan/proc/redis/compressor"
	"github.com/samaritan-proxy/samaritan/proc/redis/compressor/snappy"
	"github.com/samaritan-proxy/samaritan/stats"
)

/*
Transparent compression is to reduce the use of memory and
network bandwidth by big keys and improve request performance.
Mark the data to be compressed by adding a header before the
compressed data, the following diagram depicts the protocol:

                                                +
                                                |
                     header                           compressed data
                                                |
 1byte                                          |
+------+-------+-------+-------+--------+----------------------------------+
|      |       |       |       |        |       |         ...
+--+---+-------+--+----+---+---+----+---+--+-------------------------------+
   |              |        |        |      |    |
   |              |    algorithm    +--+---+    |
   +-------+------+                    +        |
           +                         \r\n       |
      magic number                              |
                                                |
                                                +


Only support string and hash commands currently, will implement others on demand in the future.
*/

const (
	cpsMagicNumber = "(P$"
	cpsHdrLen      = len(cpsMagicNumber) + 3
)

var (
	// compress headers
	cpsHdrs = make(map[redis.Compression_Algorithm][]byte)
	// the commands which are disabled in compress mode.
	bannedCmdsInCps = make(map[string]struct{})
	// well known commands which could skip decompress check
	wkSkipCheckCmdsInDecps = make(map[string]struct{})

	// errors
	errMissingCpsHdr           = errors.New("missing cps header")
	errMissingCpsMagicNumber   = errors.New("missing cps magic number")
	errInvalidCpsAlgorithm     = errors.New("invalid cps algorithm")
	errUnsupportedCpsAlgorithm = errors.New("unsupported cps algorithm")
)

func init() {
	// init the map of commands which is disabled in compress mode
	for _, cmd := range []string{
		"append", "eval", "setbit", "getbit", "setrange", "getrange",
	} {
		bannedCmdsInCps[cmd] = struct{}{}
	}

	// init the map of commands which could skip decompress check.
	for _, cmd := range []string{
		"incr", "decr", "incrby", "decrby", // string
		"hkeys", "hincrby", "hincrbyfloat", // hash
		"lpop", "lpush", "rpop", "rpush", "lrange", "lindex", // list
		"spop", "sunion", "smembers", // set
		"zadd", "zrange", "zcard", "zrangebyscore", // zset
		"cluster", "ping", // special
	} {
		wkSkipCheckCmdsInDecps[cmd] = struct{}{}
	}

	// init cps headers map
	algorithms := []redis.Compression_Algorithm{snappy.Name}
	for _, algorithm := range algorithms {
		var b bytes.Buffer
		b.WriteString(cpsMagicNumber)
		b.WriteByte(byte(algorithm))
		b.Write(CRLF)
		cpsHdrs[algorithm] = b.Bytes()
	}
}

type compressFilter struct {
	cfg   *config
	stats *stats.Scope
}

func newCompressFilter(cfg *config) Filter {
	f := &compressFilter{
		cfg: cfg,
	}
	// TODO: add stats
	return f
}

func (f *compressFilter) Do(cmd string, req *simpleRequest) FilterStatus {
	// skip if compression config is null
	if f.cfg == nil ||
		f.cfg.GetRedisOption() == nil ||
		f.cfg.GetRedisOption().GetCompression() == nil {
		return Continue
	}

	// register decompression hook if needed.
	if _, ok := wkSkipCheckCmdsInDecps[cmd]; !ok {
		req.RegisterHook(func(request *simpleRequest) {
			f.Decompress(request.resp)
		})
	}

	// skip if the compression is not enabled.
	cfg := f.cfg.GetRedisOption().GetCompression()
	if !cfg.Enable {
		return Continue
	}
	// reject the banned commands.
	if _, ok := bannedCmdsInCps[cmd]; ok {
		errStr := fmt.Sprintf("ERR command '%s' is disabled in compress mode", cmd)
		req.SetResponse(newError(errStr))
		return Stop
	}
	f.Compress(cfg, cmd, req.body)
	return Continue
}

func (f *compressFilter) Compress(cfg *redis.Compression, command string, resp *RespValue) {
	// get the offset of first value in resp array, there are two kind command:
	// 1) command key value
	// 2) command key field1 value1 [field2 value2]...
	//    command key time value
	var offset int
	switch command {
	case "set", "getset", "setnx":
		offset = 2
	case "hset", "hmset", "hsetnx", "psetex", "setex":
		offset = 3
	default:
		return
	}

	for i := offset; i < len(resp.Array); i += 2 {
		r := resp.Array[i]
		if uint32(len(r.Text)) < cfg.Threshold {
			continue
		}
		r.Text = f.compress(r.Text, cfg.Algorithm)
		resp.Array[i] = r
	}
}

func (f *compressFilter) compress(src []byte, algorithm redis.Compression_Algorithm) []byte {
	b := newBuffer()
	defer b.Close()

	w, err := compressor.NewWriter(algorithm.String(), b)
	if err != nil {
		return src
	}

	b.Write(cpsHdrs[algorithm])
	if _, err := w.Write(src); err != nil {
		w.Close()
		return src
	}
	w.Close()

	// Returns the original content directly if the length of
	// the compressed content is greater than the length of the
	// original content
	if b.Len() >= len(src) {
		return src
	}
	n := copy(src, b.Bytes())
	return src[:n]
}

func (f *compressFilter) Decompress(resp *RespValue) {
	switch resp.Type {
	case Integer, Error:
		return
	case Array:
		for idx, r := range resp.Array {
			f.Decompress(&r)
			resp.Array[idx] = r
		}
	default:
		if dst, err := f.decompress(resp.Text); err == nil {
			resp.Text = dst
		}
	}
}

func (f *compressFilter) decompress(src []byte) ([]byte, error) {
	// check header
	if len(src) < cpsHdrLen {
		return nil, errMissingCpsHdr
	}
	if !bytes.Equal([]byte(cpsMagicNumber), src[:len(cpsMagicNumber)]) {
		return nil, errMissingCpsMagicNumber
	}
	algorithm, ok := redis.Compression_Algorithm_name[int32(src[len(cpsMagicNumber)])]
	if !ok {
		return nil, errInvalidCpsAlgorithm
	}

	// decode with specified algorithm
	br := newReader()
	defer br.Close()
	r, err := compressor.NewReader(algorithm, br)
	if err != nil {
		return nil, errUnsupportedCpsAlgorithm
	}

	br.Reset(src[cpsHdrLen:])
	b := newBuffer()
	defer b.Close()
	_, err = b.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	dst := make([]byte, b.Len())
	copy(dst, b.Bytes())
	return dst, nil
}

func (*compressFilter) Destroy() {
	// do nothing
}
