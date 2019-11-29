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
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
)

type mockDiscoveryServiceServer struct{}

func newMockDiscoveryServiceServer() api.DiscoveryServiceServer {
	return &mockDiscoveryServiceServer{}
}

func (s *mockDiscoveryServiceServer) StreamDependencies(_ *api.DependencyDiscoveryRequest, stream api.DiscoveryService_StreamDependenciesServer) error {
	// make it block
	err := stream.RecvMsg(new(api.DependencyDiscoveryRequest))
	return err
}

func (s *mockDiscoveryServiceServer) StreamSvcConfigs(stream api.DiscoveryService_StreamSvcConfigsServer) error {
	// make it block
	_, err := stream.Recv()
	return err
}

func (s *mockDiscoveryServiceServer) StreamSvcEndpoints(stream api.DiscoveryService_StreamSvcEndpointsServer) error {
	// make it block
	_, err := stream.Recv()
	return err
}

func TestDynamicSourceServe(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	s := grpc.NewServer()
	api.RegisterDiscoveryServiceServer(s, newMockDiscoveryServiceServer())
	go func() {
		s.Serve(l) //nolint:errcheck
	}()

	config := &bootstrap.ConfigSource{}
	config.Endpoint = l.Addr().String()
	b := &bootstrap.Bootstrap{
		Instance:            mockInstance(),
		DynamicSourceConfig: config,
	}

	d, err := newDynamicSource(b)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		d.Serve()
	}()
	time.AfterFunc(time.Millisecond*500, d.Stop)
	<-done
}
