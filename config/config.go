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

package config

import (
	"encoding/json"
	"sync"

	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/logger"
)

type serviceWrapper struct {
	*service.Service
	Config    *service.Config
	Endpoints []*service.Endpoint
}

func (sw *serviceWrapper) MarshalJSON() ([]byte, error) {
	res := struct {
		Name      string              `json:"name"`
		Config    *service.Config     `json:"config"`
		Endpoints []*service.Endpoint `json:"endpoints"`
	}{
		Name:      sw.Name,
		Config:    sw.Config,
		Endpoints: sw.Endpoints,
	}
	return json.Marshal(res)
}

type Config struct {
	sync.RWMutex
	*bootstrap.Bootstrap
	d     DynamicSource
	sws   map[string]*serviceWrapper
	evtCh chan Event
}

// New creates config with the given bootstrap object.
func New(b *bootstrap.Bootstrap) (*Config, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	c := &Config{
		Bootstrap: b,
		sws:       make(map[string]*serviceWrapper),
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) init() error {
	// It's enough in most scenarios.
	evtLen := 32
	if len(c.StaticServices) > evtLen {
		evtLen = len(c.StaticServices)
	}
	c.evtCh = make(chan Event, evtLen)

	c.initStaticSvcs()
	c.initDynamic()
	return nil
}

func (c *Config) initStaticSvcs() {
	for _, svc := range c.StaticServices {
		if svc == nil {
			continue
		}

		sw := &serviceWrapper{
			Service:   &service.Service{Name: svc.Name},
			Config:    svc.Config,
			Endpoints: svc.Endpoints,
		}
		c.sws[svc.Name] = sw
		c.emitSvcAddEvent(sw)
	}
}

var dynamicSourceFactory = func(b *bootstrap.Bootstrap) (DynamicSource, error) {
	return newDynamicSource(b)
}

func (c *Config) initDynamic() error {
	if c.DynamicSourceConfig == nil {
		logger.Debugf("The config of dynamic source is empty, dynamic source will be disabled")
		return nil
	}

	if c.Instance.Belong == "" {
		logger.Warnf("The service which instance belongs to is empty, dynamic source will be disabled")
		return nil
	}

	d, err := dynamicSourceFactory(c.Bootstrap)
	if err != nil {
		return err
	}

	d.SetSvcHook(c.handleSvcUpdate)
	d.SetSvcConfigHook(c.handleSvcConfigUpdate)
	d.SetSvcEndpointHook(c.handleSvcEndpointUpdate)
	c.d = d
	go c.d.Serve()
	return nil
}

// Subscribe subscribes the config update.
func (c *Config) Subscribe() <-chan Event {
	return c.evtCh
}

// MarshalJSON implements json.Marshaler interface
func (c *Config) MarshalJSON() ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	// TODO: marshal other fields of bootstrap
	res := struct {
		Admin    *bootstrap.Admin           `json:"admin"`
		Services map[string]*serviceWrapper `json:"services"`
	}{
		Admin:    c.Admin,
		Services: c.sws,
	}
	return json.Marshal(res)
}

func (c *Config) handleSvcUpdate(added, removed []*service.Service) {
	c.Lock()
	defer c.Unlock()

	for _, svc := range added {
		sw, ok := c.sws[svc.Name]
		if ok {
			continue
		}
		sw = &serviceWrapper{Service: svc}
		c.sws[svc.Name] = sw
		// NOTE: Due to lack of config and endpoints, don't do anything.
		// Once the config and endpoints aren't empty, will emit service
		// add event.
	}

	for _, svc := range removed {
		sw, ok := c.sws[svc.Name]
		if !ok {
			continue
		}
		delete(c.sws, svc.Name)
		c.emitSvcRemoveEvent(sw)
	}
}

func (c *Config) handleSvcConfigUpdate(svcName string, newCfg *service.Config) {
	c.Lock()
	defer c.Unlock()

	sw, ok := c.sws[svcName]
	// service is already removed, ignore config update.
	if !ok {
		return
	}

	oldCfg := sw.Config
	sw.Config = newCfg

	if sw.Endpoints == nil {
		return
	}
	switch oldCfg {
	case nil:
		c.emitSvcAddEvent(sw)
	default:
		c.emitSvcConfigEvent(svcName, newCfg)
	}
}

func (c *Config) handleSvcEndpointUpdate(svcName string, added, removed []*service.Endpoint) {
	c.Lock()
	defer c.Unlock()

	if len(added) == 0 && len(removed) == 0 {
		return
	}

	sw, ok := c.sws[svcName]
	// service is already removed, ignore endpoint update.
	if !ok {
		return
	}

	oldEndpoints := sw.Endpoints
	validRemoved := make([]*service.Endpoint, 0, len(removed))
	for _, endpoint := range removed {
		i, ok := isContainEndpoint(sw.Endpoints, endpoint)
		if !ok {
			continue
		}
		sw.Endpoints = append(sw.Endpoints[:i], sw.Endpoints[i+1:]...)
		validRemoved = append(validRemoved, endpoint)
	}

	validAdded := make([]*service.Endpoint, 0, len(added))
	for _, endpoint := range added {
		_, ok := isContainEndpoint(sw.Endpoints, endpoint)
		if ok {
			continue
		}
		sw.Endpoints = append(sw.Endpoints, endpoint)
		validAdded = append(validAdded, endpoint)
	}

	if sw.Config == nil {
		return
	}
	switch oldEndpoints {
	case nil:
		c.emitSvcAddEvent(sw)
	default:
		c.emitSvcEndpointEvent(svcName, validAdded, validRemoved)
	}
}

func isContainEndpoint(endpoints []*service.Endpoint, endpoint *service.Endpoint) (int, bool) {
	for i := 0; i < len(endpoints); i++ {
		if !endpoints[i].Address.Equal(endpoint.Address) {
			continue
		}
		return i, true
	}
	return 0, false
}

func (c *Config) emitSvcAddEvent(sw *serviceWrapper) {
	evt := &SvcAddEvent{
		Name:      sw.Service.Name,
		Config:    sw.Config,
		Endpoints: sw.Endpoints,
	}
	c.evtCh <- evt
}

func (c *Config) emitSvcRemoveEvent(sw *serviceWrapper) {
	evt := &SvcRemoveEvent{Name: sw.Service.Name}
	c.evtCh <- evt
}

func (c *Config) emitSvcConfigEvent(svcName string, newCfg *service.Config) {
	evt := &SvcConfigEvent{
		Name:   svcName,
		Config: newCfg,
	}
	c.evtCh <- evt
}

func (c *Config) emitSvcEndpointEvent(svcName string, added, removed []*service.Endpoint) {
	if len(added) == 0 && len(removed) == 0 {
		return
	}
	evt := &SvcEndpointEvent{
		Name:    svcName,
		Added:   added,
		Removed: removed,
	}
	c.evtCh <- evt
}
