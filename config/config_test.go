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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

func newBootstrpTempFile(b *bootstrap.Bootstrap) (f *os.File, err error) {
	f, err = ioutil.TempFile("", "bootstrap")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	m := new(jsonpb.Marshaler)
	if err := m.Marshal(f, b); err != nil {
		return nil, err
	}
	err = f.Close()
	return
}

func newTestStaticService() *bootstrap.StaticService {
	s := &bootstrap.StaticService{
		Name: "test",
		Config: &service.Config{
			HealthCheck: &hc.HealthCheck{
				Interval:      2 * time.Second,
				Timeout:       2 * time.Second,
				FallThreshold: 3,
				RiseThreshold: 3,
				Checker: &hc.HealthCheck_TcpChecker{
					TcpChecker: &hc.TCPChecker{},
				},
			},
			Listener: &service.Listener{
				Address: &common.Address{
					Ip:   "0.0.0.0",
					Port: 12321,
				},
			},
			ConnectTimeout: utils.DurationPtr(3 * time.Second),
			IdleTimeout:    utils.DurationPtr(10 * time.Minute),
			LbPolicy:       service.LoadBalancePolicy_LEAST_CONNECTION,
			Protocol:       protocol.TCP,
		},
		Endpoints: []*service.Endpoint{
			{
				Address: &common.Address{
					Ip:   "10.101.68.90",
					Port: 7332,
				},
			},
		},
	}
	return s
}

func TestNewWithFile(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
	}

	f, err := newBootstrpTempFile(b)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	c, err := New(b)
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

func TestInitStaticSvcs(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
		StaticServices: []*bootstrap.StaticService{
			newTestStaticService(),
		},
	}

	c, err := New(b)
	assert.NoError(t, err)

	evtCh := c.Subscribe()
	e := <-evtCh
	evt, ok := e.(*SvcAddEvent)
	assert.True(t, ok)

	assert.Equal(t, "test", evt.Name)
	assert.Equal(t, b.StaticServices[0].Config, evt.Config)
	assert.Equal(t, b.StaticServices[0].Endpoints, evt.Endpoints)
}

func mockDynamicSourceFactory(d DynamicSource) (rollback func()) {
	old := dynamicSourceFactory
	dynamicSourceFactory = func(*bootstrap.Bootstrap) (DynamicSource, error) {
		return d, nil
	}
	return func() {
		dynamicSourceFactory = old
	}
}

func TestInitDynamic(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
		Instance: &common.Instance{
			Belong: "foo",
		},
		DynamicSourceConfig: &bootstrap.ConfigSource{
			Endpoint: "test",
		},
	}

	// mock dynamic source
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := NewMockDynamicSource(ctrl)
	d.EXPECT().SetDependencyHook(gomock.Any())
	d.EXPECT().SetSvcConfigHook(gomock.Any())
	d.EXPECT().SetSvcEndpointHook(gomock.Any())
	d.EXPECT().Serve()

	rollback := mockDynamicSourceFactory(d)
	defer rollback()

	c, err := New(b)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	time.Sleep(time.Millisecond * 100) // wait dynamic source serving.
}

