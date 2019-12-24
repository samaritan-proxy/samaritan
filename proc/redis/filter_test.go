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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestFilterChain_AppendFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	filter1 := NewMockFilter(ctrl)
	filter2 := NewMockFilter(ctrl)
	chain := newRequestFilterChain()

	chain.AddFilter(filter1)
	chain.AddFilter(filter2)
	assert.EqualValues(t, filter1, chain.filters[0])
	assert.EqualValues(t, filter2, chain.filters[1])
}

func TestFilterChain_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := newRequestFilterChain()
	filters := make([]Filter, 3)
	for idx := range filters {
		filter := NewMockFilter(ctrl)
		filter.EXPECT().Destroy()
		filters[idx] = filter
		chain.AddFilter(filter)
	}
	chain.Reset()
	assert.Equal(t, 0, len(chain.filters))
}

func TestFilterChain_Do(t *testing.T) {
	req := newSimpleRequest(newStringArray("get", "a"))

	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// two filters
		filter1 := NewMockFilter(ctrl)
		filter1.EXPECT().Do(gomock.Any(), gomock.Any()).Return(Continue)
		filter2 := NewMockFilter(ctrl)
		filter2.EXPECT().Do(gomock.Any(), gomock.Any()).Return(Continue)

		// register filters
		chain := newRequestFilterChain()
		chain.AddFilter(filter1)
		chain.AddFilter(filter2)

		status := chain.Do(req)
		assert.Equal(t, Continue, status)
	})

	t.Run("termination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// two filters
		filter1 := NewMockFilter(ctrl)
		filter1.EXPECT().Do(gomock.Any(), gomock.Any()).Return(Stop)
		filter2 := NewMockFilter(ctrl)
		filter2.EXPECT().Do(gomock.Any(), gomock.Any()).Return(Continue).Times(0)

		// register filters
		chain := newRequestFilterChain()
		chain.AddFilter(filter1)
		chain.AddFilter(filter2)

		status := chain.Do(req)
		assert.Equal(t, Stop, status)
	})
}
