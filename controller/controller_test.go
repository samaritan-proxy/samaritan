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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/config"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/mock"
	"github.com/samaritan-proxy/samaritan/utils"
)

func newTestController(t *testing.T) (*Controller, chan<- config.Event) {
	ch := make(chan config.Event, 1024)
	c, err := New(ch)
	assert.NoError(t, err)
	return c, ch
}

func getTestSvc() *service.Service {
	return &service.Service{
		Name: "test",
	}
}

func assertNotTimeout(t *testing.T, timeout time.Duration, fn func()) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-timer.C:
		t.Error("exec timeout")
	case <-done:
	}
}

func getTestSvcConf() *service.Config {
	return &service.Config{
		HealthCheck: &hc.HealthCheck{
			Interval:      10 * time.Second,
			Timeout:       3 * time.Second,
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
		ProtocolOptions: &service.Config_TcpOption{
			TcpOption: &protocol.TCPOption{},
		},
	}
}

func getTestEndpoints() []*service.Endpoint {
	return []*service.Endpoint{
		{
			Address: &common.Address{
				Ip:   "10.101.68.90",
				Port: 7332,
			},
		},
		{
			Address: &common.Address{
				Ip:   "10.101.68.92",
				Port: 7402,
			},
		},
		{
			Address: &common.Address{
				Ip:   "10.101.68.93",
				Port: 7316,
			},
		},
		{
			Address: &common.Address{
				Ip:   "10.101.68.94",
				Port: 7326,
			},
		},
	}
}

func newTestSvcAddEvent() *config.SvcAddEvent {
	return &config.SvcAddEvent{
		Name:      getTestSvc().Name,
		Config:    getTestSvcConf(),
		Endpoints: getTestEndpoints(),
	}
}

func newTestSvcRemoveEvent() *config.SvcRemoveEvent {
	return &config.SvcRemoveEvent{
		Name: getTestSvc().Name,
	}
}

func TestControllerHandleServiceAddAndRemove(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	var (
		startDone, stopDone = make(chan struct{}), make(chan struct{})
		oldNewProc          = newProc
	)
	newProc = func(name string, cfg *service.Config, hosts []*host.Host) (proc proc.Proc, e error) {
		assert.Equal(t, name, getTestSvc().GetName())
		assert.True(t, cfg.Equal(getTestSvcConf()))
		p := mock.NewMockProc(ctrl)
		p.EXPECT().Name().Return(name).AnyTimes()
		p.EXPECT().Start().Return(nil).Times(1).Do(func() {
			close(startDone)
		})
		p.EXPECT().Stop().Times(1).Do(func() {
			close(stopDone)
		})
		return p, nil
	}
	defer func() {
		newProc = oldNewProc
	}()

	var p proc.Proc
	// add service
	t.Run("add", func(t *testing.T) {
		ch <- newTestSvcAddEvent()
		assertNotTimeout(t, time.Second, func() {
			<-startDone
		})
		var ok bool
		p, ok = c.getProc(getTestSvc().GetName())
		assert.True(t, ok)
	})

	// add service repeat
	t.Run("add again", func(t *testing.T) {
		ch <- newTestSvcAddEvent()
		p2, ok := c.getProc(getTestSvc().GetName())
		assert.True(t, ok)
		assert.Equal(t, p, p2)
	})

	// del service
	t.Run("del", func(t *testing.T) {
		ch <- newTestSvcRemoveEvent()
		assertNotTimeout(t, time.Second, func() {
			<-stopDone
		})
		_, ok := c.getProc(getTestSvc().GetName())
		assert.False(t, ok)
	})
}

func TestControllerHandleServiceAddWithBadSvc(t *testing.T) {
	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()
	evt := newTestSvcAddEvent()
	evt.Name = ""
	ch <- evt
	for {
		if len(ch) == 0 {
			break
		}
		time.Sleep(time.Microsecond * 10)
	}
	time.Sleep(time.Second)
	assert.Empty(t, c.GetAllProcs())
}

func TestControllerHandleServiceAddWithBadSvcConfig(t *testing.T) {
	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	// with bad config
	t.Run("bad config", func(t *testing.T) {
		evt := newTestSvcAddEvent()
		evt.Config.Listener.Address.Port = 99999
		ch <- evt
		assertNotTimeout(t, time.Second, func() {
			for {
				if len(ch) == 0 {
					break
				}
				time.Sleep(time.Microsecond * 10)
			}
		})
		assert.Empty(t, c.GetAllProcs())
	})

	// with nil config
	t.Run("nil config", func(t *testing.T) {
		evt := newTestSvcAddEvent()
		evt.Config = nil
		assertNotTimeout(t, time.Second, func() {
			for {
				if len(ch) == 0 {
					break
				}
				time.Sleep(time.Microsecond * 10)
			}
		})
		assert.Empty(t, c.GetAllProcs())
	})
}

func TestTryEnsureProcWithError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	t.Run("newProc with error", func(t *testing.T) {
		backup := newProc
		done := make(chan struct{})
		newProc = func(name string, cfg *service.Config, hosts []*host.Host) (i proc.Proc, e error) {
			defer func() { close(done) }()
			return nil, errors.New("this is error")
		}
		defer func() { newProc = backup }()

		ch <- newTestSvcAddEvent()
		assertNotTimeout(t, time.Second, func() {
			<-done
		})
		assert.Empty(t, c.GetAllProcs())
	})

	t.Run("start proc with error", func(t *testing.T) {
		backup := newProc
		done := make(chan struct{})
		newProc = func(name string, cfg *service.Config, hosts []*host.Host) (proc proc.Proc, e error) {
			assert.Equal(t, name, getTestSvc().GetName())
			assert.True(t, cfg.Equal(getTestSvcConf()))
			p := mock.NewMockProc(ctrl)
			p.EXPECT().Start().Return(errors.New("this is error")).Times(1).Do(func() {
				close(done)
			})
			return p, nil
		}
		defer func() { newProc = backup }()
		ch <- newTestSvcAddEvent()
		assertNotTimeout(t, time.Second, func() {
			<-done
		})
		assert.Empty(t, c.GetAllProcs())
	})
}

func TestHandleServiceHostsAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	called := make(chan struct{})
	p := mock.NewMockProc(ctrl)
	p.EXPECT().Name().Return(getTestSvc().GetName()).AnyTimes()
	p.EXPECT().OnSvcHostAdd(gomock.Any()).Return(nil).Do(func(_ interface{}) {
		close(called)
	})
	p.EXPECT().Stop().Return(nil)

	c.addProc(p)
	ch <- &config.SvcEndpointEvent{
		Name:  getTestSvc().Name,
		Added: getTestEndpoints(),
	}
	assertNotTimeout(t, time.Second, func() {
		<-called
	})
}

func TestHandleServiceHostsDel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	called := make(chan struct{})
	p := mock.NewMockProc(ctrl)
	p.EXPECT().Name().Return(getTestSvc().GetName()).AnyTimes()
	p.EXPECT().OnSvcHostRemove(gomock.Any()).Return(nil).Do(func(_ interface{}) {
		close(called)
	})
	p.EXPECT().Stop().Return(nil)

	c.addProc(p)
	ch <- &config.SvcEndpointEvent{
		Name:    getTestSvc().Name,
		Removed: getTestEndpoints(),
	}
	assertNotTimeout(t, time.Second, func() {
		<-called
	})
}

//func TestHandleServiceHostsReplace(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	c, ch := newTestController(t)
//	assert.NoError(t, c.Start())
//	defer c.Stop()
//
//	called := make(chan struct{})
//	p := mock.NewMockProc(ctrl)
//	p.EXPECT().Name().Return(getTestSvc().GetName()).AnyTimes()
//	p.EXPECT().OnSvcAllHostReplace(gomock.Any()).Return(nil).Do(func(_ interface{}) {
//		close(called)
//	})
//	p.EXPECT().Stop().Return(nil)
//
//	c.addProc(p)
//	ch <- config.NewEndpointEvt(event.OpReplace, getTestSvc(), getTestEndpoints())
//	assertNotTimeout(t, time.Second, func() {
//		<-called
//	})
//}

func TestHandleServiceConfigUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, ch := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	called := make(chan struct{})
	p := mock.NewMockProc(ctrl)
	p.EXPECT().Name().Return(getTestSvc().GetName()).AnyTimes()
	p.EXPECT().OnSvcConfigUpdate(gomock.Any()).Return(nil).Do(func(_ interface{}) {
		close(called)
	})
	p.EXPECT().Stop().Return(nil)

	c.addProc(p)
	cfg := getTestSvcConf()
	cfg.LbPolicy = service.LoadBalancePolicy_RANDOM
	ch <- &config.SvcConfigEvent{
		Name:   getTestSvc().Name,
		Config: cfg,
	}
	assertNotTimeout(t, time.Second, func() {
		<-called
	})
}

func TestControllerDrainListeners(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c, _ := newTestController(t)
	assert.NoError(t, c.Start())
	defer c.Stop()

	for i := 0; i < 5; i++ {
		p := mock.NewMockProc(ctrl)
		p.EXPECT().Name().Return(fmt.Sprintf("test_%d", i)).AnyTimes()
		p.EXPECT().Stop().Return(nil)
		p.EXPECT().StopListen().Return(nil)
		c.addProc(p)
	}
	c.DrainListeners()
}
