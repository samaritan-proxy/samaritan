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
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gomock "github.com/golang/mock/gomock"
	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/stretchr/testify/assert"
)

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

func mockInstance() *common.Instance {
	return &common.Instance{
		Id:      "test",
		Version: "1.0.0",
		Belong:  "sash",
	}
}

func TestDependencyDiscoveryClientCreateStreamFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamDependencies(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("internal error"))
	c := newDependencyDiscoveryClient(stub)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx, mockInstance())
}

func TestDependencyDiscoveryClientRecvFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamDependenciesClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})
	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamDependencies(gomock.Any(), gomock.Any()).
		Return(stream, nil)

	c := newDependencyDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx, mockInstance())
}

func makeDependencyDiscoveryResponse(addedSvcNames, removedSvcNames []string) *api.DependencyDiscoveryResponse {
	var added, removed []*service.Service
	for _, svcName := range addedSvcNames {
		added = append(added, &service.Service{Name: svcName})
	}
	for _, svcName := range removedSvcNames {
		removed = append(removed, &service.Service{Name: svcName})
	}
	return &api.DependencyDiscoveryResponse{
		Added:   added,
		Removed: removed,
	}
}

func TestDependencyDiscoveryClientHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamDependenciesClient(ctrl)
	recvCallTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		recvCallTimes++
		// first call
		if recvCallTimes == 1 {
			return makeDependencyDiscoveryResponse([]string{"foo"}, []string{"bar"}), nil
		}
		// second call
		<-ctx.Done()
		return nil, io.EOF
	}).Times(2)

	stub := NewMockDiscoveryServiceClient(ctrl)
	req := &api.DependencyDiscoveryRequest{Instance: mockInstance()}
	stub.EXPECT().StreamDependencies(gomock.Any(), &rpcMsg{req}).
		Return(stream, nil)

	added := make(map[string]struct{})
	removed := make(map[string]struct{})
	hook := func(addedSvcs, removedSvcs []*service.Service) {
		for _, svc := range addedSvcs {
			added[svc.Name] = struct{}{}
		}
		for _, svc := range removedSvcs {
			removed[svc.Name] = struct{}{}
		}
	}

	c := newDependencyDiscoveryClient(stub)
	c.SetHook(hook)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx, mockInstance())

	// assert hook execution
	assert.Contains(t, added, "foo")
	assert.Contains(t, removed, "bar")
}

func TestDependencyDiscoveryClientAutoRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamDependenciesClient(ctrl)
	recvCallTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		recvCallTimes++
		// first call
		if recvCallTimes == 1 {
			return nil, io.EOF
		}
		<-ctx.Done()
		return nil, io.EOF
	}).Times(2)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamDependencies(gomock.Any(), gomock.Any()).
		Return(stream, nil).Times(2)

	c := newDependencyDiscoveryClient(stub)
	time.AfterFunc(time.Second*2, cancel)
	c.Run(ctx, mockInstance())
}

func TestSvcConfigDiscoveryClientCreateStreamFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("internal error"))
	c := newSvcConfigDiscoveryClient(stub)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcConfigDiscoveryClientRecvFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	c := newSvcConfigDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcConfigDiscoveryClientSendFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	stream.EXPECT().Send(gomock.Any()).DoAndReturn(func(*api.SvcConfigDiscoveryRequest) error {
		cancel()
		return io.ErrShortWrite
	})
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	c := newSvcConfigDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, func() {
		c.Subscribe("foo")
	})
	c.Run(ctx)
}

func makeSvcConfigDiscoveryRequest(subscribed, unsubscribed []string) *api.SvcConfigDiscoveryRequest {
	return &api.SvcConfigDiscoveryRequest{
		SvcNamesSubscribe:   subscribed,
		SvcNamesUnsubscribe: unsubscribed,
	}
}

func makeSvcConfigDiscoveryResponse(configs map[string]*service.Config) *api.SvcConfigDiscoveryResponse {
	return &api.SvcConfigDiscoveryResponse{
		Updated: configs,
	}
}

