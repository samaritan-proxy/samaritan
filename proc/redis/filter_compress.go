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
*/

// MagicNumber & Separator is set for the Header
const (
	MagicNumber = "(P$"
	Separator   = "\r\n"
)

var (
	cpsHdrLen = len(MagicNumber) + 1 + len(Separator)

	errMissingCpsHdr          = errors.New("missing cps header")
	errCpsMagicNumberNotFound = errors.New("cps magic number not found")
	errInvalidCpsType         = errors.New("invalid cps type")
	errUnsupportedCpsType     = errors.New("unsupported cps type")

	// bannedCmdInCps is a set of commands that are disabled in compress mode.
	bannedCmdInCps = map[string]struct{}{
		"append":   {},
		"eval":     {},
		"setbit":   {},
		"getbit":   {},
		"setrange": {},
		"getrange": {},
	}

	// well known commands which could skip decompress check
	wkSkipCheckCmdsInDecps = map[string]struct{}{
		// string commands
		"incr":   {},
		"decr":   {},
		"incrby": {},
		"decrby": {},

		// Hash commands
		"hkeys":        {},
		"hincrby":      {},
		"hincrbyfloat": {},

		// List commands
		"lpop":   {},
		"lpush":  {},
		"rpop":   {},
		"rpush":  {},
		"lrange": {},
		"lindex": {},

		// Set commands
		"spop":     {},
		"sunion":   {},
		"smembers": {},

		// Zset commands
		"zadd":          {},
		"zrange":        {},
		"zcard":         {},
		"zrangebyscore": {},

		"cluster": {},
		"ping":    {},
	}

	cpsHdrs = map[redis.Compression_Method][]byte{}
)

func init() {
	methods := []redis.Compression_Method{snappy.Name, redis.Compression_MOCK}
	for _, method := range methods {
		hdr := make([]byte, cpsHdrLen)
		copy(hdr, MagicNumber)
		hdr[len(MagicNumber)] = byte(method)
		copy(hdr[len(MagicNumber)+1:], Separator)
		cpsHdrs[method] = hdr
	}
}

// writeCpsHeader generates compress header from specific compressType
func writeCpsHeader(typ redis.Compression_Method, buf buffer) {
	buf.Write(cpsHdrs[typ])
}

var compress = func(src []byte, algorithm redis.Compression_Method) []byte {
	b := newBuffer()
	defer b.Close()

	w, err := compressor.NewWriter(algorithm.String(), b)
	if err != nil {
		return src
	}

	writeCpsHeader(algorithm, b)

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

func getCompressorName(id int32) (string, error) {
	name, ok := redis.Compression_Method_name[id]
	if !ok {
		return "", errInvalidCpsType
	}
	return name, nil
}

var decompress = func(src []byte) ([]byte, error) {
	if len(src) < cpsHdrLen {
		return nil, errMissingCpsHdr
	}
	if !bytes.Equal([]byte(MagicNumber), src[:len(MagicNumber)]) {
		return nil, errCpsMagicNumberNotFound
	}
	name, err := getCompressorName(int32(src[len(MagicNumber)]))
	if err != nil {
		return nil, err
	}

	br := newReader()
	defer br.Close()

	r, err := compressor.NewReader(name, br)
	if err != nil {
		return nil, errUnsupportedCpsType
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

var defaultCpsFilterBuilder filterBuilder = filterBuilder(new(cpsFilterBuilder))

type cpsFilterBuilder struct{}

func (*cpsFilterBuilder) Build(p filterBuildParams) (Filter, error) {
	return &CompressFilter{
		cfg:   p.Config,
		stats: p.Scope,
	}, nil
}

type CompressFilter struct {
	cfg   *config
	stats *stats.Scope
}

func (*CompressFilter) isCommandDisabled(command string) bool {
	_, ok := bannedCmdInCps[command]
	return ok
}

func (*CompressFilter) needDecompress(command string) bool {
	_, ok := wkSkipCheckCmdsInDecps[command]
	return !ok
}

func (f *CompressFilter) compress(commandName string, resp *RespValue) {
	if resp == nil {
		return
	}

	var offset int
	// offset of first value in resp array,
	// there are two kind command:
	// 1: like "SET", "MSET", offset is 2
	// 	comand key [key]..
	//
	// 2: like "HSET", "HMSET", "HSETNX", offset is 3
	// 	command key field1 [value1] field2 [value2] ...
	//	command key time [value]
	switch commandName {
	case "set", "mset", "getset", "setnx":
		offset = 2
	case "hset", "hmset", "hsetnx", "psetex", "setex":
		offset = 3
	default:
		return
	}

	cpsCfg := f.cfg.GetRedisOption().GetCompression()

	for i := offset; i < len(resp.Array); i += 2 {
		r := resp.Array[i]
		if uint32(len(r.Text)) < cpsCfg.Threshold {
			continue
		}
		r.Text = compress(r.Text, cpsCfg.Method)
		resp.Array[i] = r
	}
}

func (f *CompressFilter) decompress(resp *RespValue) {
	if resp == nil {
		return
	}

	switch resp.Type {
	case Integer, Error:
		return
	case Array:
		for idx, r := range resp.Array {
			f.decompress(&r)
			resp.Array[idx] = r
		}
	default:
		if dst, err := decompress(resp.Text); err == nil {
			resp.Text = dst
		}
	}
}

func (f *CompressFilter) Do(req *simpleRequest) FilterStatus {
	// skip processing when compression config is null
	if f == nil ||
		f.cfg == nil ||
		f.cfg.GetRedisOption() == nil ||
		f.cfg.GetRedisOption().GetCompression() == nil {
		return Continue
	}

	var (
		commandName = getCommandFromResp(req.body)
		cpsCfg      = f.cfg.GetRedisOption().GetCompression()
	)

	if f.needDecompress(commandName) {
		req.RegisterHook(func(request *simpleRequest) {
			f.decompress(request.resp)
		})
	}

	if !cpsCfg.Enable {
		return Continue
	}

	if f.isCommandDisabled(commandName) {
		errStr := fmt.Sprintf("command: %s is disabled in compress mode", commandName)
		req.SetResponse(newError(errStr))
		return Stop
	}

	f.compress(commandName, req.body)
	return Continue
}

func (*CompressFilter) Destroy() {
	// do nothing
}
