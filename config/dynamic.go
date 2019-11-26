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
//go:generate mockgen -package $GOPACKAGE --destination ./mock_discovery_test.go $REPO_URI/pb/api DiscoveryServiceClient,DiscoveryService_StreamSvcsClient,DiscoveryService_StreamSvcConfigsClient,DiscoveryService_StreamSvcEndpointsClient

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

var _ DynamicSource = new(dynamicSource)

type svcHook func(added, removed []*service.Service)
type svcConfigHook func(svcName string, newCfg *service.Config)
type svcEndpointHook func(svcName string, added, removed []*service.Endpoint)

// DynamicSource represents the dynamic config source.
type DynamicSource interface {
	// SetSvcHook sets a hook which will be called
	// when a service is added or removed. It must be
	// called before Serve.
	SetSvcHook(hook svcHook)
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
	b         *bootstrap.Bootstrap
	c         api.DiscoveryServiceClient
	cShutdown func() error

	svcCfgSubCh chan *service.Service
	svcEtSubCh  chan *service.Service
	svcSubHook  func(*service.Service) // it's only used for testing.

	svcCfgUnsubCh chan *service.Service
	svcEtUnsubCh  chan *service.Service
	svcUnsubHook  func(*service.Service) // it's only used for testing.

	svcHook    svcHook
	svcCfgHook svcConfigHook
	svcEtHook  svcEndpointHook

	quit chan struct{}
	done chan struct{}
}

var newDiscoveryServiceClient = func(cfg *bootstrap.ConfigSource) (c api.DiscoveryServiceClient, shutdown func() error, err error) {
	target := cfg.Endpoint
	// TODO: support Authentication
	cc, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	c = api.NewDiscoveryServiceClient(cc)
	return c, cc.Close, nil
}

func newDynamicSource(b *bootstrap.Bootstrap) (*dynamicSource, error) {
	c, shutdown, err := newDiscoveryServiceClient(b.DynamicSourceConfig)
	if err != nil {
		return nil, err
	}
	d := &dynamicSource{
		b:             b,
		c:             c,
		cShutdown:     shutdown,
		svcCfgSubCh:   make(chan *service.Service, 16),
		svcEtSubCh:    make(chan *service.Service, 16),
		svcCfgUnsubCh: make(chan *service.Service, 16),
		svcEtUnsubCh:  make(chan *service.Service, 16),
		quit:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	return d, nil
}

func (d *dynamicSource) SetSvcHook(hook svcHook) {
	d.svcHook = hook
}

func (d *dynamicSource) SetSvcConfigHook(hook svcConfigHook) {
	d.svcCfgHook = hook
}

func (d *dynamicSource) SetSvcEndpointHook(hook svcEndpointHook) {
	d.svcEtHook = hook
}

func (d *dynamicSource) Serve() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.StreamSvcs()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.StreamSvcConfigs()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.StreamSvcEndpoints()
	}()

	<-d.quit
	// shutdown the discovery service client
	d.cShutdown()
	wg.Wait()
	close(d.done)
}

func (d *dynamicSource) Stop() {
	close(d.quit)
	<-d.done
}

func (d *dynamicSource) StreamSvcs() {
	defer logger.Debugf("StreamSvcs done")
	for {
		d.streamSvcs(context.Background())
		select {
		case <-d.quit:
			return
		default:
		}
		logger.Warnf("StreamSvcs failed, retrying...")

		// TODO: exponential backoff
		t := time.NewTimer(time.Millisecond * 100)
		select {
		case <-t.C:
		case <-d.quit:
			return
		}
	}
}

func (d *dynamicSource) streamSvcs(ctx context.Context) {
	req := &api.SvcDiscoveryRequest{
		Instance: d.b.Instance,
	}
	stream, err := d.c.StreamSvcs(ctx, req)
	if err != nil {
		logger.Warnf("Fail to create stream client: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			logger.Warnf("Recv failed: %v", err)
			return
		}

		// TODO: validate resp
		if d.svcHook != nil {
			d.svcHook(resp.Added, resp.Removed)
		}

		// subscribe
		for _, svc := range resp.Added {
			d.subscribeSvc(svc)
		}
		// unsubscribe
		for _, svc := range resp.Removed {
			d.unsubscribeSvc(svc)
		}
	}
}

// subscribeSvc subscribes the config and endpoint change of specified service.
func (d *dynamicSource) subscribeSvc(svc *service.Service) {
	d.svcCfgSubCh <- svc
	d.svcEtSubCh <- svc
	if d.svcSubHook != nil {
		d.svcSubHook(svc)
	}
}

