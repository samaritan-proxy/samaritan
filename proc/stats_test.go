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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/stats"
)

func TestNewStats(t *testing.T) {
	s := NewStats(stats.CreateScope("test"))
	assert.NotNil(t, s)

	assert.Equal(t, "test.downstream.cx_total", s.Downstream.CxTotal.Name())
	assert.Equal(t, "test.downstream.cx_destroy_total", s.Downstream.CxDestroyTotal.Name())
	assert.Equal(t, "test.downstream.cx_active", s.Downstream.CxActive.Name())
	assert.Equal(t, "test.downstream.cx_length_sec", s.Downstream.CxLengthSec.Name())
	assert.Equal(t, "test.downstream.cx_rx_bytes_total", s.Downstream.CxRxBytesTotal.Name())
	assert.Equal(t, "test.downstream.cx_tx_bytes_total", s.Downstream.CxTxBytesTotal.Name())

	assert.Equal(t, "test.downstream.rq_total", s.Downstream.RqTotal.Name())
	assert.Equal(t, "test.downstream.rq_failure_total", s.Downstream.RqFailureTotal.Name())
	assert.Equal(t, "test.downstream.rq_active", s.Downstream.RqActive.Name())
	assert.Equal(t, "test.downstream.rq_duration_ms", s.Downstream.RqDurationMs.Name())
	assert.Equal(t, "test.downstream.rq_rx_bytes_length", s.Downstream.RqRxBytesLength.Name())
	assert.Equal(t, "test.downstream.rq_tx_bytes_length", s.Downstream.RqTxBytesLength.Name())

	assert.Equal(t, "test.downstream.cx_restricted", s.Downstream.CxRestricted.Name())

	assert.Equal(t, "test.upstream.cx_total", s.Upstream.CxTotal.Name())
	assert.Equal(t, "test.upstream.cx_destroy_total", s.Upstream.CxDestroyTotal.Name())
	assert.Equal(t, "test.upstream.cx_active", s.Upstream.CxActive.Name())
	assert.Equal(t, "test.upstream.cx_length_sec", s.Upstream.CxLengthSec.Name())
	assert.Equal(t, "test.upstream.cx_rx_bytes_total", s.Upstream.CxRxBytesTotal.Name())
	assert.Equal(t, "test.upstream.cx_tx_bytes_total", s.Upstream.CxTxBytesTotal.Name())

	assert.Equal(t, "test.upstream.rq_total", s.Upstream.RqTotal.Name())
	assert.Equal(t, "test.upstream.rq_failure_total", s.Upstream.RqFailureTotal.Name())
	assert.Equal(t, "test.upstream.rq_active", s.Upstream.RqActive.Name())
	assert.Equal(t, "test.upstream.rq_duration_ms", s.Upstream.RqDurationMs.Name())
	assert.Equal(t, "test.upstream.rq_rx_bytes_length", s.Upstream.RqRxBytesLength.Name())
	assert.Equal(t, "test.upstream.rq_tx_bytes_length", s.Upstream.RqTxBytesLength.Name())

	assert.Equal(t, "test.upstream.cx_connect_timeout", s.Upstream.CxConnectTimeout.Name())
	assert.Equal(t, "test.upstream.cx_connect_fail", s.Upstream.CxConnectFail.Name())
}
