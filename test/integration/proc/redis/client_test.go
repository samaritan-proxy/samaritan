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
	"math"
	"strings"

	"github.com/gomodule/redigo/redis"
)

type Client struct {
	redis.Conn
}

func newClient(network, address string) (*Client, error) {
	conn, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &Client{
		Conn: conn,
	}, nil
}

func newClientWithAuth(network, address string) (*Client, error) {
	conn, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}

	_, err = conn.Do("auth", AuthPwd)
	if err != nil && strings.Contains(err.Error(), "ERR Client sent AUTH, but no password is set") {
		err = nil
	}

	if err != nil {
		return nil, err
	}

	return &Client{
		Conn: conn,
	}, nil
}

func (c *Client) Set(key string, value interface{}) (string, error) {
	return redis.String(c.Do("SET", key, value))
}

func (c *Client) Get(key string) (string, error) {
	return redis.String(c.Do("GET", key))
}

func (c *Client) Del(keys ...string) (int, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("DEL", args...))
}

func (c *Client) Dump(key string) (string, error) {
	return redis.String(c.Do("DUMP", key))
}

func (c *Client) Exists(keys ...string) (int, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("EXISTS", args...))
}

func (c *Client) Expire(key string, ttl int) (bool, error) {
	return redis.Bool(c.Do("EXPIRE", key, ttl))
}

func (c *Client) TTL(key string) (int, error) {
	return redis.Int(c.Do("TTL", key))
}

func (c *Client) ExpireAt(key string, when int) (bool, error) {
	return redis.Bool(c.Do("EXPIREAT", key, when))
}

func (c *Client) Persist(key string) (bool, error) {
	return redis.Bool(c.Do("PERSIST", key))
}

func (c *Client) PExpire(key string, time int) (bool, error) {
	return redis.Bool(c.Do("PEXPIRE", key, time))
}

func (c *Client) PExpireAt(key string, when int) (bool, error) {
	return redis.Bool(c.Do("PEXPIREAT", key, when))
}

func (c *Client) PTTL(key string) (int, error) {
	return redis.Int(c.Do("PTTL", key))
}

func (c *Client) Restore(key string, ttl int, val string) (string, error) {
	return redis.String(c.Do("RESTORE", key, ttl, val))
}

func (c *Client) Type(key string) (string, error) {
	return redis.String(c.Do("TYPE", key))
}

func (c *Client) GetRange(key string, start, end int) (string, error) {
	return redis.String(c.Do("GETRANGE", key, start, end))
}

func (c *Client) GetSet(key string, value interface{}) (string, error) {
	return redis.String(c.Do("GETSET", key, value))
}

func (c *Client) Incr(key string) (int, error) {
	return redis.Int(c.Do("Incr", key))
}

func (c *Client) IncrBy(key string, increment int) (int, error) {
	return redis.Int(c.Do("INCRBY", key, increment))
}

func (c *Client) IncrByFloat(key string, increment float64) (float64, error) {
	return redis.Float64(c.Do("INCRBYFLOAT", key, increment))
}

type SortOption struct {
	By            string
	Offset, Count float64
	Get           []string
	Order         string
	IsAlpha       bool
	Store         string
}

func (c *Client) Sort(key string, opt SortOption) ([]string, error) {
	args := []interface{}{key}
	if opt.By != "" {
		args = append(args, "by", opt.By)
	}
	if opt.Offset != 0 || opt.Count != 0 {
		args = append(args, "limit", opt.Offset, opt.Count)
	}
	for _, get := range opt.Get {
		args = append(args, "get", get)
	}
	if opt.Order != "" {
		args = append(args, opt.Order)
	}
	if opt.IsAlpha {
		args = append(args, "alpha")
	}
	if opt.Store != "" {
		args = append(args, "store", opt.Store)
	}
	return redis.Strings(c.Do("SORT", args...))
}

func (c *Client) Append(key string, val string) (int, error) {
	return redis.Int(c.Do("APPEND", key, val))
}

type BitCountOption struct {
	Start, End int64
}

