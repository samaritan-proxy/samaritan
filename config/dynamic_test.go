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
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

type discoveryServiceClientFactory func(*bootstrap.ConfigSource) (api.DiscoveryServiceClient, func() error, error)

func newDiscoveryServiceClientFacotry(c api.DiscoveryServiceClient, shutdown func() error) discoveryServiceClientFactory {
	if shutdown == nil {
		shutdown = func() error {
			return nil
		}
	}
	return func(*bootstrap.ConfigSource) (api.DiscoveryServiceClient, func() error, error) {
		return c, shutdown, nil
	}
}

func mockNewDiscoveryServiceClient(factory discoveryServiceClientFactory) (rollback func()) {
	oldFactory := newDiscoveryServiceClient
	newDiscoveryServiceClient = factory
	return func() {
		newDiscoveryServiceClient = oldFactory
	}
}

type rpcMsg struct {
	msg proto.Message
}

func (r *rpcMsg) Matches(msg interface{}) bool {
	m, ok := msg.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(m, r.msg)
}

func (r *rpcMsg) String() string {
	return fmt.Sprintf("is %s", r.msg)
}

func newTestInstance() *common.Instance {
	return &common.Instance{
		Id:      "test",
		Version: "1.0.0",
		Belong:  "sam",
	}
}

func TestDynamicSourceStreamSvcs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := new(bootstrap.Bootstrap)
	b.Instance = newTestInstance()
	quit := make(chan struct{})

	// mock discovery service client and stream
	req := &api.DependencyDiscoveryRequest{Instance: b.Instance}
	stream := NewMockDiscoveryService_StreamDependencyClient(ctrl)
	addedSvcs := []*service.Service{
		{Name: "foo"},
	}
	removedSvcs := []*service.Service{
		{Name: "bar"},
	}
	stream.EXPECT().Recv().Return(&api.DependencyDiscoveryResponse{
		Added:   addedSvcs,
		Removed: removedSvcs,
	}, nil)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		<-quit
		return nil, io.EOF
	})
	c := NewMockDiscoveryServiceClient(ctrl)
	c.EXPECT().StreamDependency(gomock.Any(), &rpcMsg{msg: req}).Return(stream, nil)

	// mock discovery service client facotory
	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	d, err := newDynamicSource(b)
	assert.NoError(t, err)

	// register hooks
	svcHookCalled := false
	d.SetSvcHook(func(added, removed []*service.Service) {
		assert.Equal(t, added, addedSvcs)
		assert.Equal(t, removed, removedSvcs)
		svcHookCalled = true
	})

	svcSubHookCalled := false
	svcUnsubHookCalled := false
	d.svcSubHook = func(svc *service.Service) {
		svcSubHookCalled = true
		assert.Contains(t, addedSvcs, svc)
	}
	d.svcUnsubHook = func(svc *service.Service) {
		svcUnsubHookCalled = true
		assert.Contains(t, removedSvcs, svc)
	}

	// close the discovery service client
	time.AfterFunc(time.Millisecond*100, func() {
		close(quit)
	})
	d.streamSvcs(context.Background())

	// assert hooks
	assert.True(t, svcHookCalled)
	assert.True(t, svcSubHookCalled)
	assert.True(t, svcUnsubHookCalled)
}

func TestDynamicSourceStreamSvcConfigs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := new(bootstrap.Bootstrap)
	quit := make(chan struct{})

	// mock discovery service client and stream
	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	req := &api.SvcConfigDiscoveryRequest{
		SvcNamesSubscribe:   []string{"foo"},
		SvcNamesUnsubscribe: []string{"bar", "zoo"},
	}
	stream.EXPECT().Send(&rpcMsg{msg: req}).Return(nil)
	stream.EXPECT().Recv().Return(&api.SvcConfigDiscoveryResponse{
		Updated: map[string]*service.Config{
			"foo": nil,
		},
	}, nil)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		<-quit
		return nil, io.EOF
	})
	c := NewMockDiscoveryServiceClient(ctrl)
	c.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	// mock discovery service client facotory
	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	d, err := newDynamicSource(b)
	assert.NoError(t, err)

	hookCalled := false
	d.SetSvcConfigHook(func(svcName string, newCfg *service.Config) {
		hookCalled = true
		assert.Equal(t, "foo", svcName)
	})
	d.subscribeSvc(&service.Service{Name: "foo"})
	d.unsubscribeSvc(&service.Service{Name: "bar"})
	d.unsubscribeSvc(&service.Service{Name: "zoo"})

	// close the discovery service client
	time.AfterFunc(time.Millisecond*100, func() {
		close(quit)
		close(d.quit)
	})
	d.streamSvcConfigs(context.Background())
	assert.True(t, hookCalled)
}

func TestDynamicSourceStreamSvcEndpoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := new(bootstrap.Bootstrap)
	quit := make(chan struct{})

	// mock discovery service client and stream
	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	req := &api.SvcEndpointDiscoveryRequest{
		SvcNamesSubscribe:   []string{"foo"},
		SvcNamesUnsubscribe: []string{"bar", "zoo"},
	}
	stream.EXPECT().Send(&rpcMsg{msg: req}).Return(nil)
	addedEndpoints := []*service.Endpoint{
		{Address: &common.Address{Ip: "127.0.0.1", Port: 8888}},
		{Address: &common.Address{Ip: "127.0.0.1", Port: 8889}},
	}
	removedEndpoints := []*service.Endpoint{
		{Address: &common.Address{Ip: "127.0.0.1", Port: 9000}},
		{Address: &common.Address{Ip: "127.0.0.1", Port: 9001}},
	}
	stream.EXPECT().Recv().Return(&api.SvcEndpointDiscoveryResponse{
		SvcName: "foo",
		Added:   addedEndpoints,
		Removed: removedEndpoints,
	}, nil)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		<-quit
		return nil, io.EOF
	})
	c := NewMockDiscoveryServiceClient(ctrl)
	c.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	// mock discovery service client facotory
	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	d, err := newDynamicSource(b)
	assert.NoError(t, err)

	hookCalled := false
	d.SetSvcEndpointHook(func(svcName string, added, removed []*service.Endpoint) {
		hookCalled = true
		assert.Equal(t, "foo", svcName)
		assert.Equal(t, addedEndpoints, added)
		assert.Equal(t, removedEndpoints, removed)
	})
	d.subscribeSvc(&service.Service{Name: "foo"})
	d.unsubscribeSvc(&service.Service{Name: "bar"})
	d.unsubscribeSvc(&service.Service{Name: "zoo"})

	// close the discovery service client
	time.AfterFunc(time.Millisecond*100, func() {
		close(quit)
		close(d.quit)
	})
	d.streamSvcEndpoints(context.Background())
	assert.True(t, hookCalled)
}

func TestStopDynamicSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock
	c := NewMockDiscoveryServiceClient(ctrl)
	err := errors.New("internal error")
	c.EXPECT().StreamDependency(gomock.Any(), gomock.Any()).Return(nil, err).AnyTimes()
	c.EXPECT().StreamSvcConfigs(gomock.Any()).Return(nil, err).AnyTimes()
	c.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(nil, err).AnyTimes()

	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	b := new(bootstrap.Bootstrap)
	d, err := newDynamicSource(b)
	assert.NoError(t, err)

	done := make(chan struct{})
	defer func() {
		<-done
	}()
	go func() {
		defer close(done)
		d.Serve()
	}()

	select {
	case <-done:
		t.Error("expect serving, but stopped")
		return
	case <-time.After(time.Millisecond * 500):
	}
	d.Stop()
}
