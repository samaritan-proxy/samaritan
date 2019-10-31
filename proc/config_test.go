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

package proc

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

func TestSetDefaultValue(t *testing.T) {
	cases := []struct {
		Input, Except *service.Config
	}{
		{nil, nil},
		{
			new(service.Config),
			&service.Config{
				ConnectTimeout: utils.DurationPtr(defaultConnectTimeout),
				IdleTimeout:    utils.DurationPtr(defaultIdleTimeout),
			},
		},
		{
			&service.Config{
				IdleTimeout: utils.DurationPtr(defaultIdleTimeout),
			},
			&service.Config{
				ConnectTimeout: utils.DurationPtr(defaultConnectTimeout),
				IdleTimeout:    utils.DurationPtr(defaultIdleTimeout),
			},
		},
		{
			&service.Config{
				ConnectTimeout: utils.DurationPtr(defaultConnectTimeout),
			},
			&service.Config{
				ConnectTimeout: utils.DurationPtr(defaultConnectTimeout),
				IdleTimeout:    utils.DurationPtr(defaultIdleTimeout),
			},
		},
		{
			&service.Config{
				ConnectTimeout: utils.DurationPtr(100 * time.Second),
				IdleTimeout:    utils.DurationPtr(200 * time.Second),
			},
			&service.Config{
				ConnectTimeout: utils.DurationPtr(100 * time.Second),
				IdleTimeout:    utils.DurationPtr(200 * time.Second),
			},
		},
	}
	for idx, c := range cases {
		t.Run(fmt.Sprintf("case:%d", idx+1), func(t *testing.T) {
			setDefaultValue(c.Input)
			assert.True(t, c.Except.Equal(c.Input))
		})
	}
}
