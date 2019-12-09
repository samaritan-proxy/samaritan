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
	"math/rand"
	"sync"
	"time"

	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

//go:generate mockgen -package $GOPACKAGE --destination ./mock_discovery_test.go $REPO_URI/pb/api DiscoveryServiceClient,DiscoveryService_StreamDependenciesClient,DiscoveryService_StreamSvcConfigsClient,DiscoveryService_StreamSvcEndpointsClient

type discoveryClient struct {
	dependency  *dependencyDiscoveryClient
	svcConfig   *svcConfigDiscoveryClient
	svcEndpoint *svcEndpointDiscoveryClient
}

func newDiscoveryClient(stub api.DiscoveryServiceClient) *discoveryClient {
	return &discoveryClient{
		dependency:  newDependencyDiscoveryClient(stub),
		svcConfig:   newSvcConfigDiscoveryClient(stub),
		svcEndpoint: newSvcEndpointDiscoveryClient(stub),
	}
}

func (c *discoveryClient) StreamDependencies(ctx context.Context, inst *common.Instance, hook dependencyHook) {
	wrappedHook := func(added, removed []*service.Service) {
		if hook != nil {
			hook(added, removed)
		}
		// subscribe the added services
		for _, svc := range added {
			c.svcConfig.Subscribe(svc.Name)
			c.svcEndpoint.Subscribe(svc.Name)
		}
		// unsubscribe the removed services
		for _, svc := range removed {
			c.svcConfig.Unsubscribe(svc.Name)
			c.svcEndpoint.Unsubscribe(svc.Name)
		}
	}
	impl := c.dependency
	impl.SetHook(wrappedHook)
	impl.Run(ctx, inst)
}

func (c *discoveryClient) StreamSvcConfigs(ctx context.Context, hook svcConfigHook) {
	impl := c.svcConfig
	impl.SetHook(hook)
	impl.Run(ctx)
}

func (c *discoveryClient) StreamSvcEndpoints(ctx context.Context, hook svcEndpointHook) {
	impl := c.svcEndpoint
	impl.SetHook(hook)
	impl.Run(ctx)
}

type dependencyDiscoveryClient struct {
	api.DiscoveryServiceClient

	hook func(added, removed []*service.Service)
}

func newDependencyDiscoveryClient(client api.DiscoveryServiceClient) *dependencyDiscoveryClient {
	return &dependencyDiscoveryClient{
		DiscoveryServiceClient: client,
	}
}

// SetHook must be called before Run.
func (c *dependencyDiscoveryClient) SetHook(hook dependencyHook) {
	c.hook = hook
}

