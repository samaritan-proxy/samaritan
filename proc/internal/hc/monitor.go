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
	"context"
	"sync"
	"time"

	hostpkg "github.com/samaritan-proxy/samaritan/host"
	loggerpkg "github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/tcp"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
)

// MaximumConcurrency of hc is used to avoid unexpected large number of hosts resulting in huge number of goroutines.
// Observed that number of hosts in prod env are almost all less than one thousand, we arbitrarily choose 2048 for MaximumConcurrency as the upperbound.
const MaximumConcurrency = 2048

var (
	defaultInterval             = time.Second * 10
	defaultTimeout              = time.Second * 3
	defaultFallThreshold uint32 = 3
	defaultRiseThreshold uint32 = 3
	defaultChecker              = &hc.HealthCheck_TcpChecker{TcpChecker: &hc.TCPChecker{}}
	defaultConfig               = &hc.HealthCheck{
		Interval:      defaultInterval,
		Timeout:       defaultTimeout,
		FallThreshold: defaultFallThreshold,
		RiseThreshold: defaultRiseThreshold,
		Checker:       defaultChecker,
	}
)

// Monitor monitors the health state of hosts.
type Monitor struct {
	logger           log.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	done             chan struct{}
	config           *hc.HealthCheck
	strategyUpdateCh chan struct{}
	checker          checker
	hostSet          *hostpkg.Set
}

// NewMonitor creates a health monitor.
func NewMonitor(config *hc.HealthCheck, hostSet *hostpkg.Set, logger log.Logger) (*Monitor, error) {
	if logger == nil {
		logger = loggerpkg.Get()
	}

	// TODO: don't return nil when config is empty
	if config == nil {
		logger.Infof("health check config is null, healthy check will be disabled")
		return nil, nil
	}
	if err := config.Validate(); err != nil {
		logger.Infof("invalid health check config: %v, will use default", err)
		config = defaultConfig
	}

	checker, err := newChecker(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Monitor{
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		done:             make(chan struct{}),
		config:           config,
		strategyUpdateCh: make(chan struct{}, 1),
		checker:          checker,
		hostSet:          hostSet,
	}
	return m, nil
}

// ResetHealthCheck resets the health check configurations.
func (m *Monitor) ResetHealthCheck(config *hc.HealthCheck) error {
	if m == nil {
		return nil
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if !config.Checker.Equal(m.config.Checker) {
		checker, err := newChecker(config)
		if err != nil {
			m.checker = tcp.NewChecker()
			return err
		}
		m.checker = checker
	}
	m.config = config
	m.strategyUpdateCh <- struct{}{}
	return nil
}

// Start starts the monitor.
func (m *Monitor) Start() {
	if m == nil {
		return
	}
	go m.loop()
}

func (m *Monitor) loop() {
	defer close(m.done)

	ticker := time.NewTicker(m.config.Interval)
	for {
		select {
		case <-m.ctx.Done():
			ticker.Stop()
			return
		case <-m.strategyUpdateCh:
			ticker.Stop()
			ticker = time.NewTicker(m.config.Interval)
		case <-ticker.C:
			m.checkHosts()
		}
	}
}

func (m *Monitor) checkHostAndUpdateStatus(host *hostpkg.Host) {
	if m.checkHost(host) {
		if host.IncSuccessfulCount() > uint64(m.config.RiseThreshold) {
			if m.hostSet.MarkHostHealthy(host) {
				m.logger.Infof("Host %s is healthy", host)
			}
		}
		return
	}
	if host.IncFailedCount() > uint64(m.config.FallThreshold) {
		if m.hostSet.MarkHostUnhealthy(host) {
			m.logger.Warnf("Host %s is unhealthy", host)
		}
	}
}

func (m *Monitor) checkHosts() {
	hosts := m.hostSet.All()
	var (
		concurrency = MinInt(len(hosts), MaximumConcurrency)
		hostCh      = make(chan *hostpkg.Host, concurrency)
		wg          sync.WaitGroup
	)

	go func() {
		for _, host := range hosts {
			hostCh <- host
		}
		close(hostCh)
	}()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			for host := range hostCh {
				m.checkHostAndUpdateStatus(host)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func (m *Monitor) checkHost(host *hostpkg.Host) bool {
	if err := m.checker.Check(host.Addr, m.config.Timeout); err != nil {
		m.logger.Debugf("Check host[%s] state failed: %s", host, err)
		return false
	}
	return true
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	if m == nil {
		return
	}
	m.cancel()
	<-m.done
}