func (c *Client) BitCount(key string, opt *BitCountOption) (int, error) {
	args := []interface{}{key}
	if opt != nil {
		args = append(args, opt.Start, opt.End)
	}
	return redis.Int(c.Do("BITCOUNT", args...))
}

type BitPosOption struct {
	Start, End int64
}

func (c *Client) BitPos(key string, bit int, opt *BitPosOption) (int, error) {
	args := []interface{}{key, bit}
	if opt != nil {
		args = append(args, opt.Start)
		if opt.End != 0 {
			args = append(args, opt.End)
		}
	}
	return redis.Int(c.Do("BITPOS", args...))
}

func (c *Client) Decr(key string) (int, error) {
	return redis.Int(c.Do("DECR", key))
}

func (c *Client) DecrBy(key string, decrement int) (int, error) {
	return redis.Int(c.Do("DECRBY", key, decrement))
}

func (c *Client) SetBit(key string, offset, value int) (int, error) {
	return redis.Int(c.Do("SETBIT", key, offset, value))
}

func (c *Client) GetBit(key string, offset int) (int, error) {
	return redis.Int(c.Do("GETBIT", key, offset))
}

func (c *Client) MGet(keys ...string) ([]string, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Strings(c.Do("MGET", args...))
}

func (c *Client) MSet(pairs ...interface{}) (string, error) {
	return redis.String(c.Do("MSET", pairs...))
}

func (c *Client) PSetEx(key string, ttl int, value interface{}) (string, error) {
	return redis.String(c.Do("PSETEX", key, ttl, value))
}

func (c *Client) SetEx(key string, ttl int, value interface{}) (string, error) {
	return redis.String(c.Do("SETEX", key, ttl, value))
}

func (c *Client) SetNx(key string, value interface{}) (int, error) {
	return redis.Int(c.Do("SETNX", key, value))
}

func (c *Client) SetRange(key string, offset int, value interface{}) (int, error) {
	return redis.Int(c.Do("SETRANGE", key, offset, value))
}

func (c *Client) StrLen(key string) (int, error) {
	return redis.Int(c.Do("STRLEN", key))
}

func (c *Client) HSet(key string, field string, value interface{}) (int, error) {
	return redis.Int(c.Do("HSET", key, field, value))
}

func (c *Client) HDel(key string, fields ...string) (int, error) {
	args := []interface{}{key}
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Int(c.Do("HDEL", args...))
}

func (c *Client) HExists(key string, field string) (bool, error) {
	return redis.Bool(c.Do("HEXISTS", key, field))
}

func (c *Client) HGet(key string, field string) (string, error) {
	return redis.String(c.Do("HGET", key, field))
}

func (c *Client) HGetAll(key string) (map[string]string, error) {
	return redis.StringMap(c.Do("HGETALL", key))
}

func (c *Client) HIncrBy(key string, field string, increment int) (int, error) {
	return redis.Int(c.Do("HINCRBY", key, field, increment))
}

func (c *Client) HIncrByFloat(key string, field string, increment float64) (float64, error) {
	return redis.Float64(c.Do("HINCRBYFLOAT", key, field, increment))
}

func (c *Client) HKeys(key string) ([]string, error) {
	return redis.Strings(c.Do("HKEYS", key))
}

func (c *Client) HLen(key string) (int, error) {
	return redis.Int(c.Do("HLEN", key))
}

func (c *Client) HMGet(key string, fields ...string) ([]string, error) {
	args := make([]interface{}, 1, len(fields)+1)
	args[0] = key
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Strings(c.Do("HMGET", args...))
}

func (c *Client) HMSet(key string, mapping map[string]interface{}) (string, error) {
	args := []interface{}{key}
	for field, value := range mapping {
		args = append(args, field, value)
	}
	return redis.String(c.Do("HMSET", args...))
}

func (c *Client) HSetNx(key string, field string, val interface{}) (bool, error) {
	return redis.Bool(c.Do("HSETNX", key, field, val))
}

func (c *Client) HVals(key string) ([]string, error) {
	return redis.Strings(c.Do("HVALS", key))
}

