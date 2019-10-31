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
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/samaritan-proxy/samaritan/stats"
)

const statsTagName = "stats"

var (
	scopeType     = reflect.TypeOf(new(stats.Scope))
	counterType   = reflect.TypeOf(new(stats.Counter))
	gaugeType     = reflect.TypeOf(new(stats.Gauge))
	histogramType = reflect.TypeOf(new(stats.Histogram))
)

func toUnderscore(s string) string {
	// convert aAa to a_Aa
	r, _ := regexp.Compile(`(\w)([A-Z][a-z])`)
	s = r.ReplaceAllString(s, "${1}_${2}")
	// convert aBB to a_BB
	r, _ = regexp.Compile(`([a-z])([A-Z]+)`)
	s = r.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}

// BuildStats builds the stats with given scope. It will
// initialize the Scope, Counter, Gauge, Histogram fields.
//
// Exmaples of struct fields and their initialized value:
//
//   // CxTotal value will be *scope.Counter("cx_total")
//	 CxTotal *stats.Counter
//
//   // CxActive value will be *scope.Gauge("cx_active")
//   CxActive *stats.Gauge
//
//   // CxDuration value will be *scope.Histogram("cx_duration")
//   CxDuration *stats.Histogram
//
func BuildStats(scope *stats.Scope, stats interface{}) error {
	v := reflect.ValueOf(stats)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("non-pointer %s", v.Type().String())
	}
	if v.IsNil() {
		return fmt.Errorf("nil %s", v.Type().String())
	}

	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("non-struct %s", v.Kind())
	}

	n := v.NumField()
	for i := 0; i < n; i++ {
		f := v.Field(i)
		fName := v.Type().Field(i).Name
		if alias := v.Type().Field(i).Tag.Get(statsTagName); alias != "" {
			fName = alias
		}
		var fv interface{}
		switch f.Type() {
		case scopeType:
			fv = scope
		case counterType:
			fv = scope.Counter(toUnderscore(fName))
		case gaugeType:
			fv = scope.Gauge(toUnderscore(fName))
		case histogramType:
			fv = scope.Histogram(toUnderscore(fName))
		}

		if fv != nil && f.CanSet() {
			f.Set(reflect.ValueOf(fv))
		}
	}
	return nil
}
