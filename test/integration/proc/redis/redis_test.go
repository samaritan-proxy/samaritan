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
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/logger"
)

var (
	ErrNil = redis.ErrNil
)

var (
	c *Client
)

func initRedisClient() {
	addr := getProxyAddress()
	var err error
	c, err = newClient("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to init redis client: %v", err)
	}
}

func TestNullKey(t *testing.T) {
	testNullKey(t)
}

func testNullKey(t *testing.T) {
	result, err := c.Set("", 123)
	assert.NoError(t, err)
	defer c.Del("")
	assert.Equal(t, result, "OK")

	val, err := c.Get("")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)
}

func TestDel(t *testing.T) {
	testDel(t)
}

func testDel(t *testing.T) {
	result, err := c.Set("key1", 123)
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	result, err = c.Set("key2", 456)
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	n, err := c.Del("key1", "key2", "key3")
	assert.NoError(t, err)
	assert.Equal(t, n, 2)
}

func TestDump(t *testing.T) {
	testDump(t)
}

func testDump(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "123")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	result, err = c.Dump("mykey")
	assert.NoError(t, err)
	assert.NotEqual(t, "123", result)
	if len(result) == 0 {
		t.Error("Dump failed: serialized value is empty")
	}
}
func TestExists(t *testing.T) {
	testExists(t)
}

func testExists(t *testing.T) {
	defer c.Del("key1", "key2", "nosuchkey")

	result, err := c.Set("key1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	n, err := c.Exists("key1")
	assert.NoError(t, err)
	assert.Equal(t, n, 1)

	n, err = c.Exists("nosuchkey")
	assert.NoError(t, err)
	assert.Equal(t, n, 0)

	result, err = c.Set("key2", "World")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	n, err = c.Exists("key1", "key2", "nosuchkey")
	assert.NoError(t, err)
	assert.Equal(t, n, 2)
}

func TestTTL(t *testing.T) {
	testTTL(t)
}

func testTTL(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	success, err := c.Expire("mykey", 10)
	assert.NoError(t, err)
	assert.True(t, success)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, ttl, 10)
}

func TestExpire(t *testing.T) {
	testExpire(t)
}

func testExpire(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	success, err := c.Expire("mykey", 10)
	assert.NoError(t, err)
	assert.True(t, success)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, ttl, 10)

	result, err = c.Set("mykey", "Hello World")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	ttl, err = c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, ttl, -1)
}

func TestExpireAt(t *testing.T) {
	testExpireAt(t)
}

func testExpireAt(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	n, err := c.Exists("mykey")
	assert.NoError(t, err)
	assert.Equal(t, n, 1)

	ok, err := c.ExpireAt("mykey", 1293840000)
	assert.NoError(t, err)
	assert.True(t, ok)

	n, err = c.Exists("mykey")
	assert.NoError(t, err)
	assert.Equal(t, n, 0)
}

func TestPersist(t *testing.T) {
	testPersist(t)
}

func testPersist(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	ok, err := c.Expire("mykey", 10)
	assert.NoError(t, err)
	assert.True(t, ok)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, ttl, 10)

	ok, err = c.Persist("mykey")
	assert.NoError(t, err)
	assert.True(t, ok)

	ttl, err = c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, ttl, -1)
}

func TestPexpire(t *testing.T) {
	testPexpire(t)
}

func testPexpire(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	ok, err := c.PExpire("mykey", 3000)
	assert.NoError(t, err)
	assert.True(t, ok)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	assert.NotEqual(t, 0, ttl)
}

func TestPexpireAt(t *testing.T) {
	testPexpireAt(t)
}

func testPexpireAt(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	expiration := 900 * time.Millisecond
	when := time.Now().Add(expiration).UnixNano() / int64(time.Millisecond)
	ok, err := c.PExpireAt("mykey", int(when))
	assert.NoError(t, err)
	assert.True(t, ok)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	if ttl <= 0 {
		t.Errorf("Ttl is %d", ttl)
	}

	pttl, err := c.PTTL("mykey")
	assert.NoError(t, err)
	if pttl <= 0 {
		t.Errorf("Pttl is %d", pttl)
	}
}

func TestPTTL(t *testing.T) {
	testPTTL(t)
}

func testPTTL(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	ok, err := c.Expire("mykey", 1)
	assert.NoError(t, err)
	assert.True(t, ok)

	pttl, err := c.PTTL("mykey")
	assert.NoError(t, err)
	if pttl <= 0 {
		t.Errorf("Pttl is %d", pttl)
	}
}

func TestType(t *testing.T) {
	testType(t)
}

func testType(t *testing.T) {
	defer c.Del("key1", "key2", "key3")

	result, err := c.Set("key1", "value")
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	n, err := c.LPush("key2", "value")
	assert.NoError(t, err)
	assert.Equal(t, n, 1)

	n, err = c.SAdd("key3", "value")
	assert.NoError(t, err)
	assert.Equal(t, n, 1)

	typ, err := c.Type("key1")
	assert.NoError(t, err)
	assert.Equal(t, typ, "string")

	typ, err = c.Type("key2")
	assert.NoError(t, err)
	assert.Equal(t, typ, "list")

	typ, err = c.Type("key3")
	assert.NoError(t, err)
	assert.Equal(t, typ, "set")
}

func TestRestore(t *testing.T) {
	testRestore(t)
}

func testRestore(t *testing.T) {
	defer c.Del("mykey")

	data := "\n\x11\x11\x00\x00\x00\x0e\x00\x00\x00\x03\x00\x00\xf2\x02\xf3\x02\xf4\xff\x06\x00Z1_\x1cg\x04!\x18"
	result, err := c.Restore("mykey", 0, data)
	assert.NoError(t, err)
	assert.Equal(t, result, "OK")

	typ, err := c.Type("mykey")
	assert.NoError(t, err)
	assert.Equal(t, typ, "list")

	values, err := c.LRange("mykey", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, values, []string{"1", "2", "3"})
}

func TestSort(t *testing.T) {
	testSort(t)
}

func testSort(t *testing.T) {
	defer c.Del("mylist")

	result, err := c.RPush("mylist", 2, 1, 3)
	assert.NoError(t, err)
	assert.Equal(t, 3, result)

	values, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, values, []string{"2", "1", "3"})

	values, err = c.Sort("mylist", SortOption{Order: "ASC"})
	assert.NoError(t, err)
	assert.Equal(t, values, []string{"1", "2", "3"})
}

func TestAppend(t *testing.T) {
	testAppend(t)
}

