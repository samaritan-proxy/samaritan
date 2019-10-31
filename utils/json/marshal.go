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
	"bytes"
	"encoding/json"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

// MarshalJSON returns the JSON encoding of v.
// It supports the standard go object, either the protobuf message.
func Marshal(v interface{}) ([]byte, error) {
	switch obj := v.(type) {
	case json.Marshaler:
		return obj.MarshalJSON()
	case proto.Message:
		buf := bytes.NewBuffer(nil)
		if err := new(jsonpb.Marshaler).Marshal(buf, obj); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		return json.Marshal(obj)
	}
}

// indent appends to dst an indented form of the JSON-encoded src.
func jsonIndent(b []byte, prefix, indent string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := json.Indent(buf, b, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalIndent is like Marshal but applies Indent to format the output.
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	return jsonIndent(b, prefix, indent)
}
