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
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/stats"
)

func TestPlainStatsFormatter(t *testing.T) {
	scope := stats.CreateScope("plain-stats")
	defer stats.DeleteScope(scope)

	g1 := scope.Gauge("g1")
	g1.Inc()
	c1 := scope.Counter("c1")
	c1.Inc()
	h1 := scope.Histogram("h1")
	h1.Record(1)

	f := newPlainStatsFormatter()
	res := f.Format(scope.Gauges(), scope.Counters(), scope.Histograms())

	var expect bytes.Buffer
	expect.WriteString(fmt.Sprintf("%s: %v\n", c1.Name(), c1.Value()))
	expect.WriteString(fmt.Sprintf("%s: %v\n", g1.Name(), g1.Value()))
	expect.WriteString(fmt.Sprintf("%s: %v\n", h1.Name(), h1.Summary()))
	assert.Equal(t, expect.Bytes(), res)
}

func TestFormatGaugeForPrometheus(t *testing.T) {
	g1 := stats.NewGauge("foo.sash", "foo",
		[]*stats.Tag{{Name: "tag1", Value: "sash"}})
	g2 := stats.NewGauge("foo.bos", "foo",
		[]*stats.Tag{{Name: "tag1", Value: "bos"}})
	g3 := stats.NewGauge("bar", "bar", nil)
	g1.Inc()
	g2.Set(2)

	f := newPrometheusStatsFormatter()
	res := f.Format([]*stats.Gauge{g1, g2, g3}, nil, nil)
	expect := `# TYPE samaritan_foo gauge
samaritan_foo{tag1="sash"} 1
samaritan_foo{tag1="bos"} 2
# TYPE samaritan_bar gauge
samaritan_bar{} 0
`
	assert.Equal(t, expect, string(res))
}

func TestFormatCounterForPrometheus(t *testing.T) {
	c1 := stats.NewCounter("foo.sash", "foo",
		[]*stats.Tag{{Name: "tag1", Value: "sash"}})
	c2 := stats.NewCounter("foo.bos", "foo",
		[]*stats.Tag{{Name: "tag1", Value: "bos"}})
	c3 := stats.NewCounter("bar", "bar", nil)
	c1.Inc()
	c2.Add(2)

	f := newPrometheusStatsFormatter()
	res := f.Format(nil, []*stats.Counter{c1, c2, c3}, nil)
	expect := `# TYPE samaritan_foo counter
samaritan_foo{tag1="sash"} 1
samaritan_foo{tag1="bos"} 2
# TYPE samaritan_bar counter
samaritan_bar{} 0
`
	assert.Equal(t, expect, string(res))
}