func TestSvcConfigDiscoveryClientAutoResubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	stream.EXPECT().Send(&rpcMsg{makeSvcConfigDiscoveryRequest([]string{"foo"}, nil)}).Return(io.ErrShortWrite)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	c := newSvcConfigDiscoveryClient(stub)
	c.Subscribe("foo")
	c.Unsubscribe("bar")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcConfigDiscoveryClientHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	recvCallTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		recvCallTimes++
		if recvCallTimes == 1 {
			return makeSvcConfigDiscoveryResponse(map[string]*service.Config{"foo": nil}), nil
		}
		<-ctx.Done()
		return nil, io.EOF
	}).Times(2)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	updated := make(map[string]*service.Config)
	hook := func(svcName string, svcConfig *service.Config) {
		updated[svcName] = svcConfig
	}

	c := newSvcConfigDiscoveryClient(stub)
	c.SetHook(hook)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)

	// assert hook execution
	assert.Contains(t, updated, "foo")
}

func TestSvcConfigDiscoveryClientSubAndUnsub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})
	stream.EXPECT().Send(&rpcMsg{makeSvcConfigDiscoveryRequest([]string{"foo"}, nil)}).Return(nil)
	stream.EXPECT().Send(&rpcMsg{makeSvcConfigDiscoveryRequest(nil, []string{"foo"})}).Return(nil)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(stream, nil)

	c := newSvcConfigDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, func() {
		c.Subscribe("foo")
		time.Sleep(time.Millisecond * 100)
		c.Unsubscribe("foo")
		time.Sleep(time.Millisecond * 100)
		cancel()
	})
	c.Run(ctx)
}
func TestSvcEndpointDiscoveryClientCreateStreamFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any(), gomock.Any()).
		Return(nil, io.ErrUnexpectedEOF)
	c := newSvcEndpointDiscoveryClient(stub)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcEndpointDiscoveryClientRecvFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	c := newSvcEndpointDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcEndpointDiscoveryClientSendFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	stream.EXPECT().Send(gomock.Any()).DoAndReturn(func(*api.SvcEndpointDiscoveryRequest) error {
		cancel()
		return io.ErrShortWrite
	})
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	c := newSvcEndpointDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, func() {
		c.Subscribe("foo")
	})
	c.Run(ctx)
}

func makeSvcEndpointDiscoveryRequest(subscribed, unsubscribed []string) *api.SvcEndpointDiscoveryRequest {
	return &api.SvcEndpointDiscoveryRequest{
		SvcNamesSubscribe:   subscribed,
		SvcNamesUnsubscribe: unsubscribed,
	}
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

func TestSvcEndpointDiscoveryClientAutoResubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	stream.EXPECT().Send(&rpcMsg{makeSvcEndpointDiscoveryRequest([]string{"foo"}, nil)}).Return(io.ErrShortWrite)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	c := newSvcEndpointDiscoveryClient(stub)
	c.Subscribe("foo")
	c.Unsubscribe("bar")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)
}

func TestSvcEndpointDiscoveryClientHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	recvCallTimes := 0
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		recvCallTimes++
		if recvCallTimes == 1 {
			resp := makeSvcEndpointDiscoveryResponse(
				"foo",
				[]*service.Endpoint{makeEndpoint("127.0.0.1", 8888)},
				[]*service.Endpoint{makeEndpoint("127.0.0.1", 9999)},
			)
			return resp, nil
		}
		<-ctx.Done()
		return nil, io.EOF
	}).Times(2)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	var added, removed map[string]struct{}
	hook := func(svcName string, addedEts, removedEts []*service.Endpoint) {
		toMap := func(ets []*service.Endpoint) map[string]struct{} {
			m := make(map[string]struct{})
			for _, et := range ets {
				addr := fmt.Sprintf("%s:%d", et.Address.Ip, et.Address.Port)
				m[addr] = struct{}{}
			}
			return m
		}
		added = toMap(addedEts)
		removed = toMap(removedEts)
	}

	c := newSvcEndpointDiscoveryClient(stub)
	c.SetHook(hook)
	time.AfterFunc(time.Millisecond*100, cancel)
	c.Run(ctx)

	// assert hook execution
	assert.Equal(t, 1, len(added))
	assert.Contains(t, added, "127.0.0.1:8888")
	assert.Equal(t, 1, len(removed))
	assert.Contains(t, removed, "127.0.0.1:9999")
}

