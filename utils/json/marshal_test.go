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

package json

import (
	"testing"

	pbtest "github.com/gogo/protobuf/test/example"
	"github.com/stretchr/testify/assert"
)

func TestMarshalProtoMsg(t *testing.T) {
	b, err := Marshal(&pbtest.A{
		Description: "hello protobuf",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"Description": "hello protobuf"}`, string(b))
}

func TestMarshal(t *testing.T) {
	s := struct {
		A string `json:"a"`
	}{
		A: "hello",
	}
	b, err := Marshal(s)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"a": "hello"}`, string(b))
}

func TestMarshalIndent(t *testing.T) {
	s := struct {
		String string `json:"string"`
		Int    int    `json:"int"`
	}{
		String: "foo",
		Int:    100,
	}
	b, err := MarshalIndent(s, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, `{
  "string": "foo",
  "int": 100
}`, string(b))
}
