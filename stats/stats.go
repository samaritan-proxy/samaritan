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

package stats

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/kirk91/stats"

	"github.com/samaritan-proxy/samaritan/logger"
)

// alias
type (
	Sink                = stats.Sink
	Scope               = stats.Scope
	Counter             = stats.Counter
	Gauge               = stats.Gauge
	Tag                 = stats.Tag
	Histogram           = stats.Histogram
	HistogramStatistics = stats.HistogramStatistics
)

var (
	defaultStore *stats.Store

	defaultTags          map[string]string
	tagExtractStrategies []stats.TagExtractStrategy
)

func init() {
	defaultStore = stats.NewStore(stats.NewStoreOption())

	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}
	defaultTags = map[string]string{"hostname": hostname}
	tagExtractStrategies = []stats.TagExtractStrategy{
		{
			// service.(<service_name>.)*
			Name:  "service_name",
			Regex: "^service\\.((.*?)\\.)",
		},
		{
			// service.[<service_name>.]redis.(<redis_cmd>.)<base_stat>
			Name:  "redis_cmd",
			Regex: "^service(?:\\.).*?\\.redis\\.((.*?)\\.)",
		},
	}
}

// Init inits the stats.
func Init(sinks ...Sink) error {
	// set tag options
	tagOption := stats.NewTagOption().WithDefaultTags(defaultTags).
		WithTagExtractStrategies(tagExtractStrategies...)
	if err := defaultStore.SetTagOption(tagOption); err != nil {
		return err
	}

	// add sinks
	for _, sink := range sinks {
		defaultStore.AddSink(sink)
	}

	go defaultStore.FlushingLoop(context.Background())
	go collectRuntime(context.Background())
	return nil
}

func collectRuntime(ctx context.Context) {
	scope := CreateScope("runtime.")
	defer DeleteScope(scope)

	runtimeCollectInterval := time.Second * 4
	ticker := time.NewTicker(runtimeCollectInterval)
	defer ticker.Stop()

	var lastNumGC uint32
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		numGC := memStats.NumGC
		if numGC < lastNumGC {
			lastNumGC = 0
		}
		scope.Counter("gc_total").Add(uint64(numGC - lastNumGC))

		if numGC-lastNumGC >= 256 {
			lastNumGC = numGC - 255
		}
		for i := lastNumGC; i < numGC; i++ {
			pause := memStats.PauseNs[i%256]
			scope.Histogram("gc_pause_us").Record(pause / uint64(time.Microsecond))
		}
		lastNumGC = numGC

		scope.Gauge("goroutines").Set(uint64(runtime.NumGoroutine()))
		scope.Gauge("gc_pause_total_ms").Set(memStats.PauseTotalNs / uint64(time.Millisecond))
		scope.Gauge("alloc_bytes").Set(memStats.Alloc)
		scope.Gauge("total_alloc_bytes").Set(memStats.TotalAlloc)
		scope.Gauge("sys_bytes").Set(memStats.Sys)
		scope.Gauge("heap_alloc_bytes").Set(memStats.HeapAlloc)
		scope.Gauge("heap_sys_bytes").Set(memStats.HeapSys)
		scope.Gauge("heap_idle_bytes").Set(memStats.HeapIdle)
		scope.Gauge("heap_inuse_bytes").Set(memStats.HeapInuse)
		scope.Gauge("heap_released_bytes").Set(memStats.HeapReleased)
		scope.Gauge("heap_objects").Set(memStats.HeapObjects)
		scope.Gauge("stack_inuse_bytes").Set(memStats.StackInuse)
		scope.Gauge("stack_sys_bytes").Set(memStats.StackSys)
		scope.Gauge("lookups").Set(memStats.Lookups)
		scope.Gauge("mallocs").Set(memStats.Mallocs)
		scope.Gauge("frees").Set(memStats.Frees)
	}
}

// CreateScope creates a named scope.
func CreateScope(name string) *Scope {
	return defaultStore.CreateScope(name)
}

// DeleteScope deletes the named scope.
func DeleteScope(scope *Scope) {
	defaultStore.DeleteScope(scope)
}

// Scopes returns all known scopes.
func Scopes() []*Scope {
	return defaultStore.Scopes()
}

// Counters returns all known counters.
func Counters() []*Counter {
	return defaultStore.Counters()
}

// Gauges returns all known gauges.
func Gauges() []*Gauge {
	return defaultStore.Gauges()
}

// Histograms returns all known histograms.
func Histograms() []*Histogram {
	return defaultStore.Histograms()
}

// NewCounter creates a counter with given params.
// NOTE: It should only be used in unit tests.
func NewCounter(name, tagExtractedName string, tags []*Tag) *Counter {
	return stats.NewCounter(name, tagExtractedName, tags)
}

// NewGauge creates a gauge with given params.
// NOTE: It should only be used in unit tests.
func NewGauge(name, tagExtractedName string, tags []*Tag) *Gauge {
	return stats.NewGauge(name, tagExtractedName, tags)
}

// NewHistogram creates a histogram with given params.
// NOTE: It should only be used in unit tests.
func NewHistogram(name, tagExtractedName string, tags []*Tag) *Histogram {
	return stats.NewHistogram(nil, name, tagExtractedName, tags)
}
