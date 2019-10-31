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

package controller

import (
	"fmt"
	"sync"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/config"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/proc"
	_ "github.com/samaritan-proxy/samaritan/proc/redis" // register redis processor.
	_ "github.com/samaritan-proxy/samaritan/proc/tcp"   // register tcp processor.
)

// Controller is used to manage and drive processor.
type Controller struct {
	mu sync.RWMutex

	evtC  <-chan config.Event
	procs map[string]proc.Proc

	startOnce sync.Once
	quit      chan struct{}
	quitOnce  sync.Once
	done      chan struct{}
}

// New creates a controller with given event channel.
func New(evtC <-chan config.Event) (*Controller, error) {
	ctl := &Controller{
		evtC:  evtC,
		procs: make(map[string]proc.Proc),
		quit:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	return ctl, nil
}

// Start starts the controller.
func (c *Controller) Start() error {
	c.startOnce.Do(func() {
		go func() {
			defer close(c.done)
			for {
				select {
				case <-c.quit:
					c.stopAllProcs()
					return
				case evt := <-c.evtC:
					c.handleEvent(evt)
				}
			}
		}()
	})
	return nil
}

func (c *Controller) handleEvent(evt config.Event) {
	switch evt := evt.(type) {
	case *config.SvcAddEvent:
		c.handleSvcAdd(evt.Name, evt.Config, evt.Endpoints)
	case *config.SvcRemoveEvent:
		c.handleSvcDel(evt.Name)
	case *config.SvcConfigEvent:
		c.handleSvcConfigUpdate(evt.Name, evt.Config)
	case *config.SvcEndpointEvent:
		c.handleSvcEndpointsAdd(evt.Name, evt.Added)
		c.handleSvcEndpointsRemove(evt.Name, evt.Removed)
	default:
		logger.Warnf("unkown event: %v", evt)
	}

}

// Stop stops the controller.
func (c *Controller) Stop() {
	c.quitOnce.Do(func() {
		close(c.quit)
	})
	<-c.done
}

func (c *Controller) handleSvcAdd(svcName string, cfg *service.Config, endpoints []*service.Endpoint) {
	if _, ok := c.getProc(svcName); ok {
		return
	}
	c.tryEnsureProc(svcName, cfg, endpointsToHosts(endpoints))
}

func (c *Controller) handleSvcDel(svcName string) {
	if p, ok := c.getProc(svcName); ok {
		p.Stop()
		c.removeProc(p)
	}
}

func (c *Controller) getProc(name string) (proc.Proc, bool) {
	c.mu.RLock()
	proc, ok := c.procs[name]
	c.mu.RUnlock()
	return proc, ok
}

func (c *Controller) addProc(proc proc.Proc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	procName := proc.Name()
	c.procs[procName] = proc
	logger.Infof("Add processor %s", procName)
}

func (c *Controller) removeProc(proc proc.Proc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeProcLocked(proc)
}

func (c *Controller) removeProcLocked(proc proc.Proc) {
	procName := proc.Name()
	delete(c.procs, procName)
	logger.Infof("Remove processor %s", procName)
}

var newProc = func(name string, cfg *service.Config, hosts []*host.Host) (proc.Proc, error) {
	return proc.New(name, cfg, hosts)
}

func (c *Controller) tryEnsureProc(svcName string, cfg *service.Config, hosts []*host.Host) (p proc.Proc) {
	if svcName == "" {
		logger.Debugf("empty service name")
		return
	}
	if err := cfg.Validate(); cfg == nil || err != nil {
		logger.Debugf("invalid config")
		return
	}

	proc, err := newProc(svcName, cfg, hosts)
	if err != nil {
		logger.Warnf("Create processor %s failed: %v", svcName, err)
		return
	}
	if err := proc.Start(); err != nil {
		logger.Warnf("Start processor %s failed: %v", svcName, err)
		return
	}
	c.addProc(proc)
	return proc
}

func (c *Controller) handleSvcEndpointsAdd(svcName string, endpoints []*service.Endpoint) {
	if len(endpoints) == 0 {
		return
	}

	procName := svcName
	p, ok := c.getProc(procName)
	if !ok {
		logger.Warnf("failed to get proc of service when add endpoints: %s", svcName)
		return
	}
	hosts := endpointsToHosts(endpoints)
	p.OnSvcHostAdd(hosts)
	logger.Infof("Add hosts %v to processor %s", hosts, procName)
}

func (c *Controller) handleSvcEndpointsRemove(svcName string, endpoints []*service.Endpoint) {
	if len(endpoints) == 0 {
		return
	}

	procName := svcName
	p, ok := c.getProc(procName)
	if !ok {
		logger.Warnf("failed to get proc of service when remove endpoints: %s", svcName)
		return
	}

	hosts := endpointsToHosts(endpoints)
	p.OnSvcHostRemove(hosts)
	logger.Infof("Remove hosts %v from processor %s", hosts, procName)
}

func (c *Controller) handleSvcEndpointsReplace(svcName string, endpoints []*service.Endpoint) {
	if len(endpoints) == 0 {
		return
	}

	procName := svcName
	p, ok := c.getProc(procName)
	if !ok {
		logger.Warnf("failed to get proc of service when replace endpoints: %s", svcName)
		return
	}

	hosts := endpointsToHosts(endpoints)
	p.OnSvcAllHostReplace(hosts)
	logger.Infof("Replace all hosts of processor %s with %v", procName, hosts)
}

func (c *Controller) handleSvcConfigUpdate(svcName string, newCfg *service.Config) {
	proc, ok := c.getProc(svcName)
	if !ok {
		logger.Warnf("failed to get proc of service when update config: %s", svcName)
		return
	}
	if err := proc.OnSvcConfigUpdate(newCfg); err != nil {
		logger.Warnf("failed to update svc config: %v", err)
	}
}

func (c *Controller) stopAllProcs() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, proc := range c.procs {
		// NOTE: maybe need to stop procs concurrently.
		if err := proc.Stop(); err != nil {
			logger.Warnf("Stop processor %s failed: %v", name, err)
		}
		c.removeProcLocked(proc)
	}
}

// DrainListeners drains all processor listener.
func (c *Controller) DrainListeners() {
	c.mu.RLock()
	procs := make(map[string]proc.Proc, len(c.procs))
	for name, proc := range c.procs {
		procs[name] = proc
	}
	c.mu.RUnlock()

	for name, proc := range procs {
		if err := proc.StopListen(); err != nil {
			logger.Warnf("Proc[%s] stop listen failed: %v", name, err)
		}
		logger.Infof("Proc[%s] stop listen done", name)
	}
}

// GetProc returns the specified processor.
func (c *Controller) GetProc(name string) (proc.Proc, bool) {
	return c.getProc(name)
}

// GetAllProcessors return all registered processors.
func (c *Controller) GetAllProcs() []proc.Proc {
	c.mu.RLock()
	procs := make([]proc.Proc, 0, len(c.procs))
	for _, proc := range c.procs {
		procs = append(procs, proc)
	}
	c.mu.RUnlock()
	return procs
}

func endpointsToHosts(endpoints []*service.Endpoint) []*host.Host {
	hosts := make([]*host.Host, 0, len(endpoints))
	for _, endpoint := range endpoints {
		typ := host.TypeMain
		if endpoint.Type == service.Endpoint_BACKUP {
			typ = host.TypeBackup
		}
		addr := fmt.Sprintf("%s:%d", endpoint.Address.Ip, endpoint.Address.Port)
		hosts = append(hosts, host.NewWithType(addr, typ))
	}
	return hosts
}
