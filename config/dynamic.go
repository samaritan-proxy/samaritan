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

// FIXME: mockgen can't handle cycle imports in reflect mode when outside of GOPATH currently,
// so add self_package parameter temporarily. Refer to: https://github.com/golang/mock/issues/310
//go:generate mockgen -package $GOPACKAGE -self_package $REPO_URI/$GOPACKAGE --destination ./mock_dynamic_test.go $REPO_URI/$GOPACKAGE  DynamicSource
//go:generate mockgen -package $GOPACKAGE --destination ./mock_discovery_test.go $REPO_URI/pb/api DiscoveryServiceClient,DiscoveryService_StreamDependenciesClient,DiscoveryService_StreamSvcConfigsClient,DiscoveryService_StreamSvcEndpointsClient

import (
	"context"
	"sync"
	"time"

	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var _ DynamicSource = new(dynamicSource)

type dependencyHook func(added, removed []*service.Service)
type svcConfigHook func(svcName string, newCfg *service.Config)
type svcEndpointHook func(svcName string, added, removed []*service.Endpoint)

// DynamicSource represents the dynamic config source.
type DynamicSource interface {
	// SetSvcHook sets a hook which will be called
	// when a service is added or removed. It must be
	// called before Serve.
	SetDependencyHook(hook dependencyHook)
	// SetSvcConfigHook sets a hook which wil be called
	// when the proxy config of subscribed service update.
	// It must be called before Serve.
	SetSvcConfigHook(hook svcConfigHook)
	// SetSvcEndpointHook sets a hook which will be called
	// when the endpoints of subscribed service updated.
	// It must be called before Serve.
	SetSvcEndpointHook(hook svcEndpointHook)

	// Serve starts observing the config update from remote.
	// It will return until Stop is called.
	Serve()
	// Stop stops observing the config update from remote.
	Stop()
}

type dynamicSource struct {
	b *bootstrap.Bootstrap

	conn *grpc.ClientConn
	c    *discoveryClient

	dependHook dependencyHook
	svcCfgHook svcConfigHook
	svcEtHook  svcEndpointHook

	quit chan struct{}
	done chan struct{}
}

func newDynamicSource(b *bootstrap.Bootstrap) (*dynamicSource, error) {
	d := &dynamicSource{
		b:    b,
		quit: make(chan struct{}),
		done: make(chan struct{}),
	}
	err := d.initDiscoveryClient()
	return d, err
}

func (d *dynamicSource) initDiscoveryClient() error {
	target := d.b.DynamicSourceConfig.Endpoint
	// TODO: support Authentication
	options := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Second * 30,
			Timeout: time.Second * 10,
		}),
	}
	conn, err := grpc.Dial(target, options...)
	if err != nil {
		return err
	}

	d.conn = conn
	stub := api.NewDiscoveryServiceClient(conn)
	d.c = newDiscoveryClient(stub)
	return nil
}

func (d *dynamicSource) SetDependencyHook(hook dependencyHook) {
	d.dependHook = hook
}

func (d *dynamicSource) SetSvcConfigHook(hook svcConfigHook) {
	d.svcCfgHook = hook
}

func (d *dynamicSource) SetSvcEndpointHook(hook svcEndpointHook) {
	d.svcEtHook = hook
}

func (d *dynamicSource) Serve() {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.c.StreamDependencies(ctx, d.b.Instance, d.dependHook)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.c.StreamSvcConfigs(ctx, d.svcCfgHook)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.c.StreamSvcEndpoints(ctx, d.svcEtHook)
	}()

	<-d.quit
	cancel()
	// close grpc client conn
	d.conn.Close()
	wg.Wait()
	close(d.done)
}

func (d *dynamicSource) Stop() {
	close(d.quit)
	<-d.done
}