func (c *Client) HScan(key string, cursor uint, match string, count int) (map[string]string, int, error) {
	args := []interface{}{key, cursor}
	if match != "" {
		args = append(args, "match", match)
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	arr, err := redis.Values(c.Do("HSCAN", args...))
	if err != nil {
		return nil, 0, err
	}
	nextCursor, _ := redis.Int(arr[0], nil)
	data, _ := redis.StringMap(arr[1], nil)
	return data, nextCursor, nil
}

func (c *Client) LRange(key string, start, end int) ([]string, error) {
	return redis.Strings(c.Do("LRANGE", key, start, end))
}

func (c *Client) RPush(key string, values ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(values)+1)
	args[0] = key
	args = append(args, values...)
	return redis.Int(c.Do("RPUSH", args...))
}

func (c *Client) LPush(key string, values ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(values)+1)
	args[0] = key
	args = append(args, values...)
	return redis.Int(c.Do("LPUSH", args...))
}

func (c *Client) LIndex(key string, index int) (string, error) {
	return redis.String(c.Do("LINDEX", key, index))
}

func (c *Client) LInsert(key string, op string, pivot, val interface{}) (int, error) {
	return redis.Int(c.Do("LINSERT", key, op, pivot, val))
}

func (c *Client) LLen(key string) (int, error) {
	return redis.Int(c.Do("LLEN", key))
}

func (c *Client) LPop(key string) (string, error) {
	return redis.String(c.Do("LPOP", key))
}

func (c *Client) LPushX(key string, val interface{}) (int, error) {
	return redis.Int(c.Do("LPUSHX", key, val))
}

func (c *Client) LRem(key string, count int, val interface{}) (int, error) {
	return redis.Int(c.Do("LREM", key, count, val))
}

func (c *Client) LSet(key string, index int, val interface{}) (string, error) {
	return redis.String(c.Do("LSET", key, index, val))
}

func (c *Client) LTrim(key string, start, end int) (string, error) {
	return redis.String(c.Do("LTRIM", key, start, end))
}

func (c *Client) RPop(key string) (string, error) {
	return redis.String(c.Do("RPOP", key))
}

func (c *Client) RPushX(key string, val interface{}) (int, error) {
	return redis.Int(c.Do("RPUSHX", key, val))
}

func (c *Client) RPopLPush(key string, destination string) (string, error) {
	return redis.String(c.Do("RPOPLPUSH", key, destination))
}

func (c *Client) SAdd(key string, members ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(members)+1)
	args[0] = key
	args = append(args, members...)
	return redis.Int(c.Do("SADD", args...))
}

func (c *Client) SMembers(key string) ([]string, error) {
	return redis.Strings(c.Do("SMEMBERS", key))
}

func (c *Client) SCard(key string) (int, error) {
	return redis.Int(c.Do("SCARD", key))
}

func (c *Client) SDiff(keys ...string) ([]string, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Strings(c.Do("SDIFF", args...))
}

func (c *Client) SDiffStore(dest string, keys ...string) (int, error) {
	args := make([]interface{}, 1, len(keys)+1)
	args[0] = dest
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("SDIFFSTORE", args...))
}

func (c *Client) SInter(keys ...string) ([]string, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Strings(c.Do("SINTER", args...))
}

func (c *Client) SInterStore(dest string, keys ...string) (int, error) {
	args := make([]interface{}, 1, len(keys)+1)
	args[0] = dest
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("SINTERSTORE", args...))
}

func (c *Client) SIsMember(key string, memeber interface{}) (bool, error) {
	return redis.Bool(c.Do("SISMEMBER", key, memeber))
}

func (c *Client) SMove(source, destination string, member interface{}) (bool, error) {
	return redis.Bool(c.Do("SMOVE", source, destination, member))
}

func (c *Client) SPop(key string) (string, error) {
	return redis.String(c.Do("SPOP", key))
}

func (c *Client) SRandMember(key string) (string, error) {
	return redis.String(c.Do("SRANDMEMBER", key))
}

func (c *Client) SRem(key string, members ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(members)+1)
	args[0] = key
	args = append(args, members...)
	return redis.Int(c.Do("SREM", args...))
}

func (c *Client) SUnion(keys ...string) ([]string, error) {
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Strings(c.Do("SUNION", args...))
}

