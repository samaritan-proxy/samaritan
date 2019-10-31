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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/config"
)

func TestHandleGetConfig(t *testing.T) {
	c, err := config.New(&bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "0.0.0.0",
				Port: 1000,
			},
		},
	})
	assert.NoError(t, err)
	var (
		s = New(c)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "http://1.2.3.4", nil)
	)
	s.handleGetConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