// unsubscribeSvc unsubscribes the config and endpoint change of specified service.
func (d *dynamicSource) unsubscribeSvc(svc *service.Service) {
	d.svcCfgUnsubCh <- svc
	d.svcEtUnsubCh <- svc
	if d.svcUnsubHook != nil {
		d.svcUnsubHook(svc)
	}
}

func (d *dynamicSource) StreamSvcConfigs() {
	defer func() {
		logger.Debugf("StreamSvcConfigs done")
	}()

	for {
		d.streamSvcConfigs(context.Background())
		select {
		case <-d.quit:
			return
		default:
		}
		logger.Warnf("StreamSvcConfigs failed, retrying...")

		// TODO: exponential backoff
		t := time.NewTimer(time.Millisecond * 100)
		select {
		case <-t.C:
		case <-d.quit:
			return
		}
	}
}

func (d *dynamicSource) streamSvcConfigs(ctx context.Context) {
	// create stream client
	stream, err := d.c.StreamSvcConfigs(ctx)
	if err != nil {
		logger.Warnf("Fail to create stream client: %v", err)
		return
	}

	recvDone := make(chan struct{})
	defer func() {
		// wait recv goroutine done
		<-recvDone
	}()

	go func() {
		defer close(recvDone)
		for {
			resp, err := stream.Recv()
			if err != nil {
				logger.Warnf("Recv failed: %v", err)
				return
			}

			// TODO: validate resp
			if d.svcCfgHook == nil {
				continue
			}
			for svcName, svcConfig := range resp.Updated {
				d.svcCfgHook(svcName, svcConfig)
			}
		}
	}()

	for {
		var subscribe, unsubscribe []string
		select {
		case svc := <-d.svcCfgSubCh:
			subscribe = append(subscribe, svc.Name)
		case svc := <-d.svcCfgUnsubCh:
			unsubscribe = append(unsubscribe, svc.Name)
		case <-recvDone:
			return
		}

		// batch
		// TODO: limit the size
		for {
			select {
			case svc := <-d.svcCfgSubCh:
				subscribe = append(subscribe, svc.Name)
			case svc := <-d.svcCfgUnsubCh:
				unsubscribe = append(unsubscribe, svc.Name)
			case <-recvDone:
				return
			default:
				goto SEND
			}
		}

	SEND:
		req := &api.SvcConfigDiscoveryRequest{
			SvcNamesSubscribe:   subscribe,
			SvcNamesUnsubscribe: unsubscribe,
		}
		err := stream.Send(req)
		if err != nil {
			logger.Warnf("Send failed: %v", err)
			return
		}
	}
}

func (d *dynamicSource) StreamSvcEndpoints() {
	defer func() {
		logger.Debugf("StreamSvcEndpoints done")
	}()

	for {
		d.streamSvcEndpoints(context.Background())
		select {
		case <-d.quit:
			return
		default:
		}
		logger.Warnf("StreamSvcEndpoints failed, retrying...")

		// TODO: exponential backoff
		t := time.NewTimer(time.Millisecond * 100)
		select {
		case <-t.C:
		case <-d.quit:
			return
		}
	}
}

func (d *dynamicSource) streamSvcEndpoints(ctx context.Context) {
	// make the stream client
	stream, err := d.c.StreamSvcEndpoints(ctx)
	if err != nil {
		logger.Warnf("Fail to create stream client: %v", err)
		return
	}

	recvDone := make(chan struct{})
	defer func() {
		// wait recv goroutine done
		<-recvDone
	}()

	go func() {
		defer close(recvDone)
		for {
			resp, err := stream.Recv()
			if err != nil {
				logger.Warnf("Recv failed: %v", err)
				return
			}

			// TODO: validate resp
			if d.svcEtHook != nil {
				d.svcEtHook(resp.SvcName, resp.Added, resp.Removed)
			}
		}
	}()

	for {
		var subscribe, unsubscribe []string
		select {
		case svc := <-d.svcEtSubCh:
			subscribe = append(subscribe, svc.Name)
		case svc := <-d.svcEtUnsubCh:
			unsubscribe = append(unsubscribe, svc.Name)
		case <-recvDone:
			return
		}

		// batch
		// TODO: limit the size
		for {
			select {
			case svc := <-d.svcEtSubCh:
				subscribe = append(subscribe, svc.Name)
			case svc := <-d.svcEtUnsubCh:
				unsubscribe = append(unsubscribe, svc.Name)
			case <-recvDone:
				return
			default:
				goto SEND
			}
		}

	SEND:
		req := &api.SvcEndpointDiscoveryRequest{
			SvcNamesSubscribe:   subscribe,
			SvcNamesUnsubscribe: unsubscribe,
		}
		err := stream.Send(req)
		if err != nil {
			logger.Warnf("Send failed: %v", err)
			return
		}
	}
}