func testAppend(t *testing.T) {
	defer c.Del("mykey", "ts")

	result, err := c.Exists("mykey")
	assert.NoError(t, err)
	assert.Equal(t, result, 0)

	result, err = c.Append("mykey", "Hello")
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, result, 5)

	result, err = c.Append("mykey", " World")
	assert.NoError(t, err)
	assert.Equal(t, result, 11)

	val, err := c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", val)

	result, err = c.Append("ts", "0043")
	assert.NoError(t, err)
	assert.Equal(t, 4, result)

	result, err = c.Append("ts", "0035")
	assert.NoError(t, err)
	assert.Equal(t, 8, result)

	val, err = c.GetRange("ts", 0, 3)
	assert.NoError(t, err)
	assert.Equal(t, "0043", val)

	val, err = c.GetRange("ts", 4, 7)
	assert.NoError(t, err)
	assert.Equal(t, "0035", val)
}

func TestBitCount(t *testing.T) {
	testBitCount(t)
}

func testBitCount(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "foobar")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	count, err := c.BitCount("mykey", nil)
	assert.NoError(t, err)
	assert.Equal(t, 26, count)

	count, err = c.BitCount("mykey", &BitCountOption{Start: 0, End: 0})
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	count, err = c.BitCount("mykey", &BitCountOption{Start: 1, End: 1})
	assert.NoError(t, err)
	assert.Equal(t, 6, count)
}

func TestBitPos(t *testing.T) {
	testBitPos(t)
}

func testBitPos(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "\xff\xf0\x00")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	pos, err := c.BitPos("mykey", 0, nil)
	assert.NoError(t, err)
	assert.Equal(t, 12, pos)

	result, err = c.Set("mykey", "\x00\xff\xf0")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	pos, err = c.BitPos("mykey", 1, &BitPosOption{Start: 0})
	assert.NoError(t, err)
	assert.Equal(t, 8, pos)

	pos, err = c.BitPos("mykey", 1, &BitPosOption{Start: 2})
	assert.NoError(t, err)
	assert.Equal(t, 16, pos)

	result, err = c.Set("mykey", "\x00\x00\x00")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	pos, err = c.BitPos("mykey", 1, nil)
	assert.NoError(t, err)
	assert.Equal(t, -1, pos)
}

func TestDecr(t *testing.T) {
	testDecr(t)
}

func testDecr(t *testing.T) {
	defer c.Del("mykey")

	val, err := c.Set("mykey", 10)
	assert.NoError(t, err)
	assert.Equal(t, "OK", val)

	n, err := c.Decr("mykey")
	assert.NoError(t, err)
	assert.Equal(t, 9, n)
}

func TestDecrBy(t *testing.T) {
	testDecrBy(t)
}

func testDecrBy(t *testing.T) {
	defer c.Del("mykey")

	val, err := c.Set("mykey", 10)
	assert.NoError(t, err)
	assert.Equal(t, "OK", val)

	n, err := c.DecrBy("mykey", 3)
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
}

func TestGet(t *testing.T) {
	testGet(t)
}

func testGet(t *testing.T) {
	defer c.Del("nonexisting", "mykey")

	val, err := c.Get("nonexisting")
	assert.Equal(t, err, ErrNil)
	assert.Equal(t, "", val)

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err = c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestGetBit(t *testing.T) {
	testGetBit(t)
}

func testGetBit(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.SetBit("mykey", 7, 1)
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	bitVal, err := c.GetBit("mykey", 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, bitVal)

	bitVal, err = c.GetBit("mykey", 7)
	assert.NoError(t, err)
	assert.Equal(t, 1, bitVal)

	bitVal, err = c.GetBit("mykey", 100)
	assert.NoError(t, err)
	assert.Equal(t, 0, bitVal)
}

func TestGetRange(t *testing.T) {
	testGetRange(t)
}

func testGetRange(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "This is a string")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.GetRange("mykey", 0, 3)
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, "This", val)

	val, err = c.GetRange("mykey", -3, -1)
	assert.NoError(t, err)
	assert.Equal(t, "ing", val)

	val, err = c.GetRange("mykey", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, "This is a string", val)

	val, err = c.GetRange("mykey", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, "string", val)
}

func TestGetSet(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.GetSet("mykey", "World")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)

	val, err = c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "World", val)
}

func TestIncr(t *testing.T) {
	testIncr(t)
}

func testIncr(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", 10)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	n, err := c.Incr("mykey")
	assert.NoError(t, err)
	assert.Equal(t, 11, n)

	val, err := c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "11", val)
}

func TestIncrBy(t *testing.T) {
	testIncrBy(t)
}

func testIncrBy(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", 10)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	n, err := c.IncrBy("mykey", 5)
	assert.NoError(t, err)
	assert.Equal(t, 15, n)
}

func TestIncrByFloat(t *testing.T) {
	testIncrByFloat(t)
}

func testIncrByFloat(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.Set("mykey", 10.50)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.IncrByFloat("mykey", 0.1)
	assert.NoError(t, err)
	assert.Equal(t, float64(10.6), val)

	result, err = c.Set("mykey", 5.0e3)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err = c.IncrByFloat("mykey", 2.0e2)
	assert.NoError(t, err)
	assert.Equal(t, float64(5200), val)
}

func TestMGet(t *testing.T) {
	testMGet(t)
}

func testMGet(t *testing.T) {
	defer c.Del("key1", "key2", "nonexisting")

	result, err := c.Set("key1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	result, err = c.Set("key2", "World")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	values, err := c.MGet("key1", "key2", "nonexisting")
	assert.NoError(t, err)
	assert.Equal(t, []string{"Hello", "World", ""}, values)
}

func TestMSet(t *testing.T) {
	testMSet(t)
}

func testMSet(t *testing.T) {
	defer c.Del("key1", "key2")

	result, err := c.MSet("key1", "Hello", "key2", "World")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)

	val, err = c.Get("key2")
	assert.NoError(t, err)
	assert.Equal(t, "World", val)
}

func TestPSetEx(t *testing.T) {
	testPSetEx(t)
}

