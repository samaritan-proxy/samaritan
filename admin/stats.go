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
	"regexp"
	"sort"
	"strings"

	"github.com/samaritan-proxy/samaritan/stats"
)

var (
	_ statsFormater = new(plainStatsFormatter)
	_ statsFormater = new(prometheusStatsFormatter)
)

type statsFormater interface {
	Format([]*stats.Gauge, []*stats.Counter, []*stats.Histogram) []byte
}

type plainStatsFormatter struct{}

func newPlainStatsFormatter() *plainStatsFormatter {
	return new(plainStatsFormatter)
}

func (f *plainStatsFormatter) Format(gauges []*stats.Gauge, counters []*stats.Counter, histograms []*stats.Histogram) []byte {
	metricNames := make([]string, 0)
	metrics := make(map[string]interface{})
	recordMetric := func(name string, value interface{}) {
		metricNames = append(metricNames, name)
		metrics[name] = value
	}

	for _, gauge := range gauges {
		recordMetric(gauge.Name(), gauge.Value())
	}
	for _, counter := range counters {
		recordMetric(counter.Name(), counter.Value())
	}
	for _, histogram := range histograms {
		recordMetric(histogram.Name(), histogram.Summary())
	}

	sort.Strings(metricNames) // alphabet order
	var buf bytes.Buffer
	for _, name := range metricNames {
		buf.WriteString(fmt.Sprintf("%s: %v\n", name, metrics[name]))
	}
	return buf.Bytes()
}

type prometheusStatsFormatter struct {
	metricTypes map[string]struct{}
}

func newPrometheusStatsFormatter() *prometheusStatsFormatter {
	return &prometheusStatsFormatter{
		metricTypes: make(map[string]struct{}),
	}
}

// Format formats the metrics to a text-based format which prometheus accpets.
// Refer to https://prometheus.io/docs/instrumenting/exposition_formats/
func (f *prometheusStatsFormatter) Format(gauges []*stats.Gauge, counters []*stats.Counter, histograms []*stats.Histogram) []byte {
	buf := new(bytes.Buffer)

	for _, gauge := range gauges {
		f.formatGauge(buf, gauge)
	}
	for _, counter := range counters {
		f.formatCounter(buf, counter)
	}
	for _, histogram := range histograms {
		f.formatHistogram(buf, histogram)
	}

	return buf.Bytes()
}

func (f *prometheusStatsFormatter) formatCounter(buf *bytes.Buffer, c *stats.Counter) {
	name := f.formatMeticName(c.TagExtractedName())
	value := c.Value()
	tags := f.formatTags(c.Tags())
	if f.recordMetricType(name) {
		buf.WriteString(fmt.Sprintf("# TYPE %s counter\n", name))
	}
	buf.WriteString(fmt.Sprintf("%s{%s} %d\n", name, tags, value))
}

func (f *prometheusStatsFormatter) formatGauge(buf *bytes.Buffer, g *stats.Gauge) {
	name := f.formatMeticName(g.TagExtractedName())
	value := g.Value()
	tags := f.formatTags(g.Tags())
	if f.recordMetricType(name) {
		buf.WriteString(fmt.Sprintf("# TYPE %s gauge\n", name))
	}
	buf.WriteString(fmt.Sprintf("%s{%s} %d\n", name, tags, value))
}

func (f *prometheusStatsFormatter) formatHistogram(buf *bytes.Buffer, h *stats.Histogram) {
	name := f.formatMeticName(h.TagExtractedName())
	tags := f.formatTags(h.Tags())
	if f.recordMetricType(name) {
		buf.WriteString(fmt.Sprintf("# TYPE %s histogram\n", name))
	}
	hStats := h.CumulativeStatistics()
	f.formatHistogramValue(buf, name, tags, hStats)
}

func (f *prometheusStatsFormatter) formatHistogramValue(buf *bytes.Buffer, name, tags string, hStats *stats.HistogramStatistics) {
	sbs := hStats.SupportedBuckets()
	cbs := hStats.ComputedBuckets()
	for i := 0; i < len(sbs); i++ {
		b := sbs[i]
		v := cbs[i]
		bucketTags := fmt.Sprintf("%s,le=\"%.32g\"", tags, b)
		// trim the comma prefix when tags is empty
		bucketTags = strings.TrimPrefix(bucketTags, ",")
		buf.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", name, bucketTags, v))
	}
	bucketTags := strings.TrimPrefix(fmt.Sprintf("%s,le=\"+Inf\"", tags), ",")
	buf.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", name, bucketTags, hStats.SampleCount()))
	buf.WriteString(fmt.Sprintf("%s_sum{%s} %.32g\n", name, tags, hStats.SampleSum()))
	buf.WriteString(fmt.Sprintf("%s_count{%s} %d\n", name, tags, hStats.SampleCount()))
}

func (f *prometheusStatsFormatter) recordMetricType(metricName string) bool {
	if _, ok := f.metricTypes[metricName]; ok {
		return false
	}
	f.metricTypes[metricName] = struct{}{}
	return true
}

func (f *prometheusStatsFormatter) formatMeticName(name string) string {
	// A metric name should have a (single-word) application prefix relevant to
	// the domain the metric belongs to.
	// Refer to https://prometheus.io/docs/practices/naming/#metric-names
	return f.sanitizeName(fmt.Sprintf("samaritan_%s", name))
}

func (f *prometheusStatsFormatter) formatTags(tags []*stats.Tag) string {
	res := make([]string, len(tags))
	for i, tag := range tags {
		res[i] = fmt.Sprintf("%s=\"%s\"", f.sanitizeName(tag.Name), tag.Value)
	}
	return strings.Join(res, ",")
}

func (f *prometheusStatsFormatter) sanitizeName(name string) string {
	// The name must match the regex [a-zA-Z_][a-zA-Z0-9_]* as required by
	// prometheus. Refer to https://prometheus.io/docs/concepts/data_model/
	re := regexp.MustCompile("[^a-zA-Z0-9_]")
	return re.ReplaceAllString(name, "_")
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	f := newPlainStatsFormatter()
	b := f.Format(stats.Gauges(), stats.Counters(), stats.Histograms())
	w.Write(b)
}

func (s *Server) handleGetStatsForPrometheus(w http.ResponseWriter, r *http.Request) {
	f := newPrometheusStatsFormatter()
	b := f.Format(stats.Gauges(), stats.Counters(), stats.Histograms())
	w.Write(b)
}