func (c *dependencyDiscoveryClient) Run(ctx context.Context, inst *common.Instance) {
	jitter := 0.2
	baseInterval := time.Second
	for {
		c.run(ctx, inst)
		select {
		case <-ctx.Done():
			return
		default:
		}

		interval := time.Duration((1 + (2*rand.Float64()-1)*jitter) * float64(baseInterval))
		logger.Warnf("The discovery loop of dependencies terminated unexpectedly, retry after %s", interval)
		t := time.NewTimer(interval)
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (c *dependencyDiscoveryClient) run(ctx context.Context, inst *common.Instance) {
	req := &api.DependencyDiscoveryRequest{Instance: inst}
	stream, err := c.StreamDependencies(ctx, req)
	if err != nil {
		logger.Warnf("Fail to create dependencies discovery stream: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			logger.Warnf("Recv from dependencies discovery stream failed: %v", err)
			return
		}

		if c.hook != nil {
			c.hook(resp.Added, resp.Removed)
		}
	}
}

type svcDiscoveryStream interface {
	Send(subscribed, unsubscribed []string) error
	Recv() error
}

type svcDiscoveryStreamMaker func(ctx context.Context) (svcDiscoveryStream, error)

type svcConfigDiscoveryStream struct {
	c   *svcConfigDiscoveryClient
	raw api.DiscoveryService_StreamSvcConfigsClient
}

func (stream *svcConfigDiscoveryStream) Send(subscribed, unsubscribed []string) error {
	req := &api.SvcConfigDiscoveryRequest{
		SvcNamesSubscribe:   subscribed,
		SvcNamesUnsubscribe: unsubscribed,
	}
	return stream.raw.Send(req)
}

func (stream *svcConfigDiscoveryStream) Recv() error {
	resp, err := stream.raw.Recv()
	if err != nil {
		return err
	}

	if stream.c.hook == nil {
		return nil
	}
	for svcName, svcConfig := range resp.Updated {
		stream.c.hook(svcName, svcConfig)
	}
	return nil
}

type svcConfigDiscoveryClient struct {
	api.DiscoveryServiceClient
	*svcDiscoveryClient
	hook svcConfigHook
}

func newSvcConfigDiscoveryClient(raw api.DiscoveryServiceClient) *svcConfigDiscoveryClient {
	c := &svcConfigDiscoveryClient{
		DiscoveryServiceClient: raw,
	}
	impl := newSvcDiscoveryClient("config", c.newStream)
	c.svcDiscoveryClient = impl
	return c
}

func (c *svcConfigDiscoveryClient) newStream(ctx context.Context) (svcDiscoveryStream, error) {
	raw, err := c.StreamSvcConfigs(ctx)
	if err != nil {
		return nil, err
	}
	stream := &svcConfigDiscoveryStream{
		c:   c,
		raw: raw,
	}
	return stream, nil
}

// SetHook must be called before Run.
func (c *svcConfigDiscoveryClient) SetHook(hook svcConfigHook) {
	c.hook = hook
}

type svcEndpointDiscoveryStream struct {
	c   *svcEndpointDiscoveryClient
	raw api.DiscoveryService_StreamSvcEndpointsClient
}

func (stream *svcEndpointDiscoveryStream) Send(subscribed, unsubscribed []string) error {
	req := &api.SvcEndpointDiscoveryRequest{
		SvcNamesSubscribe:   subscribed,
		SvcNamesUnsubscribe: unsubscribed,
	}
	return stream.raw.Send(req)
}

func (stream *svcEndpointDiscoveryStream) Recv() error {
	resp, err := stream.raw.Recv()
	if err != nil {
		return err
	}
	if stream.c.hook != nil {
		stream.c.hook(resp.SvcName, resp.Added, resp.Removed)
	}
	return nil
}

type svcEndpointDiscoveryClient struct {
	api.DiscoveryServiceClient
	*svcDiscoveryClient

	hook svcEndpointHook
}

func newSvcEndpointDiscoveryClient(raw api.DiscoveryServiceClient) *svcEndpointDiscoveryClient {
	c := &svcEndpointDiscoveryClient{
		DiscoveryServiceClient: raw,
	}
	impl := newSvcDiscoveryClient("endpoint", c.newStream)
	c.svcDiscoveryClient = impl
	return c
}

func (c *svcEndpointDiscoveryClient) newStream(ctx context.Context) (svcDiscoveryStream, error) {
	raw, err := c.StreamSvcEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	stream := &svcEndpointDiscoveryStream{
		c:   c,
		raw: raw,
	}
	return stream, nil
}

// SetHook must be called before Run.
func (c *svcEndpointDiscoveryClient) SetHook(hook svcEndpointHook) {
	c.hook = hook
}

type svcDiscoveryClient struct {
	sync.RWMutex
	scope string

	subscribed map[string]struct{}
	subCh      chan string
	unsubCh    chan string

	newStream svcDiscoveryStreamMaker
}

func newSvcDiscoveryClient(scope string, streamMaker svcDiscoveryStreamMaker) *svcDiscoveryClient {
	return &svcDiscoveryClient{
		scope:      scope,
		subscribed: make(map[string]struct{}, 16),
		subCh:      make(chan string, 16),
		unsubCh:    make(chan string, 16),
		newStream:  streamMaker,
	}
}

func (c *svcDiscoveryClient) Subscribe(svcName string) {
	c.Lock()
	defer c.Unlock()
	_, ok := c.subscribed[svcName]
	if ok {
		return
	}
	c.subscribed[svcName] = struct{}{}
	c.subCh <- svcName
}

func (c *svcDiscoveryClient) Unsubscribe(svcName string) {
	c.Lock()
	defer c.Unlock()
	_, ok := c.subscribed[svcName]
	if !ok {
		return
	}
	delete(c.subscribed, svcName)
	c.unsubCh <- svcName
}

func (c *svcDiscoveryClient) Run(ctx context.Context) {
	jitter := 0.2
	baseInterval := time.Second
	for {
		c.run(ctx)
		select {
		case <-ctx.Done():
			return
		default:
		}

		interval := time.Duration((1 + (2*rand.Float64()-1)*jitter) * float64(baseInterval))
		logger.Warnf("The discovery loop of service %s terminated unexpectedly, retry after %s", c.scope, interval)
		t := time.NewTimer(interval)
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (c *svcDiscoveryClient) run(ctx context.Context) {
	stream, err := c.newStream(ctx)
	if err != nil {
		logger.Warnf("Fail to create service %s discovery stream: %v", c.scope, err)
		return
	}

	// resubscribe the services.
	if err := c.resubscribe(stream); err != nil {
		logger.Warnf("Resubscribe services on %s discovery stream failed: %v", c.scope, err)
		return
	}

	recvDone := make(chan struct{})
	defer func() {
		<-recvDone
	}()

	go func() {
		defer close(recvDone)
		c.loopRecv(stream)
	}()
	c.loopSend(stream, recvDone)
}

func (c *svcDiscoveryClient) resubscribe(stream svcDiscoveryStream) error {
	c.RLock()
	// load all subscribed services.
	svcNames := make([]string, 0, len(c.subscribed))
	for svcName := range c.subscribed {
		svcNames = append(svcNames, svcName)
	}
	// clean sub/unsub channel
	c.cleanSubChLocked()
	c.cleanUnsubChLocked()
	c.RUnlock()

	// skip if no subscribed services.
	if len(svcNames) == 0 {
		return nil
	}

	return stream.Send(svcNames, nil)
}

func (c *svcDiscoveryClient) cleanSubChLocked() {
	for {
		select {
		case <-c.subCh:
		default:
			return
		}
	}
}

func (c *svcDiscoveryClient) cleanUnsubChLocked() {
	for {
		select {
		case <-c.unsubCh:
		default:
			return
		}
	}
}

func (c *svcDiscoveryClient) loopRecv(stream svcDiscoveryStream) {
	for {
		if err := stream.Recv(); err != nil {
			logger.Warnf("Recv from service %s stream failed: %v", c.scope, err)
			return
		}
	}
}

func (c *svcDiscoveryClient) loopSend(stream svcDiscoveryStream, stop <-chan struct{}) {
	for {
		var subscribed, unsubscribed []string
		select {
		case svcName := <-c.subCh:
			subscribed = append(subscribed, svcName)
		case svcName := <-c.unsubCh:
			unsubscribed = append(unsubscribed, svcName)
		case <-stop:
			return
		}

		// batch
		for {
			select {
			case svcName := <-c.subCh:
				subscribed = append(subscribed, svcName)
			case svcName := <-c.unsubCh:
				unsubscribed = append(unsubscribed, svcName)
			case <-stop:
				return
			default:
				goto SEND
			}
		}

	SEND:
		err := stream.Send(subscribed, unsubscribed)
		if err != nil {
			logger.Warnf("Send to service %s discovery stream failed: %v", c.scope, err)
			return
		}
	}
}