func TestFormatHistogramForPrometheus(t *testing.T) {
	t.Run("no values and tags", func(t *testing.T) {
		h := stats.NewHistogram("h", "h", nil)
		f := newPrometheusStatsFormatter()
		res := f.Format(nil, nil, []*stats.Histogram{h})
		expect := `# TYPE samaritan_h histogram
samaritan_h_bucket{le="0.5"} 0
samaritan_h_bucket{le="1"} 0
samaritan_h_bucket{le="5"} 0
samaritan_h_bucket{le="10"} 0
samaritan_h_bucket{le="25"} 0
samaritan_h_bucket{le="50"} 0
samaritan_h_bucket{le="100"} 0
samaritan_h_bucket{le="250"} 0
samaritan_h_bucket{le="500"} 0
samaritan_h_bucket{le="1000"} 0
samaritan_h_bucket{le="2500"} 0
samaritan_h_bucket{le="5000"} 0
samaritan_h_bucket{le="10000"} 0
samaritan_h_bucket{le="30000"} 0
samaritan_h_bucket{le="60000"} 0
samaritan_h_bucket{le="300000"} 0
samaritan_h_bucket{le="600000"} 0
samaritan_h_bucket{le="1800000"} 0
samaritan_h_bucket{le="3600000"} 0
samaritan_h_bucket{le="+Inf"} 0
samaritan_h_sum{} 0
samaritan_h_count{} 0
`
		assert.Equal(t, expect, string(res))
	})

	t.Run("has values and tags", func(t *testing.T) {
		h1 := stats.NewHistogram("h", "h", []*stats.Tag{{Name: "tag1", Value: "foo"}})
		h2 := stats.NewHistogram("h", "h", []*stats.Tag{{Name: "tag2", Value: "bar"}})
		h1.Record(800)
		h1.Record(8000)
		h1.RefreshIntervalStatistics()
		h2.Record(50000)
		h2.RefreshIntervalStatistics()

		f := newPrometheusStatsFormatter()
		res := f.Format(nil, nil, []*stats.Histogram{h1, h2})
		expect := `# TYPE samaritan_h histogram
samaritan_h_bucket{tag1="foo",le="0.5"} 0
samaritan_h_bucket{tag1="foo",le="1"} 0
samaritan_h_bucket{tag1="foo",le="5"} 0
samaritan_h_bucket{tag1="foo",le="10"} 0
samaritan_h_bucket{tag1="foo",le="25"} 0
samaritan_h_bucket{tag1="foo",le="50"} 0
samaritan_h_bucket{tag1="foo",le="100"} 0
samaritan_h_bucket{tag1="foo",le="250"} 0
samaritan_h_bucket{tag1="foo",le="500"} 0
samaritan_h_bucket{tag1="foo",le="1000"} 1
samaritan_h_bucket{tag1="foo",le="2500"} 1
samaritan_h_bucket{tag1="foo",le="5000"} 1
samaritan_h_bucket{tag1="foo",le="10000"} 2
samaritan_h_bucket{tag1="foo",le="30000"} 2
samaritan_h_bucket{tag1="foo",le="60000"} 2
samaritan_h_bucket{tag1="foo",le="300000"} 2
samaritan_h_bucket{tag1="foo",le="600000"} 2
samaritan_h_bucket{tag1="foo",le="1800000"} 2
samaritan_h_bucket{tag1="foo",le="3600000"} 2
samaritan_h_bucket{tag1="foo",le="+Inf"} 2
samaritan_h_sum{tag1="foo"} 8855
samaritan_h_count{tag1="foo"} 2
samaritan_h_bucket{tag2="bar",le="0.5"} 0
samaritan_h_bucket{tag2="bar",le="1"} 0
samaritan_h_bucket{tag2="bar",le="5"} 0
samaritan_h_bucket{tag2="bar",le="10"} 0
samaritan_h_bucket{tag2="bar",le="25"} 0
samaritan_h_bucket{tag2="bar",le="50"} 0
samaritan_h_bucket{tag2="bar",le="100"} 0
samaritan_h_bucket{tag2="bar",le="250"} 0
samaritan_h_bucket{tag2="bar",le="500"} 0
samaritan_h_bucket{tag2="bar",le="1000"} 0
samaritan_h_bucket{tag2="bar",le="2500"} 0
samaritan_h_bucket{tag2="bar",le="5000"} 0
samaritan_h_bucket{tag2="bar",le="10000"} 0
samaritan_h_bucket{tag2="bar",le="30000"} 0
samaritan_h_bucket{tag2="bar",le="60000"} 1
samaritan_h_bucket{tag2="bar",le="300000"} 1
samaritan_h_bucket{tag2="bar",le="600000"} 1
samaritan_h_bucket{tag2="bar",le="1800000"} 1
samaritan_h_bucket{tag2="bar",le="3600000"} 1
samaritan_h_bucket{tag2="bar",le="+Inf"} 1
samaritan_h_sum{tag2="bar"} 50500
samaritan_h_count{tag2="bar"} 1
`
		assert.Equal(t, expect, string(res))
	})
}

func TestGetPlainStats(t *testing.T) {
	scope := stats.CreateScope("plain")
	defer stats.DeleteScope(scope)
	counter1 := scope.Counter("counter1")
	counter1.Inc()
	gauge1 := scope.Gauge("gauge1")
	gauge1.Inc()

	req := httptest.NewRequest(http.MethodGet, "http://1.2.3.4", nil)
	resp := httptest.NewRecorder()

	s := new(Server)
	s.handleGetStats(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), fmt.Sprintf("%s: %d", counter1.Name(), counter1.Value()))
	assert.Contains(t, resp.Body.String(), fmt.Sprintf("%s: %d", gauge1.Name(), gauge1.Value()))
}

func TestGetPrometheusStats(t *testing.T) {
	scope := stats.CreateScope("prometheus")
	defer stats.DeleteScope(scope)
	counter1 := scope.Counter("counter1")
	counter1.Inc()
	gauge1 := scope.Gauge("gauge1")
	gauge1.Inc()

	req := httptest.NewRequest(http.MethodGet, "http://1.2.3.4", nil)
	resp := httptest.NewRecorder()

	s := new(Server)
	s.handleGetStatsForPrometheus(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "# TYPE samaritan_prometheus_gauge1 gauge\n")
	assert.Contains(t, resp.Body.String(), "samaritan_prometheus_gauge1{} 1\n")
	assert.Contains(t, resp.Body.String(), "# TYPE samaritan_prometheus_counter1 counter\n")
	assert.Contains(t, resp.Body.String(), "samaritan_prometheus_counter1{} 1\n")
}
