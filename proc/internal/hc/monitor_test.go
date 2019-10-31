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

package hc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/logger"
)

func TestNewMonitor(t *testing.T) {
	ep, _ := newEndpoint()
	defer ep.Stop()
	hostSet := host.NewSet(host.New(ep.Addr()))
	cases := []struct {
		Config           *hc.HealthCheck
		HS               *host.Set
		NotUseDefaultCfg bool
	}{
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_TcpChecker{
					TcpChecker: &hc.TCPChecker{},
				},
			}, hostSet, true,
		},
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_RedisChecker{
					RedisChecker: &hc.RedisChecker{},
				},
			}, hostSet, true,
		},
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_MysqlChecker{
					MysqlChecker: &hc.MySQLChecker{},
				},
			}, hostSet, false,
		},
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_AtcpChecker{
					AtcpChecker: &hc.ATCPChecker{
						Action: []*hc.ATCPChecker_Action{},
					},
				},
			}, hostSet, false,
		},
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_MysqlChecker{
					MysqlChecker: &hc.MySQLChecker{
						Username: "mysql",
					},
				},
			}, hostSet, true,
		},
		{
			&hc.HealthCheck{
				Interval:      defaultInterval,
				Timeout:       defaultTimeout,
				FallThreshold: defaultFallThreshold,
				RiseThreshold: defaultRiseThreshold,
				Checker: &hc.HealthCheck_AtcpChecker{
					AtcpChecker: &hc.ATCPChecker{
						Action: []*hc.ATCPChecker_Action{
							{
								Send:   []byte(`"PING\r\n"`),
								Expect: []byte(`"+PONG"`),
							},
						},
					},
				},
			}, hostSet, true,
		},
		{
			nil, hostSet, false,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case:%d", i+1), func(t *testing.T) {
			m, err := NewMonitor(c.Config, c.HS)
			assert.NoError(t, err)
			if c.Config == nil {
				assert.Nil(t, m)
				return
			}
			if c.NotUseDefaultCfg {
				assert.Equal(t, c.Config, m.config)
			} else {
				assert.Equal(t, defaultHCConfig, m.config)
			}
		})
	}
}

func TestResetHealthCheckPolicy(t *testing.T) {
	ep, _ := newEndpoint()
	defer ep.Stop()

	h := host.New(ep.Addr())
	hostSet := host.NewSet(h)
	hostSet.MarkHostUnhealthy(h)
	config := &hc.HealthCheck{
		Interval:      100 * time.Second,
		Timeout:       2 * time.Second,
		FallThreshold: 3,
		RiseThreshold: 2,
		Checker: &hc.HealthCheck_TcpChecker{
			TcpChecker: &hc.TCPChecker{},
		},
	}
	m, err := NewMonitor(config, hostSet)
	assert.NoError(t, err)
	m.Start()
	defer m.Stop()

	newConfig := &hc.HealthCheck{
		Interval:      200 * time.Millisecond,
		Timeout:       200 * time.Millisecond,
		FallThreshold: 3,
		RiseThreshold: 2,
		Checker: &hc.HealthCheck_TcpChecker{
			TcpChecker: &hc.TCPChecker{},
		},
	}
	m.ResetHealthCheck(newConfig)
	assert.Equal(t, newConfig, m.config)

	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(hostSet.Healthy()))
}

type Endpoint struct {
	lis net.Listener
}

func newEndpoint() (*Endpoint, error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	ep := &Endpoint{
		lis: lis,
	}
	return ep, err
}

func (ep *Endpoint) Stop() {
	ep.lis.Close()
}

func (ep *Endpoint) Addr() string {
	return ep.lis.Addr().String()
}

func TestCheckHosts(t *testing.T) {
	ep1, _ := newEndpoint()
	ep2, _ := newEndpoint()
	defer ep1.Stop()
	defer ep2.Stop()

	host1 := host.New(ep1.Addr())
	host2 := host.New(ep2.Addr())
	hostSet := host.NewSet(host1, host2)
	hostSet.MarkHostUnhealthy(host1)
	hostSet.MarkHostUnhealthy(host2)

	hcPolicy := genConfigForTest()
	m, err := NewMonitor(hcPolicy, hostSet)
	assert.NoError(t, err)
	m.Start()
	defer m.Stop()

	time.Sleep(time.Second * 1)
	assert.Equal(t, 2, len(hostSet.Healthy()))

	ep1.Stop()
	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(hostSet.Healthy()))

	ep2.Stop()
	time.Sleep(time.Second * 1)
	assert.Equal(t, 0, len(hostSet.Healthy()))
}