func (c *Client) SUnionStore(dest string, keys ...string) (int, error) {
	args := make([]interface{}, 1, len(keys)+1)
	args[0] = dest
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("SUNIONSTORE", args...))
}

func (c *Client) SScan(key string, cursor int, match string, count int) ([]string, int, error) {
	args := []interface{}{key, cursor}
	if match != "" {
		args = append(args, "match", match)
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	arr, err := redis.Values(c.Do("SSCAN", args...))
	if err != nil {
		return nil, 0, err
	}
	nextCursor, _ := redis.Int(arr[0], nil)
	keys, _ := redis.Strings(arr[1], nil)
	return keys, nextCursor, nil
}

type Z struct {
	Score  float64
	Member interface{}
}

func (c *Client) ZAdd(key string, members ...Z) (int, error) {
	args := make([]interface{}, 1, 2*len(members)+1)
	args[0] = key
	for _, m := range members {
		args = append(args, m.Score)
		args = append(args, m.Member)
	}
	return redis.Int(c.Do("ZADD", args...))
}

func (c *Client) ZRange(key string, start, stop int) ([]string, error) {
	return redis.Strings(c.Do("ZRANGE", key, start, stop))
}

func (c *Client) ZRangeWithScores(key string, start, stop int) ([]Z, error) {
	values, err := redis.Values(c.Do("ZRANGE", key, start, stop, "WITHSCORES"))
	if err != nil {
		return nil, err
	}
	n := len(values)
	zz := make([]Z, n/2)
	for i := 0; i < n; i += 2 {
		z := &zz[i/2]
		z.Member, _ = redis.String(values[i], nil)
		z.Score, _ = redis.Float64(values[i+1], nil)
	}
	return zz, nil
}

func (c *Client) ZCard(key string) (int, error) {
	return redis.Int(c.Do("ZCARD", key))
}

func (c *Client) ZCount(key string, min, max string) (int, error) {
	return redis.Int(c.Do("ZCOUNT", key, min, max))
}

func (c *Client) ZIncrBy(key string, increment float64, member string) (float64, error) {
	return redis.Float64(c.Do("ZINCRBY", key, increment, member))
}

type ZStore struct {
	Weights   []float64
	Aggregate string
}

func (c *Client) ZInterStore(dest string, store ZStore, keys ...string) (int, error) {
	args := make([]interface{}, 2+len(keys))
	args[0] = dest
	args[1] = len(keys)
	for i, key := range keys {
		args[2+i] = key
	}
	if len(store.Weights) > 0 {
		args = append(args, "weights")
		for _, weight := range store.Weights {
			args = append(args, weight)
		}
	}
	if store.Aggregate != "" {
		args = append(args, "aggregate", store.Aggregate)
	}
	return redis.Int(c.Do("ZINTERSTORE", args...))
}

func (c *Client) ZLexCount(key string, min, max string) (int, error) {
	return redis.Int(c.Do("ZLEXCOUNT", key, min, max))
}

type ZRangeBy struct {
	Min, Max      string
	Offset, Count int64
}

func (c *Client) ZRangeByLex(key string, opt ZRangeBy) ([]string, error) {
	return c.zRangeBy("ZRANGEBYLEX", key, opt, false)
}

func (c *Client) ZRangeByScore(key string, opt ZRangeBy) ([]string, error) {
	return c.zRangeBy("ZRANGEBYSCORE", key, opt, false)
}

func (c *Client) zRangeBy(zcmd, key string, opt ZRangeBy, withScores bool) ([]string, error) {
	args := []interface{}{key, opt.Min, opt.Max}
	if withScores {
		args = append(args, "withscores")
	}
	if opt.Offset != 0 || opt.Count != 0 {
		args = append(
			args,
			"limit",
			opt.Offset,
			opt.Count,
		)
	}
	return redis.Strings(c.Do(zcmd, args...))
}

func (c *Client) ZRevRangeByLex(key string, opt ZRangeBy) ([]string, error) {
	return c.zRevRangeBy("ZREVRANGEBYLEX", key, opt)
}

func (c *Client) zRevRangeBy(zcmd, key string, opt ZRangeBy) ([]string, error) {
	args := []interface{}{key, opt.Max, opt.Min}
	if opt.Offset != 0 || opt.Count != 0 {
		args = append(
			args,
			"limit",
			opt.Offset,
			opt.Count,
		)
	}
	return redis.Strings(c.Do(zcmd, args...))
}

func (c *Client) ZRank(key string, member string) (int, error) {
	return redis.Int(c.Do("ZRANK", key, member))
}

func (c *Client) ZRem(key string, members ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(members)+1)
	args[0] = key
	args = append(args, members...)
	return redis.Int(c.Do("ZREM", args...))
}

