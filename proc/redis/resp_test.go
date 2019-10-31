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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	v := newError("unknown error")
	assert.Equal(t, Error, v.Type)
	assert.NotEmpty(t, v.Text)
}

func TestNewSimpleString(t *testing.T) {
	v := newSimpleString("ping")
	assert.Equal(t, SimpleString, v.Type)
	assert.NotEmpty(t, v.Text)
}

func TestNewBulkString(t *testing.T) {
	v := newBulkString("get")
	assert.Equal(t, BulkString, v.Type)
	assert.NotEmpty(t, v.Text)
}

func TestNewNullBulkString(t *testing.T) {
	v := newNullBulkString()
	assert.Equal(t, BulkString, v.Type)
	assert.Nil(t, v.Text)
}

func TestNewInteger(t *testing.T) {
	v := newInteger(10)
	assert.Equal(t, Integer, v.Type)
	assert.Equal(t, int64(10), v.Int)
}

func TestNewArray(t *testing.T) {
	v := newArray([]RespValue{
		*newBulkString("get"),
		*newBulkString("a"),
	})
	assert.Equal(t, Array, v.Type)
	assert.Equal(t, 2, len(v.Array))
}
