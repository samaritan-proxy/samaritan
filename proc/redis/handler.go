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
	"os"
	"strconv"
	"time"

	"github.com/samaritan-proxy/samaritan/stats"
)

var (
	simpleCommands = []string{
		"dump", "expire", "expireat", "persist", "pexpire", "pexpireat", "pttl",
		"restore", "sort", "ttl", "type",

		// string
		"append", "bitcount", "bitpos", "decr", "decrby", "get", "getbit", "getrange",
		"getset", "incr", "incrby", "incrbyfloat", "psetex", "set", "setbit", "setex",
		"setnx", "setrange", "strlen",

		// hash
		"hdel", "hexists", "hget", "hgetall", "hincrby", "hincrbyfloat", "hkeys",
		"hlen", "hmget", "hmset", "hset", "hsetnx", "hstrlen", "hvals", "hscan",

		// list
		"lindex", "linsert", "llen", "lpop", "lpush", "lpushx", "lrange", "lrem",
		"lset", "ltrim", "rpop", "rpoplpush", "rpush", "rpushx",

		// set
		"sadd", "scard", "sdiff", "sdiffstore", "sinter", "sinterstore", "sismember",
		"smembers", "smove", "spop", "srandmember", "srem", "sunion", "sunionstore",
		"sscan",

		// zset
		"zadd", "zcard", "zcount", "zincrby", "zinterstore", "zlexcount", "zrange",
		"zrangebylex", "zrangebyscore", "zrank", "zrem", "zremrangebylex", "zremrangebyrank",
		"zremrangebyscore", "zrevrange", "zrevrangebylex", "zrevrangebyscore", "zrevrank",
		"zscore", "zunionstore", "zscan", "pfadd", "pfcount", "pfmerge",

		// geo
		"geoadd", "geodist", "geohash", "geopos", "georadius", "georadiusbymember",
	}

	sumResultCommands = []string{"del", "exists", "touch", "unlink"}
)

type commandStats struct {
	Total         *stats.Counter
	Success       *stats.Counter
	Error         *stats.Counter
	LatencyMicros *stats.Histogram
}

func newCommandStats(scope *stats.Scope, cmd string) *commandStats {
	cmdScope := scope.NewChild(cmd)
	return &commandStats{
		Total:         cmdScope.Counter("total"),
		Success:       cmdScope.Counter("success"),
		Error:         cmdScope.Counter("error"),
		LatencyMicros: cmdScope.Histogram("latency_micros"),
	}
}

type commandHandleFunc func(*upstream, *rawRequest)

type commandHandler struct {
	stats  *commandStats
	handle commandHandleFunc
}

func handleSimpleCommand(u *upstream, req *rawRequest) {
	body := req.Body()
	if len(body.Array) < 2 {
		req.SetResponse(newError(invalidRequest))
		return
	}

	simpleReq := newSimpleRequest(body)
	simpleReq.RegisterHook(func(simpleReq *simpleRequest) {
		req.SetResponse(simpleReq.Response())
	})
	key := body.Array[1].Text
	u.MakeRequest(key, simpleReq)
}

func handleSumResultCommand(u *upstream, req *rawRequest) {
	sumResultReq, err := newSumResultRequest(req)
	if err != nil {
		req.SetResponse(newError(err.Error()))
		return
	}

	simpleReqs := sumResultReq.Split()
	for i := 0; i < len(simpleReqs); i++ {
		simpleReq := simpleReqs[i]
		key := simpleReq.Body().Array[1].Text
		u.MakeRequest(key, simpleReq)
	}
}

func handleMSet(u *upstream, req *rawRequest) {
	msetReq, err := newMSetRequest(req)
	if err != nil {
		req.SetResponse(newError(err.Error()))
		return
	}

	simpleReqs := msetReq.Split()
	for i := 0; i < len(simpleReqs); i++ {
		simpleReq := simpleReqs[i]
		key := simpleReq.Body().Array[1].Text
		u.MakeRequest(key, simpleReq)
	}
}

func handleMGet(u *upstream, req *rawRequest) {
	mgetReq, err := newMGetRequest(req)
	if err != nil {
		req.SetResponse(newError(err.Error()))
		return
	}

	simpleReqs := mgetReq.Split()
	for i := 0; i < len(simpleReqs); i++ {
		simpleReq := simpleReqs[i]
		key := simpleReq.Body().Array[1].Text
		u.MakeRequest(key, simpleReq)
	}
}

func handleEval(u *upstream, req *rawRequest) {
	// EVAL script numkeys key [key ...] arg [arg ...]
	body := req.Body()
	if len(body.Array) < 4 {
		req.SetResponse(newError(invalidRequest))
		return
	}

	simpleReq := newSimpleRequest(body)
	simpleReq.RegisterHook(func(simpleReq *simpleRequest) {
		req.SetResponse(simpleReq.Response())
	})
	key := body.Array[3].Text
	u.MakeRequest(key, simpleReq)
}

var (
	respOK   = newSimpleString("OK")
	respPong = newSimpleString("PONG")
)

func handlePing(u *upstream, req *rawRequest) {
	req.SetResponse(respPong)
}

func handleQuit(u *upstream, req *rawRequest) {
	req.SetResponse(respOK)
}

func handleTime(u *upstream, req *rawRequest) {
	now := time.Now()
	unixTime := now.Unix()
	nano := int64(now.Nanosecond())
	micro := nano / 1000

	resp := &RespValue{
		Type: Array,
		Array: []RespValue{
			{Type: BulkString, Text: []byte(strconv.FormatInt(unixTime, 10))},
			{Type: BulkString, Text: []byte(strconv.FormatInt(micro, 10))},
		},
	}
	req.SetResponse(resp)
}

func handleSelect(u *upstream, req *rawRequest) {
	req.SetResponse(respOK)
}

func handleInfo(u *upstream, req *rawRequest) {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "pid: %d\n", os.Getegid())
	// TODO: add more details
	req.SetResponse(&RespValue{
		Type: BulkString,
		Text: b.Bytes(),
	})
}

func handleScan(u *upstream, req *rawRequest) {
	// TODO: implement it
}
