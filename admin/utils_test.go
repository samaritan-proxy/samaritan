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

package admin

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetContent(t *testing.T) {
	w := httptest.NewRecorder()
	setContent(w, ContentApplicationJSON)
	assert.Equal(t, ContentApplicationJSON, w.Header().Get("Content-Type"))
}

func TestWriteMessage(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoError(t, writeMessage(w, 200, "foo"))
	assert.Equal(t, ContentApplicationJSON, w.Header().Get("Content-Type"))
	assert.Equal(t, 200, w.Code)
	b, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"msg": "foo"}`, string(b))
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	s := struct {
		Str string `json:"str"`
		Int int    `json:"int"`
	}{
		Str: "foo",
		Int: 100,
	}
	assert.NoError(t, writeJSON(w, s))
	assert.Equal(t, ContentApplicationJSON, w.Header().Get("Content-Type"))
	b, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"str": "foo", "int": 100}`, string(b))
}

type marshalWithErr struct{}

func (*marshalWithErr) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

func TestWriteJSONWithError(t *testing.T) {
	w := httptest.NewRecorder()
	assert.Error(t, writeJSON(w, new(marshalWithErr)))
	assert.Equal(t, ContentApplicationJSON, w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	b, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(b), `marshal error`, )
}
