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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	v := newError("unknown error")
	assert.Equal(t, Error, v.Type)
	assert.NotEmpty(t, v.Text)
	assert.Equal(t, "unknown error", v.String())
}

func TestNewSimpleString(t *testing.T) {
	v := newSimpleString("ping")
	assert.Equal(t, SimpleString, v.Type)
	assert.NotEmpty(t, v.Text)
	assert.Equal(t, "ping", v.String())
}

func TestNewBulkString(t *testing.T) {
	v := newBulkString("get")
	assert.Equal(t, BulkString, v.Type)
	assert.NotEmpty(t, v.Text)
	assert.Equal(t, "get", v.String())
}

func TestNewNullBulkString(t *testing.T) {
	v := newNullBulkString()
	assert.Equal(t, BulkString, v.Type)
	assert.Nil(t, v.Text)
	assert.Equal(t, "", v.String())
}

func TestNewInteger(t *testing.T) {
	v := newInteger(10)
	assert.Equal(t, Integer, v.Type)
	assert.Equal(t, int64(10), v.Int)
	assert.Equal(t, "10", v.String())
}

func TestNewArray(t *testing.T) {
	v := newArray(
		*newBulkString("get"),
		*newBulkString("a"),
	)
	assert.Equal(t, Array, v.Type)
	assert.Equal(t, 2, len(v.Array))
	assert.Equal(t, "get a", v.String())
}

func TestNewStringArray(t *testing.T) {
	v := newStringArray("get", "a")
	assert.Equal(t, Array, v.Type)
	assert.Equal(t, 2, len(v.Array))
	assert.Equal(t, "get a", v.String())
}

func TestRespValueEqual(t *testing.T) {
	cases := []struct {
		A, B   *RespValue
		Expect bool
	}{
		{
			A:      nil,
			B:      nil,
			Expect: true,
		},
		{
			A:      newInteger(1),
			B:      nil,
			Expect: false,
		},
		{
			A:      newInteger(1),
			B:      newInteger(1),
			Expect: true,
		},
		{
			A:      newInteger(1),
			B:      newInteger(2),
			Expect: false,
		},
		{
			A:      newSimpleString("foo"),
			B:      newSimpleString("foo"),
			Expect: true,
		},
		{
			A:      newSimpleString("foo"),
			B:      newSimpleString("fo"),
			Expect: false,
		},
		{
			A:      newError("error"),
			B:      newError("error"),
			Expect: true,
		},
		{
			A:      newError("error"),
			B:      newError("err"),
			Expect: false,
		},
		{
			A:      newStringArray("SET", "FOO", "BAR"),
			B:      newStringArray("SET", "FOO", "BAR"),
			Expect: true,
		},
		{
			A:      newStringArray("SET", "FOO", "BAR"),
			B:      newStringArray("SET", "FOO", "BA"),
			Expect: false,
		},
		{
			A:      newStringArray("SET", "FOO", "BAR"),
			B:      newStringArray("SET", "FOO"),
			Expect: false,
		},
	}
	for idx, c := range cases {
		t.Run(fmt.Sprintf("case %d", idx+1), func(t *testing.T) {
			if c.A != nil && c.B != nil {
				assert.False(t, c.A == c.B)
			}
			assert.Equal(t, c.Expect, c.A.Equal(c.B))
		})
	}
}
