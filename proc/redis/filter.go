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
	"errors"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/stats"
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

// Filter is a Filter of request
type Filter interface {
	Do(req *simpleRequest) FilterStatus
	Destroy()
}

// filterBuilder creates a Filter
type filterBuilder interface {
	Build(filterBuildParams) (Filter, error)
}

// filterBuildParams wraps all the parameters required by the builder
type filterBuildParams struct {
	Config *service.Config
	Scope  *stats.Scope
}

// FilterChain is a set of filters
type FilterChain struct {
	filtersList []Filter
}

// newRequestFilterChain return a new FilterChain
func newRequestFilterChain() *FilterChain {
	return &FilterChain{
		filtersList: make([]Filter, 0),
	}
}

// AddFilter add a Filter
func (c *FilterChain) AddFilter(f Filter) error {
	if f == nil {
		return ErrFilterIsNull
	}
	c.filtersList = append(c.filtersList, f)
	return nil
}

// Reset clean and destroy all filters
func (c *FilterChain) Reset() {
	for _, f := range c.filtersList {
		f.Destroy()
	}
	c.filtersList = make([]Filter, 0)
}

// Do call all filters in FilterChain
func (c *FilterChain) Do(r *simpleRequest) FilterStatus {
	if c == nil {
		return Continue
	}
	for _, f := range c.filtersList {
		if f.Do(r) == Stop {
			return Stop
		}
	}
	return Continue
}