func TestSvcEndpointDiscoveryClientSubAndUnsub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())

	stream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})
	stream.EXPECT().Send(&rpcMsg{makeSvcEndpointDiscoveryRequest([]string{"foo"}, nil)}).Return(nil)
	stream.EXPECT().Send(&rpcMsg{makeSvcEndpointDiscoveryRequest(nil, []string{"foo"})}).Return(nil)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(stream, nil)

	c := newSvcEndpointDiscoveryClient(stub)
	time.AfterFunc(time.Millisecond*100, func() {
		c.Subscribe("foo")
		time.Sleep(time.Millisecond * 100)
		c.Unsubscribe("foo")
		time.Sleep(time.Millisecond * 100)
		cancel()
	})
	c.Run(ctx)
}

func TestDiscoveryClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())

	dependencyStream := NewMockDiscoveryService_StreamDependenciesClient(ctrl)
	recvCallTimes := 0
	dependencyStream.EXPECT().Recv().DoAndReturn(func() (*api.DependencyDiscoveryResponse, error) {
		recvCallTimes++
		switch recvCallTimes {
		case 1:
			time.Sleep(time.Millisecond * 50)
			return makeDependencyDiscoveryResponse([]string{"foo"}, nil), nil
		case 2:
			time.Sleep(time.Millisecond * 50)
			return makeDependencyDiscoveryResponse(nil, []string{"foo"}), nil
		default:
			<-ctx.Done()
			return nil, io.EOF
		}
	}).Times(3)

	svcConfigStream := NewMockDiscoveryService_StreamSvcConfigsClient(ctrl)
	svcConfigStream.EXPECT().Recv().DoAndReturn(func() (*api.SvcConfigDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})
	svcConfigStream.EXPECT().Send(&rpcMsg{makeSvcConfigDiscoveryRequest([]string{"foo"}, nil)}).Return(nil)
	svcConfigStream.EXPECT().Send(&rpcMsg{makeSvcConfigDiscoveryRequest(nil, []string{"foo"})}).Return(nil)

	svcEndpointStream := NewMockDiscoveryService_StreamSvcEndpointsClient(ctrl)
	svcEndpointStream.EXPECT().Recv().DoAndReturn(func() (*api.SvcEndpointDiscoveryResponse, error) {
		<-ctx.Done()
		return nil, io.EOF
	})
	svcEndpointStream.EXPECT().Send(&rpcMsg{makeSvcEndpointDiscoveryRequest([]string{"foo"}, nil)}).Return(nil)
	svcEndpointStream.EXPECT().Send(&rpcMsg{makeSvcEndpointDiscoveryRequest(nil, []string{"foo"})}).Return(nil)

	stub := NewMockDiscoveryServiceClient(ctrl)
	stub.EXPECT().StreamDependencies(gomock.Any(), gomock.Any()).Return(dependencyStream, nil)
	stub.EXPECT().StreamSvcConfigs(gomock.Any()).Return(svcConfigStream, nil)
	stub.EXPECT().StreamSvcEndpoints(gomock.Any()).Return(svcEndpointStream, nil)

	var dependHookCalled bool
	dependHook := func(added, removed []*service.Service) { dependHookCalled = true }

	c := newDiscoveryClient(stub)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		c.StreamDependencies(ctx, mockInstance(), dependHook)
	}()
	go func() {
		defer wg.Done()
		c.StreamSvcConfigs(ctx, nil)
	}()
	go func() {
		defer wg.Done()
		c.StreamSvcEndpoints(ctx, nil)
	}()

	time.AfterFunc(time.Second, cancel)
	wg.Wait()

	// assert hook execution
	assert.True(t, dependHookCalled)
}
