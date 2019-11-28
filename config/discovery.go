package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samaritan-proxy/samaritan/pb/api"
	"github.com/samaritan-proxy/samaritan/pb/common"
)

type dependencyDiscoveryClient struct {
	api.DiscoveryServiceClient
}

func newDependencyDiscoveryClient(client api.DiscoveryServiceClient) *dependencyDiscoveryClient {
	return &dependencyDiscoveryClient{
		DiscoveryServiceClient: client,
	}
}

func (c *dependencyDiscoveryClient) Run(ctx context.Context, inst *common.Instance) {
	for {
		c.run(ctx, inst)

		t := time.NewTimer(time.Millisecond * 100)
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
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return
		}

		fmt.Println(resp.Added, resp.Removed)
	}
}

type svcDiscoveryStream interface {
	Send(subscribed, unsubscribed []string) error
	Recv() error
}

type svcDiscoveryStreamMaker func(ctx context.Context) (svcDiscoveryStream, error)

type svcDiscoveryClient struct {
	sync.RWMutex

	subscribed map[string]struct{}
	subCh      chan string
	unsubCh    chan string

	newStream svcDiscoveryStreamMaker
}

func newSvcDiscoveryClient(streamMaker svcDiscoveryStreamMaker) *svcDiscoveryClient {
	return &svcDiscoveryClient{
		subscribed: make(map[string]struct{}, 8),
		subCh:      make(chan string, 8),
		unsubCh:    make(chan string, 8),
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
	for {
		c.run(ctx)

		t := time.NewTimer(time.Millisecond * 100)
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
		return
	}

	// resubscribe the services.
	if err := c.resubscribe(stream); err != nil {
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
	return
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
		case _ = <-c.subCh:
		default:
			return
		}
	}
}

func (c *svcDiscoveryClient) cleanUnsubChLocked() {
	for {
		select {
		case _ = <-c.unsubCh:
		default:
			return
		}
	}
}

func (c *svcDiscoveryClient) loopRecv(stream svcDiscoveryStream) {
	for {
		if err := stream.Recv(); err != nil {
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
			return
		}
	}
}

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
	fmt.Println(resp)
	return nil
}

type svcConfigDiscoveryClient struct {
	api.DiscoveryServiceClient
	*svcDiscoveryClient
}

func newSvcConfigDiscoveryClient(raw api.DiscoveryServiceClient) *svcConfigDiscoveryClient {
	c := &svcConfigDiscoveryClient{
		DiscoveryServiceClient: raw,
	}
	base := newSvcDiscoveryClient(c.newStream)
	c.svcDiscoveryClient = base
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
	fmt.Println(resp)
	return nil
}

type svcEndpointDiscoveryClient struct {
	api.DiscoveryServiceClient
	*svcDiscoveryClient
}

func newSvcEndpointDiscoveryClient(raw api.DiscoveryServiceClient) *svcEndpointDiscoveryClient {
	c := &svcEndpointDiscoveryClient{
		DiscoveryServiceClient: raw,
	}
	base := newSvcDiscoveryClient(c.newStream)
	c.svcDiscoveryClient = base
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