func testPSetEx(t *testing.T) {
	defer c.Del("key")

	result, err := c.PSetEx("key", 1000, "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	pttl, err := c.PTTL("key")
	assert.NoError(t, err)
	if pttl <= 0 {
		t.Errorf("PTTL of 'key' is %d", pttl)
	}

	val, err := c.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestSet(t *testing.T) {
	testSet(t)
}

func testSet(t *testing.T) {
	defer c.Del("key")

	result, err := c.Set("key", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestSetBit(t *testing.T) {
	testSetBit(t)
}

func testSetBit(t *testing.T) {
	defer c.Del("key")

	result, err := c.SetBit("key", 7, 1)
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	result, err = c.SetBit("key", 7, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, "\x00", val)
}

func TestSetEx(t *testing.T) {
	testSetEx(t)
}

func testSetEx(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.SetEx("mykey", 10, "Hello")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	ttl, err := c.TTL("mykey")
	assert.NoError(t, err)
	assert.Equal(t, 10, ttl)

	val, err := c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestSetNx(t *testing.T) {
	testSetNx(t)
}

func testSetNx(t *testing.T) {
	defer c.Del("mykey")

	result, err := c.SetNx("mykey", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.SetNx("mykey", "World")
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	val, err := c.Get("mykey")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestSetRange(t *testing.T) {
	testSetRange(t)
}

func testSetRange(t *testing.T) {
	defer c.Del("key1", "key2")

	result, err := c.Set("key1", "Hello World")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	n, err := c.SetRange("key1", 6, "Redis")
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, 11, n)

	val, err := c.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "Hello Redis", val)

	n, err = c.SetRange("key2", 6, "Redis")
	assert.NoError(t, err)
	assert.Equal(t, 11, n)

	val, err = c.Get("key2")
	assert.NoError(t, err)
	assert.Equal(t, "\u0000\u0000\u0000\u0000\u0000\u0000Redis", val)
}

func TestStrLen(t *testing.T) {
	testStrLen(t)
}

func testStrLen(t *testing.T) {
	defer c.Del("mykey", "nonexisting")

	result, err := c.Set("mykey", "Hello World")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	len, err := c.StrLen("mykey")
	assert.NoError(t, err)
	assert.Equal(t, 11, len)

	len, err = c.StrLen("nonexisting")
	assert.NoError(t, err)
	assert.Equal(t, 0, len)
}

func TestHDel(t *testing.T) {
	testHDel(t)
}

func testHDel(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "foo")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	n, err := c.HDel("myhash", "field1")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	exists, err := c.HExists("myhash", "field2")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestHGet(t *testing.T) {
	testHGet(t)
}

func testHGet(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "foo")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.HGet("myhash", "field1")
	assert.NoError(t, err)
	assert.Equal(t, "foo", val)

	val, err = c.HGet("myhash", "field2")
	assert.Equal(t, err, ErrNil)
	assert.Equal(t, "", val)
}

func TestHGetAll(t *testing.T) {
	testHGetAll(t)
}

func testHGetAll(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "foo")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.HSet("myhash", "field2", "bar")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.HGetAll("myhash")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"field1": "foo", "field2": "bar"}, val)
}

func TestHIncrBy(t *testing.T) {
	testHIncrBy(t)
}

func testHIncrBy(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", 5)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.HIncrBy("myhash", "field1", 1)
	assert.NoError(t, err)
	assert.Equal(t, 6, val)

	val, err = c.HIncrBy("myhash", "field1", -1)
	assert.NoError(t, err)
	assert.Equal(t, 5, val)

	val, err = c.HIncrBy("myhash", "field1", -10)
	assert.NoError(t, err)
	assert.Equal(t, -5, val)
}

func TestHIncrByFloat(t *testing.T) {
	testHIncrByFloat(t)
}

func testHIncrByFloat(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", 10.50)
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.HIncrByFloat("myhash", "field1", 0.1)
	assert.NoError(t, err)
	assert.Equal(t, float64(10.6), val)

	result, err = c.HSet("myhash", "field1", 5.0e3)
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	val, err = c.HIncrByFloat("myhash", "field1", 2.0e2)
	assert.NoError(t, err)
	assert.Equal(t, float64(5200), val)
}

func TestHKeys(t *testing.T) {
	testHKeys(t)
}

func testHKeys(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.HSet("myhash", "field2", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	fields, err := c.HKeys("myhash")
	assert.NoError(t, err)
	assert.Equal(t, []string{"field1", "field2"}, fields)
}

func TestHLen(t *testing.T) {
	testHLen(t)
}

func testHLen(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.HSet("myhash", "field2", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	len, err := c.HLen("myhash")
	assert.NoError(t, err)
	assert.Equal(t, 2, len)
}

func TestHMGet(t *testing.T) {
	testHMGet(t)
}

func testHMGet(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.HSet("myhash", "field2", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	values, err := c.HMGet("myhash", "field1", "field2", "field3")
	assert.NoError(t, err)
	assert.Equal(t, []string{"Hello", "World", ""}, values)
}

func TestHMSet(t *testing.T) {
	testHMSet(t)
}

func testHMSet(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HMSet("myhash", map[string]interface{}{
		"field1": "Hello",
		"field2": "World"},
	)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)
}

func TestHSet(t *testing.T) {
	testHSet(t)
}

func testHSet(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	val, err := c.HGet("myhash", "field1")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)
}

func TestHSetNx(t *testing.T) {
	testHSetNx(t)
}

func testHSetNx(t *testing.T) {
	defer c.Del("myhash")

	success, err := c.HSetNx("myhash", "field", "Hello")
	assert.NoError(t, err)
	assert.True(t, success)

	success, err = c.HSetNx("myhash", "field", "World")
	assert.NoError(t, err)
	assert.False(t, success)
}

func TestHVals(t *testing.T) {
	testHVals(t)
}

func testHVals(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HSet("myhash", "field1", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	result, err = c.HSet("myhash", "field2", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	vals, err := c.HVals("myhash")
	assert.NoError(t, err)
	assert.Equal(t, []string{"Hello", "World"}, vals)
}

func TestHScan(t *testing.T) {
	testHScan(t)
}

func testHScan(t *testing.T) {
	defer c.Del("myhash")

	result, err := c.HMSet("myhash", map[string]interface{}{
		"field1": "Hello", "field2": "World",
	})
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	vals, nextCursor, err := c.HScan("myhash", 0, "", 0)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"field1": "Hello", "field2": "World"}, vals)
	assert.Equal(t, 0, nextCursor)
}

func TestLIndex(t *testing.T) {
	testLIndex(t)
}

func testLIndex(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.LPush("mylist", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.LPush("mylist", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	item, err := c.LIndex("mylist", 0)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", item)

	item, err = c.LIndex("mylist", -1)
	assert.NoError(t, err)
	assert.Equal(t, "World", item)

	item, err = c.LIndex("mylist", 3)
	assert.Equal(t, ErrNil, err)
	assert.Equal(t, "", item)
}

func TestLInsert(t *testing.T) {
	testLInsert(t)
}

func testLInsert(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "World")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.LInsert("mylist", "BEFORE", "World", "There")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Hello", "There", "World"}, items)
}

func TestLLen(t *testing.T) {
	testLLen(t)
}

func testLLen(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.LPush("mylist", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.LPush("mylist", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.LLen("mylist")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)
}

func TestLPop(t *testing.T) {
	testLPop(t)
}

func testLPop(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	item, err := c.LPop("mylist")
	assert.NoError(t, err)
	assert.Equal(t, "one", item)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", "three"}, items)
}

func TestLPush(t *testing.T) {
	testLPush(t)
}

func testLPush(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.LPush("mylist", "world")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.LPush("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, items)
}

func TestLPushX(t *testing.T) {
	testLPushX(t)
}

func testLPushX(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.LPush("mylist", "world")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.LPushX("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.LPushX("myotherlist", "world")
	assert.NoError(t, err)
	assert.Equal(t, 0, llen)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, items)

	items, err = c.LRange("myotherlist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, items)
}

func TestLRange(t *testing.T) {
	testLRange(t)
}

func testLRange(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	items, err := c.LRange("mylist", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one"}, items)

	items, err = c.LRange("mylist", -3, 2)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, items)

	items, err = c.LRange("mylist", -100, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, items)

	items, err = c.LRange("mylist", 5, 10)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, items)
}

func TestLRem(t *testing.T) {
	testLRem(t)
}

func testLRem(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "foo")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	llen, err = c.RPush("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 4, llen)

	rmCount, err := c.LRem("mylist", -2, "hello")
	assert.NoError(t, err)
	assert.Equal(t, 2, rmCount)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hello", "foo"}, items)
}

func TestLSet(t *testing.T) {
	testLSet(t)
}

func testLSet(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	result, err := c.LSet("mylist", 0, "four")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	result, err = c.LSet("mylist", -2, "five")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"four", "five", "three"}, items)
}

func TestLTrim(t *testing.T) {
	testLTrim(t)
}

func testLTrim(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	result, err := c.LTrim("mylist", 1, -1)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", "three"}, items)
}

func TestRPop(t *testing.T) {
	testRPop(t)
}

func testRPop(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	item, err := c.RPop("mylist")
	assert.NoError(t, err)
	assert.Equal(t, "three", item)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, items)
}

func TestRPopLPush(t *testing.T) {
	testRPopLPush(t)
}

func testRPopLPush(t *testing.T) {
	defer c.Del("{tag}mylist", "{tag}myotherlist")

	llen, err := c.RPush("{tag}mylist", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("{tag}mylist", "two")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPush("{tag}mylist", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, llen)

	item, err := c.RPopLPush("{tag}mylist", "{tag}myotherlist")
	assert.NoError(t, err)
	assert.Equal(t, "three", item)

	items, err := c.LRange("{tag}mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, items)

	items, err = c.LRange("{tag}myotherlist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"three"}, items)
}

func TestRPush(t *testing.T) {
	testRPush(t)
}

func testRPush(t *testing.T) {
	defer c.Del("mylist")

	llen, err := c.RPush("mylist", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPush("mylist", "world")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, items)
}

func TestRPushX(t *testing.T) {
	testRPushX(t)
}

func testRPushX(t *testing.T) {
	defer c.Del("mylist", "myotherlist")

	llen, err := c.RPush("mylist", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, llen)

	llen, err = c.RPushX("mylist", "World")
	assert.NoError(t, err)
	assert.Equal(t, 2, llen)

	llen, err = c.RPushX("myotherlist", "World")
	assert.NoError(t, err)
	assert.Equal(t, 0, llen)

	items, err := c.LRange("mylist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Hello", "World"}, items)

	items, err = c.LRange("myotherlist", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, items)
}

func TestSAdd(t *testing.T) {
	testSAdd(t)
}

func testSAdd(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "World")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	members, err := c.SMembers("myset")
	assert.NoError(t, err)
	assertSet(t, []string{"World", "Hello"}, members)
}

func assertSet(t *testing.T, expected []string, actual []string) {
	if len(expected) != len(actual) {
		t.Errorf("Expect %v, but got %v", expected, actual)
	}

	n := len(expected)
	m1 := make(map[string]bool, n)
	m2 := make(map[string]bool, n)

	for _, v := range expected {
		m1[v] = true
	}
	for _, v := range actual {
		m2[v] = true
	}

	if !reflect.DeepEqual(m1, m2) {
		t.Errorf("Expect %v, but got %v", expected, actual)
	}
}

func TestSCard(t *testing.T) {
	testSCard(t)
}

func testSCard(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	len, err := c.SCard("myset")
	assert.NoError(t, err)
	assert.Equal(t, 2, len)
}

func TestSDiff(t *testing.T) {
	testSDiff(t)
}

func testSDiff(t *testing.T) {
	defer c.Del("{tag}key1", "{tag}key2")

	n, err := c.SAdd("{tag}key1", "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "b")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "d")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "e")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	diffed, err := c.SDiff("{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assertSet(t, []string{"a", "b"}, diffed)
}

func TestSDiffStore(t *testing.T) {
	testSDiffStore(t)
}

func testSDiffStore(t *testing.T) {
	defer c.Del("{tag}key1", "{tag}key2", "{tag}key")

	n, err := c.SAdd("{tag}key1", "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "b")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "d")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "e")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SDiffStore("{tag}key", "{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	members, err := c.SMembers("{tag}key")
	assert.NoError(t, err)
	assertSet(t, []string{"a", "b"}, members)
}

func TestSinter(t *testing.T) {
	testSinter(t)
}

func testSinter(t *testing.T) {
	defer c.Del("{tag}key1", "{tag}key2")

	n, err := c.SAdd("{tag}key1", "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "b")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "d")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "e")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	diffed, err := c.SInter("{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assertSet(t, []string{"c"}, diffed)
}

func TestSinterStore(t *testing.T) {
	testSinterStore(t)
}

func testSinterStore(t *testing.T) {
	defer c.Del("{tag}key", "{tag}key1", "{tag}key2")

	n, err := c.SAdd("{tag}key1", "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "b")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key2", "c")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "d")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}key1", "e")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SInterStore("{tag}key", "{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	memebers, err := c.SMembers("{tag}key")
	assert.NoError(t, err)
	assertSet(t, []string{"c"}, memebers)
}

func TestSIsMember(t *testing.T) {
	testSIsMember(t)
}

func testSIsMember(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	isMember, err := c.SIsMember("myset", "one")
	assert.NoError(t, err)
	assert.True(t, isMember)

	isMember, err = c.SIsMember("myset", "two")
	assert.NoError(t, err)
	assert.False(t, isMember)
}

func TestSMembers(t *testing.T) {
	testSMembers(t)
}

func testSMembers(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "Hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "World")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	members, err := c.SMembers("myset")
	assert.NoError(t, err)
	assertSet(t, []string{"World", "Hello"}, members)
}

func TestSMove(t *testing.T) {
	testSMove(t)
}

func testSMove(t *testing.T) {
	defer c.Del("{tag}myset", "{tag}myotherset")

	n, err := c.SAdd("{tag}myset", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}myset", "two")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("{tag}myotherset", "three")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	success, err := c.SMove("{tag}myset", "{tag}myotherset", "two")
	assert.NoError(t, err)
	assert.True(t, success)

	memebers, err := c.SMembers("{tag}myset")
	assert.NoError(t, err)
	assertSet(t, []string{"one"}, memebers)

	memebers, err = c.SMembers("{tag}myotherset")
	assert.NoError(t, err)
	assertSet(t, []string{"three", "two"}, memebers)
}

func TestSPop(t *testing.T) {
	testSPop(t)
}

func testSPop(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "two")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SAdd("myset", "three")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	member, err := c.SPop("myset")
	assert.NoError(t, err)
	assert.NotEqual(t, member, "")

	members, err := c.SMembers("myset")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(members))
}

func TestSRandMember(t *testing.T) {
	testSRandMember(t)
}

func testSRandMember(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "one", "two", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	member, err := c.SRandMember("myset")
	assert.NoError(t, err)
	assert.Contains(t, []string{"one", "two", "three"}, member)
}

func TestSRem(t *testing.T) {
	testSRem(t)
}

func testSRem(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "one", "two", "three")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.SRem("myset", "one")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.SRem("myset", "four")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	members, err := c.SMembers("myset")
	assert.NoError(t, err)
	assertSet(t, []string{"three", "two"}, members)
}

func TestSUnion(t *testing.T) {
	testSUnion(t)
}

func testSUnion(t *testing.T) {
	defer c.Del("{tag}key1", "{tag}key2")

	n, err := c.SAdd("{tag}key1", "a", "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.SAdd("{tag}key2", "c", "d", "e")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	members, err := c.SUnion("{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assertSet(t, []string{"a", "b", "c", "d", "e"}, members)
}

func TestSunionStore(t *testing.T) {
	testSunionStore(t)
}

func testSunionStore(t *testing.T) {
	defer c.Del("{tag}key1", "{tag}key2", "{tag}key")

	n, err := c.SAdd("{tag}key1", "a", "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.SAdd("{tag}key2", "c", "d", "e")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.SUnionStore("{tag}key", "{tag}key1", "{tag}key2")
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	members, err := c.SMembers("{tag}key")
	assert.NoError(t, err)
	assertSet(t, []string{"a", "b", "c", "d", "e"}, members)
}

func TestSScan(t *testing.T) {
	testSScan(t)
}

func testSScan(t *testing.T) {
	defer c.Del("myset")

	n, err := c.SAdd("myset", "a", "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	memebers, nextCursor, err := c.SScan("myset", 0, "", 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, nextCursor)
	assertSet(t, []string{"c", "b", "a"}, memebers)
}

func TestZAdd(t *testing.T) {
	testZAdd(t)
}

func testZAdd(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{1, "uno"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	vals, err := c.ZRangeWithScores("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{1, "one"}, {1, "uno"}, {2, "two"}, {3, "three"}}, vals)
}

func TestZCard(t *testing.T) {
	testZCard(t)
}

func testZCard(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZCard("myzset")
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestZCount(t *testing.T) {
	testZCount(t)
}

func testZCount(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	count, err := c.ZCount("myzset", "-inf", "+inf")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	count, err = c.ZCount("myzset", "(1", "3")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestZIncrBy(t *testing.T) {
	testZIncrBy(t)
}

func testZIncrBy(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("myzset", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	score, err := c.ZIncrBy("myzset", 2, "one")
	assert.NoError(t, err)
	assert.Equal(t, float64(3), score)

	vals, err := c.ZRangeWithScores("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{2, "two"}, {3, "one"}}, vals)
}

func TestZInterScore(t *testing.T) {
	testZInterScore(t)
}

func testZInterScore(t *testing.T) {
	defer c.Del("{tag}zset1", "{tag}zset2", "{tag}out")

	n, err := c.ZAdd("{tag}zset1", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset1", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset2", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset2", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset2", Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZInterStore("{tag}out", ZStore{Weights: []float64{2, 3}}, "{tag}zset1", "{tag}zset2")
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	vals, err := c.ZRangeWithScores("{tag}out", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{5, "one"}, {10, "two"}}, vals)
}

func TestZLexCount(t *testing.T) {
	testZLexCount(t)
}

func testZLexCount(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{0, "a"}, Z{0, "b"}, Z{0, "c"}, Z{0, "d"}, Z{0, "e"})
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	n, err = c.ZAdd("myzset", Z{0, "f"}, Z{0, "g"})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	count, err := c.ZLexCount("myzset", "-", "+")
	assert.NoError(t, err)
	assert.Equal(t, 7, count)

	count, err = c.ZLexCount("myzset", "[b", "[f")
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestZRange(t *testing.T) {
	testZRange(t)
}

func testZRange(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	values, err := c.ZRange("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, values)

	values, err = c.ZRange("myzset", 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"three"}, values)

	values, err = c.ZRange("myzset", -2, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", "three"}, values)
}

func TestZRangeByLex(t *testing.T) {
	testZRangeByLex(t)
}

func testZRangeByLex(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{0, "a"}, Z{0, "b"}, Z{0, "c"}, Z{0, "d"}, Z{0, "e"}, Z{0, "f"}, Z{0, "g"})
	assert.NoError(t, err)
	assert.Equal(t, 7, n)

	values, err := c.ZRangeByLex("myzset", ZRangeBy{Min: "-", Max: "[c"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, values)

	values, err = c.ZRangeByLex("myzset", ZRangeBy{Min: "-", Max: "(c"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, values)

	values, err = c.ZRangeByLex("myzset", ZRangeBy{Min: "[aaa", Max: "(g"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"b", "c", "d", "e", "f"}, values)
}

func TestZRevRangeByLex(t *testing.T) {
	testZRevRangeByLex(t)
}

func testZRevRangeByLex(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{0, "a"}, Z{0, "b"}, Z{0, "c"}, Z{0, "d"}, Z{0, "e"}, Z{0, "f"}, Z{0, "g"})
	assert.NoError(t, err)
	assert.Equal(t, 7, n)

	values, err := c.ZRevRangeByLex("myzset", ZRangeBy{Min: "-", Max: "[c"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, values)

	values, err = c.ZRevRangeByLex("myzset", ZRangeBy{Min: "-", Max: "(c"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"b", "a"}, values)

	values, err = c.ZRevRangeByLex("myzset", ZRangeBy{Min: "[aaa", Max: "(g"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"f", "e", "d", "c", "b"}, values)
}

func TestZRangeByScore(t *testing.T) {
	testZRangeByScore(t)
}

func testZRangeByScore(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	values, err := c.ZRangeByScore("myzset", ZRangeBy{Min: "-inf", Max: "+inf"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, values)

	values, err = c.ZRangeByScore("myzset", ZRangeBy{Min: "1", Max: "2"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, values)

	values, err = c.ZRangeByScore("myzset", ZRangeBy{Min: "(1", Max: "2"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"two"}, values)

	values, err = c.ZRangeByScore("myzset", ZRangeBy{Min: "(1", Max: "(2"})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, values)
}

func TestZRank(t *testing.T) {
	testZRank(t)
}

func testZRank(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	rank, err := c.ZRank("myzset", "three")
	assert.NoError(t, err)
	assert.Equal(t, 2, rank)

	rank, err = c.ZRank("myzset", "four")
	assert.Equal(t, ErrNil, err)
	assert.Equal(t, 0, rank)
}

func TestZRem(t *testing.T) {
	testZRem(t)
}

func testZRem(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.ZRem("myzset", "two")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	vals, err := c.ZRangeWithScores("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{1, "one"}, {3, "three"}}, vals)
}

func TestZRemRangeByLex(t *testing.T) {
	testZRemRangeByLex(t)
}

func testZRemRangeByLex(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{0, "e"}, Z{0, "d"}, Z{0, "c"}, Z{0, "b"}, Z{0, "aaaa"})
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	n, err = c.ZAdd("myzset", Z{0, "alpha"}, Z{0, "ALPHA"}, Z{0, "zip"}, Z{0, "zap"}, Z{0, "foo"})
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	values, err := c.ZRange("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"ALPHA", "aaaa", "alpha", "b", "c", "d", "e", "foo", "zap", "zip"}, values)

	n, err = c.ZRemRangeByLex("myzset", "[alpha", "[omega")
	assert.NoError(t, err)
	assert.Equal(t, 6, n)

	values, err = c.ZRange("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"ALPHA", "aaaa", "zap", "zip"}, values)
}

func TestZRemRangeByRank(t *testing.T) {
	testZRemRangeByRank(t)
}

func testZRemRangeByRank(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.ZRemRangeByRank("myzset", 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	vals, err := c.ZRangeWithScores("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{3, "three"}}, vals)
}

func TestZRemRangeByScore(t *testing.T) {
	testZRemRangeByScore(t)
}

func testZRemRangeByScore(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = c.ZRemRangeByScore("myzset", "-inf", "(2")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	vals, err := c.ZRangeWithScores("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{2, "two"}, {3, "three"}}, vals)
}

func TestZRevRange(t *testing.T) {
	testZRevRange(t)
}

func testZRevRange(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	values, err := c.ZRevRange("myzset", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"three", "two", "one"}, values)

	values, err = c.ZRevRange("myzset", 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one"}, values)

	values, err = c.ZRevRange("myzset", -2, -1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", "one"}, values)
}

func TestZRevRangeByScore(t *testing.T) {
	testZRevRangeByScore(t)
}

func testZRevRangeByScore(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	values, err := c.ZRevRangeByScore("myzset", ZRangeBy{Max: "+inf", Min: "-inf"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"three", "two", "one"}, values)

	values, err = c.ZRevRangeByScore("myzset", ZRangeBy{Max: "2", Min: "1"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", "one"}, values)

	values, err = c.ZRevRangeByScore("myzset", ZRangeBy{Max: "2", Min: "(1"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"two"}, values)

	values, err = c.ZRevRangeByScore("myzset", ZRangeBy{Max: "(2", Min: "(1"})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, values)
}

func TestZRevRank(t *testing.T) {
	testZRevRank(t)
}

func testZRevRank(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	rank, err := c.ZRevRank("myzset", "one")
	assert.NoError(t, err)
	assert.Equal(t, 2, rank)

	rank, err = c.ZRevRank("myzset", "four")
	assert.Equal(t, ErrNil, err)
	assert.Equal(t, 0, rank)
}

func TestZScore(t *testing.T) {
	testZScore(t)
}

func testZScore(t *testing.T) {
	defer c.Del("myzset")

	n, err := c.ZAdd("myzset", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	score, err := c.ZScore("myzset", "one")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), score)
}

func TestZUnionStore(t *testing.T) {
	testZUnionStore(t)
}

func testZUnionStore(t *testing.T) {
	defer c.Del("{tag}zset1", "{tag}zset2", "{tag}out")

	n, err := c.ZAdd("{tag}zset1", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset1", Z{2, "two"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset2", Z{1, "one"})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.ZAdd("{tag}zset2", Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	n, err = c.ZUnionStore("{tag}out", ZStore{Weights: []float64{2, 3}}, "{tag}zset1", "{tag}zset2")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	values, err := c.ZRangeWithScores("{tag}out", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []Z{{5, "one"}, {9, "three"}, {10, "two"}}, values)
}

func TestZScan(t *testing.T) {
	testZScan(t)
}

func testZScan(t *testing.T) {
	defer c.Del("zset1")

	n, err := c.ZAdd("zset1", Z{1, "one"}, Z{2, "two"}, Z{3, "three"})
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	members, cursor, err := c.ZScan("zset1", 0, "", 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, cursor)
	assert.Equal(t, []Z{{1, "one"}, {2, "two"}, {3, "three"}}, members)
}

func TestPFadd(t *testing.T) {
	testPFadd(t)
}

func testPFadd(t *testing.T) {
	defer c.Del("hll")

	n, err := c.PFAdd("hll", "a", "b", "c", "d", "e", "f", "g")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	count, err := c.PFCount("hll")
	assert.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestPFCount(t *testing.T) {
	testPFCount(t)
}

func testPFCount(t *testing.T) {
	defer c.Del("{tag}hll", "{tag}some-other-hll")

	n, err := c.PFAdd("{tag}hll", "foo", "bar", "zap")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.PFAdd("{tag}hll", "zap", "zap", "zap")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = c.PFAdd("{tag}hll", "foo", "bar")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	count, err := c.PFCount("{tag}hll")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	n, err = c.PFAdd("{tag}some-other-hll", 1, 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	count, err = c.PFCount("{tag}some-other-hll")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	count, err = c.PFCount("{tag}hll", "{tag}some-other-hll")
	assert.NoError(t, err)
	assert.Equal(t, 6, count)
}

func TestPFMerge(t *testing.T) {
	testPFMerge(t)
}

func testPFMerge(t *testing.T) {
	defer c.Del("{tag}hll1", "{tag}hll2", "{tag}hll3")

	n, err := c.PFAdd("{tag}hll1", "foo", "bar", "zap", "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = c.PFAdd("{tag}hll2", "a", "b", "c", "foo")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	status, err := c.PFMerge("{tag}hll3", "{tag}hll1", "{tag}hll2")
	assert.NoError(t, err)
	assert.Equal(t, "OK", status)

	count, err := c.PFCount("{tag}hll3")
	assert.NoError(t, err)
	assert.Equal(t, 6, count)
}

func TestGeoAdd(t *testing.T) {
	testGeoAdd(t)
}

func testGeoAdd(t *testing.T) {
	defer c.Del("gk1")
	n, err := c.GeoAdd("gk1",
		NewGeo("Agrigento", 13.583333, 37.316667),
		NewGeo("Catania", 15.087269, 37.502669),
	)
	assert.EqualValues(t, 2, n)
	assert.NoError(t, err)
}

func TestGeoDist(t *testing.T) {
	testGeoDist(t)
}

func testGeoDist(t *testing.T) {
	defer c.Del("gk1")
	_, err := c.GeoAdd("gk1",
		NewGeo("Agrigento", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	)
	assert.NoError(t, err)

	dist, err := c.GeoDist("gk1", "Agrigento", "Catania")
	assert.NoError(t, err)
	assert.EqualValues(t, 166274, int(dist))

	dist, err = c.GeoDist("gk1", "Agrigento", "Catania", "km")
	assert.NoError(t, err)
	assert.EqualValues(t, 166, int(dist))

	dist, err = c.GeoDist("gk1", "Agrigento", "Catania", "mi")
	assert.NoError(t, err)
	assert.EqualValues(t, 103, int(dist))

	dist, err = c.GeoDist("gk1", "Agrigento", "Catania", "ft")
	assert.NoError(t, err)
	assert.EqualValues(t, 545518, int(dist))

	dist, err = c.GeoDist("gk1", "Agrigento", "Foo")
	assert.Equal(t, redis.ErrNil, err)
	assert.EqualValues(t, 0.0, dist)
}

func TestGeoHash(t *testing.T) {
	testGeoHash(t)
}

func testGeoHash(t *testing.T) {
	defer c.Del("gk1")
	_, err := c.GeoAdd("gk1",
		NewGeo("Agrigento", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	)
	assert.NoError(t, err)

	hashs, err := c.GeoHash("gk1", "Agrigento", "Catania", "Foo")
	assert.Equal(t, []string{"sqc8b49rny0", "sqdtr74hyu0", ""}, hashs)
	assert.NoError(t, err)
}

func TestGeoPos(t *testing.T) {
	testGeoPos(t)
}

func testGeoPos(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Agrigento", 1.0, 2.0),
		NewGeo("Catania", 3.0, 4.0),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)

	exceptPoints := make([]*Point, len(geos)+1)
	for idx, geo := range geos {
		_geo := geo
		exceptPoints[idx] = &_geo.Point
	}
	points, err := c.GeoPos("gk1", "Agrigento", "Catania", "Foo")
	assert.NoError(t, err)
	for idx, point := range points {
		assert.True(t, PointEqual(exceptPoints[idx], point))
	}
}

func TestGeoRadius(t *testing.T) {
	testGeoRadius(t)
}

func testGeoRadius(t *testing.T) {
	defer c.Del("gk1")
	_, err := c.GeoAdd("gk1",
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	)
	assert.NoError(t, err)
	members, err := c.GeoRadius("gk1", Point{15, 37}, 200, "km")
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"Palermo", "Catania"}, members)
}

func TestGeoRadiusWithDist(t *testing.T) {
	testGeoRadiusWithDist(t)
}

func testGeoRadiusWithDist(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	except := []DistWithName{
		{"Palermo", 190.4424},
		{"Catania", 56.4413},
	}
	members, err := c.GeoRadiusWithDist("gk1", Point{15, 37}, 200, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, DistWithNameEqual(except[idx], member))
	}
}

func TestGeoRadiusWithCoord(t *testing.T) {
	testGeoRadiusWithCoord(t)
}

func testGeoRadiusWithCoord(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	members, err := c.GeoRadiusWithCoord("gk1", Point{15, 37}, 200, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, GeoEqual(geos[idx], member))
	}
}

func TestGeoRadiusWithDistAndCoord(t *testing.T) {
	testGeoRadiusWithDistAndCoord(t)
}

func testGeoRadiusWithDistAndCoord(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	except := []DistWithGeo{
		{geos[0], 190.4424},
		{geos[1], 56.4413},
	}
	members, err := c.GeoRadiusWithDistAndCoord("gk1", Point{15, 37}, 200, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, DistWithGeoEqual(except[idx], member))
	}
}

func TestGeoRadiusByMember(t *testing.T) {
	testGeoRadiusByMember(t)
}

func testGeoRadiusByMember(t *testing.T) {
	defer c.Del("gk1")
	_, err := c.GeoAdd("gk1",
		NewGeo("Agrigento", 13.583333, 37.316667),
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	)
	assert.NoError(t, err)
	members, err := c.GeoRadiusByMember("gk1", "Agrigento", 100, "km")
	assert.NoError(t, err)
	assert.Equal(t, []string{"Agrigento", "Palermo"}, members)
}

func TestGeoRadiusByMemberWithDist(t *testing.T) {
	testGeoRadiusByMemberWithDist(t)
}

func testGeoRadiusByMemberWithDist(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Agrigento", 13.583333, 37.316667),
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	except := []DistWithName{
		{"Agrigento", 0.0},
		{"Palermo", 90.9778},
	}
	members, err := c.GeoRadiusByMemberGeoRadiusWithDist("gk1", "Agrigento", 100, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, DistWithNameEqual(except[idx], member))
	}
}

func TestGeoRadiusByMemberWithCoord(t *testing.T) {
	testGeoRadiusByMemberWithCoord(t)
}

func testGeoRadiusByMemberWithCoord(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Agrigento", 13.583333, 37.316667),
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	members, err := c.GeoRadiusByMemberGeoRadiusWithCoord("gk1", "Agrigento", 100, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, GeoEqual(geos[:2][idx], member))
	}
}

func TestGeoRadiusByMemberWithDistAndCoord(t *testing.T) {
	testGeoRadiusByMemberWithDistAndCoord(t)
}

func testGeoRadiusByMemberWithDistAndCoord(t *testing.T) {
	defer c.Del("gk1")
	geos := []Geo{
		NewGeo("Agrigento", 13.583333, 37.316667),
		NewGeo("Palermo", 13.361389, 38.115556),
		NewGeo("Catania", 15.087269, 37.502669),
	}
	_, err := c.GeoAdd("gk1", geos...)
	assert.NoError(t, err)
	assert.NoError(t, err)
	except := []DistWithGeo{
		{geos[0], 0.0},
		{geos[1], 90.9778},
	}
	members, err := c.GeoRadiusByMemberGeoRadiusWithDistAndCoord("gk1", "Agrigento", 100, "km")
	assert.NoError(t, err)
	for idx, member := range members {
		assert.True(t, DistWithGeoEqual(except[idx], member))
	}
}

func TestEval(t *testing.T) {
	testEval(t)
}

func testEval(t *testing.T) {
	defer c.Del("key1")

	lua := "return {KEYS[1], ARGV[1]}"
	vals, err := c.Eval(lua, []string{"key1"}, "first")
	if inCpsMode {
		assert.Error(t, err)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{[]byte("key1"), []byte("first")}, vals)
}

func TestAuth(t *testing.T) {
	testAuth(t)
}

func testAuth(t *testing.T) {
	_, err := c.Auth("123")
	assert.Error(t, err)
}

func TestPing(t *testing.T) {
	testPing(t)
}

func testPing(t *testing.T) {
	result, err := c.Ping()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", result)
}

func TestQuit(t *testing.T) {
	testQuit(t)
}

func testQuit(t *testing.T) {
	result, err := c.Quit()
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)
}

func TestInfo(t *testing.T) {
	testInfo(t)
}

func testInfo(t *testing.T) {
	_, err := c.Info()
	assert.NoError(t, err)
}

func TestTime(t *testing.T) {
	testTime(t)
}

func testTime(t *testing.T) {
	_, err := c.Time()
	assert.NoError(t, err)
}

func TestLargeValue(t *testing.T) {
	testLargeValue(t)
}

func testLargeValue(t *testing.T) {
	defer c.Del("hello")

	count := 1024 * 1024 * 100
	result, err := c.Set("hello", strings.Repeat("h", count))
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, count, len(val))
}

func TestMoved(t *testing.T) {
	testMoved(t)
}

func testMoved(t *testing.T) {
	defer c.Del("hello")

	result, err := c.Set("hello", 123)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	// TODO: fix hardcode
	var (
		src        = "127.0.0.1:8000"
		srcAddr, _ = net.ResolveTCPAddr("tcp4", src)
		dst        = "127.0.0.1:8001"
		dstAddr, _ = net.ResolveTCPAddr("tcp4", dst)
		slot       = 866 // the slot of 'hello' is 866
	)

	srcID, err := defaultClusterManager.NodeID(srcAddr)
	if err != nil {
		t.Error(err)
		return
	}

	dstID, err := defaultClusterManager.NodeID(dstAddr)
	if err != nil {
		t.Error(err)
		return
	}
	// migrate slot[866] from 127.0.0.1:8000 to 127.0.0.1:8001
	c1, err := newClientWithAuth("tcp", src)
	if err != nil {
		t.Error(err)
		return
	}
	c2, err := newClientWithAuth("tcp", dst)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = c1.Exec("CLUSTER", "SETSLOT", slot, "MIGRATING", dstID)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = c2.Exec("CLUSTER", "SETSLOT", slot, "IMPORTING", srcID)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		c1.Exec("CLUSTER", "SETSLOT", slot, "STABLE")
		c2.Exec("CLUSTER", "SETSLOT", slot, "STABLE")
	}()

	val, err := c.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)

	// double check
	time.Sleep(time.Second)
	val, err = c.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)
}

func TestAsk(t *testing.T) {
	testAsk(t)
}

func testAsk(t *testing.T) {
	result, err := c.Set("hello", "world")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)
	defer c.Del("hello")

	// TODO: fix hardcode
	var (
		src        = "127.0.0.1:8000"
		srcAddr, _ = net.ResolveTCPAddr("tcp4", src)
		dst        = "127.0.0.1:8001"
		dstAddr, _ = net.ResolveTCPAddr("tcp4", dst)
		slot       = 866 // the slot of 'hello' is 866
	)

	srcID, err := defaultClusterManager.NodeID(srcAddr)
	if err != nil {
		t.Error(err)
		return
	}

	dstID, err := defaultClusterManager.NodeID(dstAddr)
	if err != nil {
		t.Error(err)
		return
	}

	c1, err := newClientWithAuth("tcp", src)
	if err != nil {
		t.Error(err)
		return
	}
	c2, err := newClientWithAuth("tcp", dst)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = c1.Exec("CLUSTER", "SETSLOT", slot, "MIGRATING", dstID)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = c2.Exec("CLUSTER", "SETSLOT", slot, "IMPORTING", srcID)
	if err != nil {
		t.Error(err)
		return
	}

	result, err = c.Set("a{hello}", "world")
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	val, err := c.Get("a{hello}")
	assert.NoError(t, err)
	assert.Equal(t, "world", val)

	c1.Exec("CLUSTER", "SETSLOT", slot, "STABLE")
	c2.Exec("CLUSTER", "SETSLOT", slot, "STABLE")

	val, err = c.Get("a{hello}")
	assert.Equal(t, err, ErrNil)
	assert.Equal(t, "", val)
}

func TestDelNode(t *testing.T) {
	testDelNode(t)
}

func testDelNode(t *testing.T) {
	defer c.Del("hello")

	result, err := c.Set("hello", 123)
	assert.NoError(t, err)
	assert.Equal(t, "OK", result)

	// TODO: fix hardcode
	slot := 866
	src := "127.0.0.1:8000"
	srcAddr, _ := net.ResolveTCPAddr("tcp4", src)
	dst := "127.0.0.1:8003"
	dstAddr, _ := net.ResolveTCPAddr("tcp4", dst)

	// add node[127.0.0.1:8003] to redis cluster
	if err := defaultClusterManager.AddMaster(dstAddr); err != nil {
		t.Error(err)
		return
	}
	// migrate slot[866] from 127.0.0.1:8000 to 127.0.0.1:8003
	if err := defaultClusterManager.ReShard(srcAddr, dstAddr, slot); err != nil {
		t.Error(err)
		return
	}

	val, err := c.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)

	// migrate slot[866] from 127.0.0.1:8003 to 127.0.0.1:8000
	if err := defaultClusterManager.ReShard(dstAddr, srcAddr, slot); err != nil {
		t.Error(err)
		return
	}
	// delete node[127.0.0.1:8003] from redis cluster
	if err := defaultClusterManager.DelNode(dstAddr); err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second * 3)
	val, err = c.Get("hello")
	assert.NoError(t, err)
	assert.Equal(t, "123", val)
}

func TestPipeline(t *testing.T) {
	testPipeline(t)
}

func testPipeline(t *testing.T) {
	defer c.Del("hello")

	for i := 0; i < 15; i++ {
		if err := c.Send("SET", "hello", strings.Repeat("x", 1024*1024*17)); err != nil {
			t.Error(err)
			return
		}
		if err := c.Send("GET", "hello"); err != nil {
			t.Error(err)
		}
	}
	if err := c.Flush(); err != nil {
		t.Error(err)
		return
	}
	_, err := c.Receive()
	assert.NoError(t, err)
}
