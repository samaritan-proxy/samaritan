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

	"github.com/stretchr/testify/assert"
)

func TestSliceAlloc(t *testing.T) {
	sa := &sliceAlloc{}
	sa.Make(100)
	assert.Equal(t, 1, sa.allocs)
	sa.Make(1024)
	assert.Equal(t, 2, sa.allocs)
	for i := 0; i < 100; i++ {
		sa.Make(100)
	}
	assert.Equal(t, 3, sa.allocs)
}
