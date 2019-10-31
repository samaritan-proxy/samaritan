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

package lb

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

func TestRoundRobinBalancer(t *testing.T) {
	rrb := newRoundRobinBalancer()
	assert.Equal(t, "RoundRobin", rrb.Name())

	// empty hosts
	assert.Equal(t, (*host.Host)(nil), rrb.PickHost([]*host.Host{}))

	// normal
	h1 := host.New(":1234")
	h2 := host.New(":1235")
	h3 := host.New(":1236")
	hosts := []*host.Host{h1, h2, h3}
	assert.Equal(t, h2, rrb.PickHost(hosts))
	assert.Equal(t, h3, rrb.PickHost(hosts))
	assert.Equal(t, h1, rrb.PickHost(hosts))

	// concurrency
	rrb = newRoundRobinBalancer()
	var (
		wg     = new(sync.WaitGroup)
		doPick = func() {
			rrb.PickHost(hosts)
			wg.Done()
		}
	)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go doPick()
	}
	wg.Wait()
	assert.EqualValues(t, uint64(10), rrb.index.Load()) //after 10 picks
	assert.Equal(t, h3, rrb.PickHost(hosts))
}

func TestRandomBalancer(t *testing.T) {
	rb := newRandomBalancer()
	assert.Equal(t, "Random", rb.Name())

	// empty hosts
	assert.Equal(t, (*host.Host)(nil), rb.PickHost([]*host.Host{}))

	// normal
	saved := randInt
	defer func() {
		randInt = saved
	}()
	randInt = func() int {
		return 4
	}

	h1 := host.New(":1234")
	h2 := host.New(":1235")
	h3 := host.New(":1236")
	hosts := []*host.Host{h1, h2, h3}
	assert.Equal(t, h2, rb.PickHost(hosts))
}

func TestLeastConnBalancer(t *testing.T) {
	lcb := newLeastConnBalancer()
	assert.Equal(t, "LeastConnection", lcb.Name())

	// empty hosts
	assert.Equal(t, (*host.Host)(nil), lcb.PickHost([]*host.Host{}))

	// one host
	h1 := host.New(":1234")
	assert.Equal(t, h1, lcb.PickHost([]*host.Host{h1}))

	// normal
	saved := randInt
	defer func() {
		randInt = saved
	}()
	result := 0
	randInt = func() int {
		result++
		return result
	}

	h2 := host.New(":1235")
	h2.IncConnCount()
	hosts := []*host.Host{h1, h2}
	assert.Equal(t, h1, lcb.PickHost(hosts))
}

func TestNewBalancer(t *testing.T) {
	tests := []struct {
		policy       service.LoadBalancePolicy
		balancerName string
	}{
		{service.LoadBalancePolicy_LEAST_CONNECTION, "LeastConnection"},
		{service.LoadBalancePolicy_RANDOM, "Random"},
		{service.LoadBalancePolicy_ROUND_ROBIN, "RoundRobin"},
	}
	for _, test := range tests {
		assert.Equal(t, test.balancerName, New(test.policy).Name())
	}
}
