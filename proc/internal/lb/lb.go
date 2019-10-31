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
	"math/rand"
	"time"

	"go.uber.org/atomic"

	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// New creates a load balancer with given policy.
func New(p service.LoadBalancePolicy) Balancer {
	switch p {
	case service.LoadBalancePolicy_LEAST_CONNECTION:
		return newLeastConnBalancer()
	case service.LoadBalancePolicy_RANDOM:
		return newRandomBalancer()
	case service.LoadBalancePolicy_ROUND_ROBIN:
		fallthrough
	default:
		return newRoundRobinBalancer()
	}
}

// Balancer is a generic load balancer.
type Balancer interface {
	Name() string
	PickHost(hosts []*host.Host) *host.Host
}

func newRoundRobinBalancer() *roundRobinBalancer {
	return &roundRobinBalancer{
		index: atomic.NewUint64(0),
	}
}

type roundRobinBalancer struct {
	index *atomic.Uint64
}

func (rrb *roundRobinBalancer) Name() string {
	return "RoundRobin"
}

func (rrb *roundRobinBalancer) PickHost(hosts []*host.Host) *host.Host {
	if len(hosts) == 0 {
		return nil
	}
	return hosts[rrb.index.Inc()%uint64(len(hosts))]
}

var randInt = func() int {
	return rand.Int()
}

func newRandomBalancer() *randomBalancer {
	return &randomBalancer{}
}

type randomBalancer struct{}

func (rb *randomBalancer) Name() string {
	return "Random"
}

func (rb *randomBalancer) PickHost(hosts []*host.Host) *host.Host {
	if len(hosts) == 0 {
		return nil
	}
	return hosts[randInt()%len(hosts)]
}

func newLeastConnBalancer() *leastConnBalancer {
	return &leastConnBalancer{}
}

type leastConnBalancer struct{}

func (lcb *leastConnBalancer) Name() string {
	return "LeastConnection"
}

func (lcb *leastConnBalancer) PickHost(hosts []*host.Host) *host.Host {
	if len(hosts) == 0 {
		return nil
	}
	host1 := hosts[randInt()%len(hosts)]
	host2 := hosts[randInt()%len(hosts)]
	if host1.ConnCount() < host2.ConnCount() {
		return host1
	}
	return host2
}
