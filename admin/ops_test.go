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
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleShutdown(t *testing.T) {
	killFunc = func(pid int, sig syscall.Signal) (err error) {
		return nil
	}
	getPidFunc = func() int {
		return 0
	}

	var (
		s = new(Server)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "http://1.2.3.4", nil)
	)
	s.handleShutdown(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	b, err := ioutil.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"msg": "OK"}`, string(b))
}

func TestHandleShutdownWithError(t *testing.T) {
	killFunc = func(pid int, sig syscall.Signal) (err error) {
		return errors.New("error")
	}
	getPidFunc = func() int {
		return 0
	}
	var (
		s = new(Server)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "http://1.2.3.4", nil)
	)
	s.handleShutdown(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