func (c *Client) ZRemRangeByLex(key string, min, max string) (int, error) {
	return redis.Int(c.Do("ZREMRANGEBYLEX", key, min, max))
}

func (c *Client) ZRemRangeByRank(key string, start, stop int) (int, error) {
	return redis.Int(c.Do("ZREMRANGEBYRANK", key, start, stop))
}

func (c *Client) ZRemRangeByScore(key string, min, max string) (int, error) {
	return redis.Int(c.Do("ZREMRANGEBYSCORE", key, min, max))
}

func (c *Client) ZRevRange(key string, start, stop int) ([]string, error) {
	return redis.Strings(c.Do("ZREVRANGE", key, start, stop))
}

func (c *Client) ZRevRangeByScore(key string, opt ZRangeBy) ([]string, error) {
	return c.zRevRangeBy("ZREVRANGEBYSCORE", key, opt)
}

func (c *Client) ZRevRank(key, member string) (int, error) {
	return redis.Int(c.Do("ZREVRANK", key, member))
}

func (c *Client) ZScore(key, member string) (float64, error) {
	return redis.Float64(c.Do("ZSCORE", key, member))
}

func (c *Client) ZUnionStore(dest string, store ZStore, keys ...string) (int, error) {
	args := make([]interface{}, 2+len(keys))
	args[0] = dest
	args[1] = len(keys)
	for i, key := range keys {
		args[2+i] = key
	}
	if len(store.Weights) > 0 {
		args = append(args, "weights")
		for _, weight := range store.Weights {
			args = append(args, weight)
		}
	}
	if store.Aggregate != "" {
		args = append(args, "aggregate", store.Aggregate)
	}
	return redis.Int(c.Do("ZUnionStore", args...))
}

func (c *Client) ZScan(key string, cursor int, match string, count int) ([]Z, int, error) {
	args := []interface{}{key, cursor}
	if match != "" {
		args = append(args, "match", match)
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	values, err := redis.Values(c.Do("ZSCAN", args...))
	if err != nil {
		return nil, 0, err
	}

	nextCursor, _ := redis.Int(values[0], nil)
	members := values[1].([]interface{})
	n := len(members)
	zz := make([]Z, n/2)
	for i := 0; i < n; i += 2 {
		z := &zz[i/2]
		z.Member, _ = redis.String(members[i], nil)
		z.Score, _ = redis.Float64(members[i+1], nil)
	}
	return zz, nextCursor, nil
}

func (c *Client) PFAdd(key string, elements ...interface{}) (int, error) {
	args := make([]interface{}, 1, len(elements)+1)
	args[0] = key
	args = append(args, elements...)
	return redis.Int(c.Do("PFADD", args...))
}

func (c *Client) PFCount(keys ...string) (int, error) {
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}
	return redis.Int(c.Do("PFCOUNT", args...))
}

func (c *Client) PFMerge(dest string, keys ...string) (string, error) {
	args := make([]interface{}, 1, len(keys)+1)
	args[0] = dest
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.String(c.Do("PFMERGE", args...))
}

type Point struct {
	Longitude, Latitude float64
}

func PointEqual(p1, p2 *Point) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	if math.Round(p1.Longitude) != math.Round(p2.Longitude) {
		return false
	}
	return math.Round(p1.Latitude) == math.Round(p2.Latitude)
}

type Geo struct {
	Point
	Name string
}

func NewGeo(name string, longitude, latitude float64) Geo {
	return Geo{
		Name: name,
		Point: Point{
			Longitude: longitude,
			Latitude:  latitude,
		},
	}
}

