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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/stats"
)

func TestBuildStats(t *testing.T) {
	var (
		x int
		y *int
	)
	assert.Error(t, BuildStats(nil, x))  // not pointer
	assert.Error(t, BuildStats(nil, &x)) // not struct pointer
	assert.Error(t, BuildStats(nil, y))  // nil pointer

	v := struct {
		*stats.Scope
		CxTotal           *stats.Counter
		CxActive          *stats.Gauge
		CxLength          *stats.Histogram
		cxLength          *stats.Histogram
		DownstreamCxTotal *stats.Counter `stats:"downstream.cx_total"`
		Foo               *int
		Bar               *string
	}{}
	scope := stats.CreateScope("")
	defer stats.DeleteScope(scope)
	assert.NoError(t, BuildStats(scope, &v))
	assert.Equal(t, v.Scope, scope)
	assert.Equal(t, v.CxTotal.Name(), "cx_total")
	assert.Equal(t, v.CxActive.Name(), "cx_active")
	assert.Equal(t, v.CxLength.Name(), "cx_length")
	assert.Equal(t, v.DownstreamCxTotal.Name(), "downstream.cx_total")
	assert.Equal(t, v.cxLength, (*stats.Histogram)(nil))
	assert.Equal(t, v.Foo, (*int)(nil))
	assert.Equal(t, v.Bar, (*string)(nil))
}
