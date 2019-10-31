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

	assert.NoError(t, chain.AddFilter(filter1))
	assert.EqualValues(t, filter1, chain.filtersList[0])

	assert.NoError(t, chain.AddFilter(filter2))
	assert.EqualValues(t, filter1, chain.filtersList[1])

	assert.Equal(t, ErrFilterIsNull, chain.AddFilter(nil))
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
		assert.NoError(t, chain.AddFilter(filter))
	}
	chain.Reset()
	assert.Equal(t, 0, len(chain.filtersList))
}

func TestFilterChain_ParseRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := newRequestFilterChain()

	names := make([]Filter, 5)
	res := make([]Filter, 0)
	for idx := range names {
		filter := NewMockFilter(ctrl)
		filter.EXPECT().Do(gomock.Any()).Return(Continue).Do(func(_ interface{}) {
			res = append(res, filter)
		})
		names[idx] = filter
		assert.NoError(t, chain.AddFilter(filter))
	}
	filter := NewMockFilter(ctrl)
	filter.EXPECT().Do(gomock.Any()).Return(Stop)
	assert.NoError(t, chain.AddFilter(filter))
	assert.NoError(t, chain.AddFilter(NewMockFilter(ctrl)))
	chain.Do(nil)
	assert.Equal(t, names, res)
}