func (g *Geo) ToArray() []interface{} {
	return []interface{}{
		g.Longitude,
		g.Latitude,
		g.Name,
	}
}

func GeoEqual(v1, v2 Geo) bool {
	if v1.Name != v2.Name {
		return false
	}
	if math.Round(v1.Longitude) != math.Round(v2.Longitude) {
		return false
	}
	return math.Round(v1.Latitude) == math.Round(v2.Latitude)
}

type DistWithName struct {
	Name string
	Dist float64
}

func DistWithNameEqual(v1, v2 DistWithName) bool {
	if v1.Name != v2.Name {
		return false
	}
	return math.Round(v1.Dist) == math.Round(v2.Dist)
}

type DistWithGeo struct {
	Geo
	Dist float64
}

func DistWithGeoEqual(v1, v2 DistWithGeo) bool {
	if math.Round(v1.Dist) != math.Round(v2.Dist) {
		return false
	}
	return GeoEqual(v1.Geo, v2.Geo)
}

func (c *Client) GeoAdd(key string, geos ...Geo) (int, error) {
	args := make([]interface{}, 0)
	args = append(args, key)
	for _, geo := range geos {
		args = append(args, geo.ToArray()...)
	}
	return redis.Int(c.Do("GEOADD", args...))
}

func (c *Client) GeoDist(key, member1, member2 string, args ...interface{}) (float64, error) {
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, key, member1, member2)
	cmdArgs = append(cmdArgs, args...)
	return redis.Float64(c.Do("GEODIST", cmdArgs...))
}

func (c *Client) GeoHash(key string, members ...string) ([]string, error) {
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, key)
	for _, member := range members {
		cmdArgs = append(cmdArgs, member)
	}
	return redis.Strings(c.Do("GEOHASH", cmdArgs...))
}

func (c *Client) GeoPos(key string, members ...string) ([]*Point, error) {
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, key)
	for _, member := range members {
		cmdArgs = append(cmdArgs, member)
	}
	values, err := redis.Values(c.Do("GEOPOS", cmdArgs...))
	if err != nil {
		return nil, err
	}
	points := make([]*Point, len(values))
	for idx, item := range values {
		switch item.(type) {
		case nil:
			points[idx] = nil
			continue
		default:
		}
		pos, err := redis.Float64s(item, nil)
		if err != nil {
			return nil, err
		}
		points[idx] = &Point{Longitude: pos[0], Latitude: pos[1]}
	}
	return points, nil
}

func (c *Client) toGeos(reply interface{}, err error) ([]Geo, error) {
	if err != nil {
		return nil, err
	}
	items, err := redis.Values(reply, nil)
	if err != nil {
		return nil, err
	}
	geos := make([]Geo, len(items))
	for idx, item := range items {
		data, err := redis.Values(item, nil)
		if err != nil {
			return nil, err
		}
		name, err := redis.String(data[0], nil)
		if err != nil {
			return nil, err
		}
		pos, err := redis.Float64s(data[1], nil)
		if err != nil {
			return nil, err
		}
		geos[idx] = NewGeo(name, pos[0], pos[1])
	}
	return geos, nil
}

func (c *Client) toDistWithNames(reply interface{}, err error) ([]DistWithName, error) {
	if err != nil {
		return nil, err
	}
	items, err := redis.Values(reply, nil)
	if err != nil {
		return nil, err
	}
	dists := make([]DistWithName, len(items))
	for idx, item := range items {
		data, err := redis.Values(item, nil)
		if err != nil {
			return nil, err
		}
		name, err := redis.String(data[0], nil)
		if err != nil {
			return nil, err
		}
		dist, err := redis.Float64(data[1], nil)
		if err != nil {
			return nil, err
		}
		dists[idx] = DistWithName{name, dist}
	}
	return dists, nil
}