func TestCheckHostsAllUnreachable(t *testing.T) {
	numOfHosts := 1000
	hostSet := genAllUnreachableHosts(numOfHosts)
	hcPolicy := genConfigForTest()

	m, err := NewMonitor(hcPolicy, hostSet)
	assert.NoError(t, err)
	m.Start()
	defer m.Stop()

	assert.Equal(t, numOfHosts, len(hostSet.Healthy()))
	time.Sleep(time.Second * 3)
	assert.Equal(t, 0, len(hostSet.Healthy()))
}

func BenchmarkCheckHostsConsecutively(b *testing.B) {
	hostSet := genPartiallyAvailableHosts(500)
	m, err := NewMonitor(genConfigForTest(), hostSet)
	assert.NoError(b, err)

	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		checkHostsConsecutively(m)
	}
}

func BenchmarkCheckHostsConcurrently(b *testing.B) {
	hostSet := genPartiallyAvailableHosts(500)
	m, err := NewMonitor(genConfigForTest(), hostSet)
	assert.NoError(b, err)

	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		m.checkHosts()
	}
}

func BenchmarkCheckAllUnreachableHostsConsecutively(b *testing.B) {
	hostSet := genAllUnreachableHosts(1000)
	hcPolicy := genConfigForTest()
	m, err := NewMonitor(hcPolicy, hostSet)
	assert.NoError(b, err)

	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		checkHostsConsecutively(m)
	}
}

func BenchmarkCheckAllUnreachableHostsConcurrently(b *testing.B) {
	hostSet := genAllUnreachableHosts(1000)
	hcPolicy := genConfigForTest()
	m, err := NewMonitor(hcPolicy, hostSet)
	assert.NoError(b, err)

	b.ResetTimer()
	for t := 0; t < b.N; t++ {
		m.checkHosts()
	}
}

func checkHostsConsecutively(m *Monitor) {
	hosts := m.hostSet.All()
	for _, host := range hosts {
		if m.checkHost(host) {
			if host.IncSuccessfulCount() > uint64(m.config.RiseThreshold) {
				if m.hostSet.MarkHostHealthy(host) {
					logger.Infof("Host %s is healthy", host)
				}
			}
			continue
		}
		if host.IncFailedCount() > uint64(m.config.FallThreshold) {
			if m.hostSet.MarkHostUnhealthy(host) {
				logger.Warnf("Host %s is unhealthy", host)
			}
		}
	}
}

func genConfigForTest() *hc.HealthCheck {
	return &hc.HealthCheck{
		Interval:      220 * time.Millisecond,
		Timeout:       200 * time.Millisecond,
		FallThreshold: 3,
		RiseThreshold: 2,
		Checker: &hc.HealthCheck_TcpChecker{
			TcpChecker: &hc.TCPChecker{},
		},
	}
}

func genPartiallyAvailableHosts(num int) *host.Set {
	hosts := make([]*host.Host, 0, num)
	eps := make([]*Endpoint, 0, num)
	for i := 0; i < num; i++ {
		ep, _ := newEndpoint()
		eps = append(eps, ep)
		h := host.New(ep.Addr())
		hosts = append(hosts, h)
		defer ep.Stop()
	}
	// stop first third of hosts
	for i := 0; i < num/3; i++ {
		eps[i].Stop()
	}

	return host.NewSet(hosts...)
}

func genAllUnreachableHosts(num int) *host.Set {
	portBase := 10000
	hosts := make([]*host.Host, 0, num)
	for i := 0; i < num; i++ {
		ip := "129.0.1.2" // arbitrarily set to an unavailable ip
		addr := fmt.Sprintf("%s:%d", ip, portBase+i)
		hosts = append(hosts, host.New(addr))
	}
	return host.NewSet(hosts...)
}
