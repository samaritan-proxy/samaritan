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

func newBulkString(s string) *RespValue {
	return &RespValue{
		Type: BulkString,
		Text: []byte(s),
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

func newArray(array []RespValue) *RespValue {
	return &RespValue{
		Type:  Array,
		Array: array,
	}
}