func (c *Client) toDistWithGeos(reply interface{}, err error) ([]DistWithGeo, error) {
	if err != nil {
		return nil, err
	}
	items, err := redis.Values(reply, nil)
	if err != nil {
		return nil, err
	}
	dists := make([]DistWithGeo, len(items))
	for idx, item := range items {
		data, err := redis.Values(item, nil)
		if err != nil {
			return nil, err
		}
		name, err := redis.String(data[0], nil)
		if err != nil {
			return nil, err
		}
		dist, err := redis.Float64(data[1], nil)
		if err != nil {
			return nil, err
		}
		pos, err := redis.Float64s(data[2], nil)
		if err != nil {
			return nil, err
		}
		dists[idx] = DistWithGeo{Dist: dist, Geo: NewGeo(name, pos[0], pos[1])}
	}
	return dists, nil
}

func (c *Client) GeoRadius(key string, point Point, radius float64, unit string) ([]string, error) {
	return redis.Strings(c.Do("GEORADIUS", key, point.Longitude, point.Latitude, radius, unit))
}

func (c *Client) GeoRadiusWithCoord(key string, point Point, radius float64, unit string) ([]Geo, error) {
	return c.toGeos(c.Do("GEORADIUS", key, point.Longitude, point.Latitude, radius, unit, "WITHCOORD"))
}

func (c *Client) GeoRadiusWithDist(key string, point Point, radius float64, unit string) ([]DistWithName, error) {
	return c.toDistWithNames(c.Do("GEORADIUS", key, point.Longitude, point.Latitude, radius, unit, "WITHDIST"))
}

func (c *Client) GeoRadiusWithDistAndCoord(key string, point Point, radius float64, unit string) ([]DistWithGeo, error) {
	return c.toDistWithGeos(c.Do("GEORADIUS", key, point.Longitude, point.Latitude, radius, unit, "WITHDIST", "WITHCOORD"))
}

func (c *Client) GeoRadiusByMember(key, member string, radius float64, unit string) ([]string, error) {
	return redis.Strings(c.Do("GEORADIUSBYMEMBER", key, member, radius, unit))
}

func (c *Client) GeoRadiusByMemberGeoRadiusWithCoord(key, member string, radius float64, unit string) ([]Geo, error) {
	return c.toGeos(c.Do("GEORADIUSBYMEMBER", key, member, radius, unit, "WITHCOORD"))
}

func (c *Client) GeoRadiusByMemberGeoRadiusWithDist(key, member string, radius float64, unit string) ([]DistWithName, error) {
	return c.toDistWithNames(c.Do("GEORADIUSBYMEMBER", key, member, radius, unit, "WITHDIST"))
}

func (c *Client) GeoRadiusByMemberGeoRadiusWithDistAndCoord(key, member string, radius float64, unit string) ([]DistWithGeo, error) {
	return c.toDistWithGeos(c.Do("GEORADIUSBYMEMBER", key, member, radius, unit, "WITHDIST", "WITHCOORD"))
}

func (c *Client) Eval(script string, keys []string, args ...interface{}) (interface{}, error) {
	cmdArgs := make([]interface{}, 2, 2+len(keys)+len(args))
	cmdArgs[0] = script
	cmdArgs[1] = len(keys)
	for _, key := range keys {
		cmdArgs = append(cmdArgs, key)
	}
	cmdArgs = append(cmdArgs, args...)
	return c.Do("EVAL", cmdArgs...)
}

func (c *Client) Auth(password string) (string, error) {
	return redis.String(c.Do("AUTH", password))
}

func (c *Client) Ping() (string, error) {
	return redis.String(c.Do("PING"))
}

func (c *Client) Quit() (string, error) {
	return redis.String(c.Do("QUIT"))
}

func (c *Client) Time() ([]string, error) {
	return redis.Strings(c.Do("TIME"))
}

func (c *Client) Info() (string, error) {
	return redis.String(c.Do("INFO"))
}

func (c *Client) Exec(cmd string, args ...interface{}) (interface{}, error) {
	return c.Do(cmd, args...)
}

func (c *Client) Scan(cursor uint64, match string, count int64) ([]string, uint64, error) {
	args := []interface{}{cursor}
	if match != "" {
		args = append(args, "match", match)
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	arr, err := redis.Values(c.Do("SCAN", args...))
	if err != nil {
		return nil, 0, err
	}
	nextCursor, _ := redis.Uint64(arr[0], nil)
	keys, _ := redis.Strings(arr[1], nil)
	return keys, nextCursor, nil
}
