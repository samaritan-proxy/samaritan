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
	"strconv"
)

type RespType byte

const (
	SimpleString RespType = '+'
	Error        RespType = '-'
	Integer      RespType = ':'
	BulkString   RespType = '$'
	Array        RespType = '*'
)

type RespValue struct {
	Type RespType

	Int   int64
	Text  []byte
	Array []RespValue
}

var (
	invalidRequest = "invalid request"
)

func (r *RespValue) String() string {
	if r == nil {
		return ""
	}
	switch r.Type {
	case SimpleString, Error, BulkString:
		return string(r.Text)
	case Integer:
		return strconv.Itoa(int(r.Int))
	case Array:
		buf := bytes.NewBuffer(nil)
		for idx, v := range r.Array {
			buf.WriteString(v.String())
			if idx != len(r.Array)-1 {
				buf.WriteString(" ")
			}
		}
		return buf.String()
	default:
		return ""
	}
}

func (r *RespValue) Equal(that *RespValue) bool {
	if that == nil {
		return r == nil
	}
	if that.Type != r.Type {
		return false
	}
	switch r.Type {
	case Integer:
		return that.Int == r.Int
	case Array:
		if len(that.Array) != len(r.Array) {
			return false
		}
		for idx, val := range r.Array {
			if !val.Equal(&that.Array[idx]) {
				return false
			}
		}
	default:
		return bytes.Equal(that.Text, r.Text)
	}
	return true
}

func newError(s string) *RespValue {
	return &RespValue{
		Type: Error,
		Text: []byte(s),
	}
}

func newSimpleString(s string) *RespValue {
	return &RespValue{
		Type: SimpleString,
		Text: []byte(s),
	}
}

func newSimpleBytes(b []byte) *RespValue {
	return &RespValue{
		Type: SimpleString,
		Text: b,
	}
}

func newBulkString(s string) *RespValue {
	return &RespValue{
		Type: BulkString,
		Text: []byte(s),
	}
}

func newBulkBytes(b []byte) *RespValue {
	return &RespValue{
		Type: BulkString,
		Text: b,
	}
}

func newNullBulkString() *RespValue {
	return &RespValue{Type: BulkString}
}

func newInteger(i int64) *RespValue {
	return &RespValue{
		Type: Integer,
		Int:  i,
	}
}

func newArray(array ...RespValue) *RespValue {
	return &RespValue{
		Type:  Array,
		Array: array,
	}
}

func newStringArray(str ...string) *RespValue {
	arr := make([]RespValue, len(str))
	for idx, s := range str {
		arr[idx] = *newBulkString(s)
	}
	return newArray(arr...)
}

func newByteArray(b ...[]byte) *RespValue {
	arr := make([]RespValue, len(b))
	for idx, s := range b {
		arr[idx] = *newBulkBytes(s)
	}
	return newArray(arr...)
}
