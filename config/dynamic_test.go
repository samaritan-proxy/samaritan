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
	return r.msg.String()
}

func newTestInstance() *common.Instance {
	return &common.Instance{
		Id:      "test",
		Version: "1.0.0",
		Belong:  "sam",
	}
}

func makeDependencyDiscoveryResponse(added, removed []*service.Service) *api.DependencyDiscoveryResponse {
	return &api.DependencyDiscoveryResponse{
		Added:   added,
		Removed: removed,
	}
}

func makeService(name string) *service.Service {
	return &service.Service{
		Name: name,
	}
}

func TestDynamicSourceStreamSvcs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := new(bootstrap.Bootstrap)
	b.Instance = newTestInstance()

	addedSvcs := []*service.Service{makeService("foo")}
	removedSvcs := []*service.Service{makeService("bar")}

	// mock stream
	stream := NewMockDiscoveryService_StreamDependenciesClient(ctrl)
	streamQuitCh := make(chan struct{})
	abortStream := func() { close(streamQuitCh) }
	recvTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		recvTimes++
		if recvTimes < 2 {
			return makeDependencyDiscoveryResponse(addedSvcs, removedSvcs), nil
		}
		// wait the stream closed
		<-streamQuitCh
		return nil, io.EOF
	}).Times(2)

	// mock client
	c := NewMockDiscoveryServiceClient(ctrl)
	req := &api.DependencyDiscoveryRequest{Instance: b.Instance}
	c.EXPECT().StreamDependencies(gomock.Any(), &rpcMsg{msg: req}).Return(stream, nil)

	// mock client facotory
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

	// abort stream
	time.AfterFunc(time.Millisecond*100, abortStream)
	d.streamSvcs(context.Background())

	// assert hooks
	assert.True(t, svcHookCalled)
	assert.True(t, svcSubHookCalled)
	assert.True(t, svcUnsubHookCalled)
}

func makeSvcConfigDiscoveryResponse(configs map[string]*service.Config) *api.SvcConfigDiscoveryResponse {
	return &api.SvcConfigDiscoveryResponse{
		Updated: configs,
	}
}

func TestDynamicSourceStreamSvcConfigs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock stream
	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	streamQuitCh := make(chan struct{})
	abortStream := func() {
		close(streamQuitCh)
	}
	// mock send method
	req := &api.SvcConfigDiscoveryRequest{
		SvcNamesSubscribe:   []string{"foo"},
		SvcNamesUnsubscribe: []string{"bar", "zoo"},
	}
	stream.EXPECT().Send(&rpcMsg{msg: req}).Return(nil)
	// mock recv method
	recvTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		recvTimes++
		if recvTimes < 2 {
			return makeSvcConfigDiscoveryResponse(map[string]*service.Config{"foo": nil}), nil
		}
		<-streamQuitCh
		return nil, io.EOF
	}).Times(2)

	// mock client
	c := NewMockDiscoveryServiceClient(ctrl)
	c.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	// mock discovery service client facotory
	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	b := new(bootstrap.Bootstrap)
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
	time.AfterFunc(time.Millisecond*100, abortStream)
	d.streamSvcConfigs(context.Background())
	assert.True(t, hookCalled)
}

func makeEndpoint(ip string, port uint32) *service.Endpoint {
	return &service.Endpoint{
		Address: &common.Address{
			Ip:   ip,
			Port: port,
		},
	}
}

func makeSvcEndpointDiscoveryResponse(svcName string, added, removed []*service.Endpoint) *api.SvcEndpointDiscoveryResponse {
	return &api.SvcEndpointDiscoveryResponse{
		SvcName: svcName,
		Added:   added,
		Removed: removed,
	}
}

func TestDynamicSourceStreamSvcEndpoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	b := new(bootstrap.Bootstrap)
	addedEndpoints := []*service.Endpoint{
		makeEndpoint("127.0.0.1", 8888),
		makeEndpoint("127.0.0.1", 8889),
	}
	removedEndpoints := []*service.Endpoint{
		makeEndpoint("127.0.0.1", 9000),
		makeEndpoint("127.0.0.1", 9001),
	}

	// mock stream
	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	streamQuitCh := make(chan struct{})
	abortStream := func() {
		close(streamQuitCh)
	}
	// mock send method
	req := &api.SvcEndpointDiscoveryRequest{
		SvcNamesSubscribe:   []string{"foo"},
		SvcNamesUnsubscribe: []string{"bar", "zoo"},
	}
	stream.EXPECT().Send(&rpcMsg{msg: req}).Return(nil)
	// mock recv method
	recvTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		recvTimes++
		if recvTimes < 2 {
			return makeSvcEndpointDiscoveryResponse("foo", addedEndpoints, removedEndpoints), nil
		}
		<-streamQuitCh
		return nil, io.EOF
	}).Times(2)

	// mock client
	c := NewMockDiscoveryServiceClient(ctrl)
	c.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	// mock client factory
	factory := newDiscoveryServiceClientFacotry(c, nil)
	rollback := mockNewDiscoveryServiceClient(factory)
	defer rollback()

	d, err := newDynamicSource(b)
	assert.NoError(t, err)

	// register hook
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
	time.AfterFunc(time.Millisecond*100, abortStream)
	d.streamSvcEndpoints(context.Background())
	assert.True(t, hookCalled)
}

func TestStopDynamicSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock
	c := NewMockDiscoveryServiceClient(ctrl)
	err := errors.New("internal error")
	c.EXPECT().StreamDependencies(gomock.Any(), gomock.Any()).Return(nil, err).AnyTimes()
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
