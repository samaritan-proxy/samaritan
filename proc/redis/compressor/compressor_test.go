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

package compressor

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
)

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := NewMockCompressor(ctrl)
	Register(protocol.MOCK.String(), c)
	defer UnRegister(protocol.MOCK.String())

	cps, ok := get(protocol.MOCK.String())
	assert.True(t, ok)
	assert.Equal(t, c, cps)

	assert.Panics(t, func() {
		Register(protocol.MOCK.String(), c)
	})
}