func TestHandleDependencyUpdate(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
	}

	c, err := New(b)
	assert.NoError(t, err)
	evtCh := c.Subscribe()

	added := []*service.Service{
		{Name: "foo"},
		{Name: "bar"},
	}
	c.handleDependencyUpdate(added, nil)
	assert.Equal(t, 2, len(c.sws))

	removed := []*service.Service{
		{Name: "bar"},
		{Name: "zoo"},
	}
	c.handleDependencyUpdate(nil, removed)
	assert.Equal(t, 1, len(c.sws))

	e := <-evtCh
	evt, ok := e.(*SvcRemoveEvent)
	assert.True(t, ok)
	assert.Equal(t, "bar", evt.Name)
}
func TestHandleSvcConfigUpdate(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
	}

	t.Run("non-existent", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()
		c.handleSvcConfigUpdate("foo", nil)
		assert.Equal(t, 0, len(evtCh))
	})

	t.Run("no enpoints", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()

		addedSvcs := []*service.Service{
			{Name: "foo"},
		}
		c.handleDependencyUpdate(addedSvcs, nil)

		newCfg := new(service.Config)
		c.handleSvcConfigUpdate("foo", newCfg)
		assert.Equal(t, 0, len(evtCh))
		assert.Equal(t, newCfg, c.sws["foo"].Config)
	})

	t.Run("has enpoints", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()

		c.handleDependencyUpdate(
			[]*service.Service{{Name: "foo"}},
			nil,
		)
		c.handleSvcEndpointUpdate(
			"foo",
			[]*service.Endpoint{
				{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
			},
			nil,
		)

		// first update will emit svc add event
		newCfg := new(service.Config)
		c.handleSvcConfigUpdate("foo", newCfg)
		_, ok := (<-evtCh).(*SvcAddEvent)
		assert.True(t, ok)

		// subsequent update will emit svc config event
		c.handleSvcConfigUpdate("foo", newCfg)
		evt, ok := (<-evtCh).(*SvcConfigEvent)
		assert.True(t, ok)
		assert.Equal(t, "foo", evt.Name)
		assert.Equal(t, newCfg, evt.Config)
	})
}

func TestHandleSvEndpointUpdate(t *testing.T) {
	b := &bootstrap.Bootstrap{
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "127.0.0.1",
				Port: 8888,
			},
		},
	}

	t.Run("non-existent", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()
		added := []*service.Endpoint{
			{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
		}
		c.handleSvcEndpointUpdate("foo", added, nil)
		assert.Equal(t, 0, len(evtCh))
	})

	t.Run("no config", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()

		addedSvcs := []*service.Service{
			{Name: "foo"},
		}
		c.handleDependencyUpdate(addedSvcs, nil)

		added := []*service.Endpoint{
			{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
		}
		c.handleSvcEndpointUpdate("foo", added, nil)
		assert.Equal(t, 0, len(evtCh))
		assert.Equal(t, added, c.sws["foo"].Endpoints)
	})

	t.Run("has config", func(t *testing.T) {
		c, err := New(b)
		assert.NoError(t, err)
		evtCh := c.Subscribe()

		c.handleDependencyUpdate(
			[]*service.Service{{Name: "foo"}},
			nil,
		)
		c.handleSvcConfigUpdate("foo", new(service.Config))

		// first update will emit svc add event
		endpoints := []*service.Endpoint{
			{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
			// duplidate endpoint
			{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
		}
		c.handleSvcEndpointUpdate("foo", endpoints, nil)
		_, ok := (<-evtCh).(*SvcAddEvent)
		assert.True(t, ok)
		assert.Equal(t, 1, len(c.sws["foo"].Endpoints))

		// subsequent update will emit svc endpoint event
		// ignore enpoint which already exists.
		c.handleSvcEndpointUpdate("foo", endpoints, nil)
		assert.Equal(t, 0, len(evtCh))

		c.handleSvcEndpointUpdate("foo", nil, endpoints)
		evt, ok := (<-evtCh).(*SvcEndpointEvent)
		assert.True(t, ok)
		assert.Equal(t, "foo", evt.Name)
		assert.Equal(t, 0, len(evt.Added))
		assert.Equal(t, 1, len(evt.Removed))
		assert.Equal(t, 0, len(c.sws["foo"].Endpoints))
	})
}

func TestIsContainEndpoint(t *testing.T) {
	endpoints := []*service.Endpoint{
		{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
		{Address: &common.Address{Ip: "127.0.0.1", Port: 8889}},
		{Address: &common.Address{Ip: "127.0.0.1", Port: 9000}},
	}

	t.Run("non-existent", func(t *testing.T) {
		endpoint := &service.Endpoint{
			Address: &common.Address{Ip: "127.0.0.1", Port: 9001},
		}
		_, ok := isContainEndpoint(endpoints, endpoint)
		assert.False(t, ok)
	})

	t.Run("existent", func(t *testing.T) {
		endpoint := &service.Endpoint{
			Address: &common.Address{Ip: "127.0.0.1", Port: 8889},
		}
		i, ok := isContainEndpoint(endpoints, endpoint)
		assert.True(t, ok)
		assert.Equal(t, 1, i)
	})
}
