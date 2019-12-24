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

//go:generate mockgen -package $GOPACKAGE -self_package $REPO_URI/proc/$GOPACKAGE -destination filter_mock_test.go $REPO_URI/proc/$GOPACKAGE Filter

import (
	"bytes"
	"errors"
)

// FilterStatus type
type FilterStatus string

// FilterStatus types
const (
	Continue FilterStatus = "Continue"
	Stop     FilterStatus = "Stop"
)

var (
	ErrFilterNotFound = errors.New("error filter not found")
	ErrFilterIsNull   = errors.New("filter is null")
	ErrFilterExist    = errors.New("filter exist")
	ErrFilterNotExist = errors.New("filter not exist")
)

// Filter is used to filter the request to backend.
type Filter interface {
	// cmd is always lowercase.
	Do(cmd string, req *simpleRequest) FilterStatus
	Destroy()
}

// FilterChain is a set of filters
type FilterChain struct {
	filters []Filter
}

// newRequestFilterChain return a new FilterChain
func newRequestFilterChain() *FilterChain {
	return &FilterChain{
		filters: make([]Filter, 0, 4),
	}
}

// AddFilter adds a filter
func (c *FilterChain) AddFilter(f Filter) {
	c.filters = append(c.filters, f)
}

// Reset clean and destroy all filters
func (c *FilterChain) Reset() {
	for _, f := range c.filters {
		f.Destroy()
	}
	c.filters = make([]Filter, 0)
}

// Do call all filters in FilterChain
func (c *FilterChain) Do(r *simpleRequest) FilterStatus {
	cmd := string(bytes.ToLower(r.Body().Array[0].Text))
	for _, f := range c.filters {
		if f.Do(cmd, r) == Stop {
			return Stop
		}
	}
	return Continue
}
