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

package proc

import (
	"github.com/samaritan-proxy/samaritan/stats"
	"github.com/samaritan-proxy/samaritan/utils"
)

type CommonStats struct {
	CxTotal        *stats.Counter   // total connections
	CxDestroyTotal *stats.Counter   // destroyed connections
	CxActive       *stats.Gauge     // active connections
	CxLengthSec    *stats.Histogram // connection length
	CxRxBytesTotal *stats.Counter   // received connection bytes
	CxTxBytesTotal *stats.Counter   // sent connection bytes

	RqTotal         *stats.Counter   // total request
	RqSuccessTotal  *stats.Counter   // success request
	RqFailureTotal  *stats.Counter   // failed request
	RqActive        *stats.Gauge     // active request
	RqDurationMs    *stats.Histogram // request duration
	RqRxBytesLength *stats.Histogram // received request bytes length
	RqTxBytesLength *stats.Histogram // sent request bytes length
}

type DownstreamStats struct {
	*stats.Scope
	CommonStats
	CxRestricted *stats.Counter // restricted connections
}

type UpstreamStats struct {
	*stats.Scope
	CommonStats
	CxConnectTimeout *stats.Counter // total connection connect timeouts
	CxConnectFail    *stats.Counter // total connection failures
}

type Stats struct {
	*stats.Scope
	Downstream *DownstreamStats
	Upstream   *UpstreamStats
}

func newCommonStats(scope *stats.Scope) CommonStats {
	s := CommonStats{}
	_ = utils.BuildStats(scope, &s)
	return s
}

func NewUpstreamStats(scope *stats.Scope) *UpstreamStats {
	s := &UpstreamStats{}
	scope = scope.NewChild("upstream")
	s.CommonStats = newCommonStats(scope)
	_ = utils.BuildStats(scope, s)
	return s
}

func NewDownstreamStats(scope *stats.Scope) *DownstreamStats {
	s := &DownstreamStats{}
	scope = scope.NewChild("downstream")
	s.CommonStats = newCommonStats(scope)
	_ = utils.BuildStats(scope, s)
	return s
}

func NewStats(scope *stats.Scope) *Stats {
	// TODO: use reflect
	s := &Stats{
		Scope:      scope,
		Downstream: NewDownstreamStats(scope),
		Upstream:   NewUpstreamStats(scope),
	}
	return s
}
