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

package redis

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol/redis"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	netutil "github.com/samaritan-proxy/samaritan/proc/internal/net"
	"github.com/samaritan-proxy/samaritan/proc/internal/syscall"
	"github.com/samaritan-proxy/samaritan/proc/redis/hotkey"
)

const (
	slotNum = 16384

	ASKING      = "asking"
	MOVED       = "moved"
	ASK         = "ask"
	CLUSTERDOWN = "clusterdown"
)

var (
	// TODO: refine message
	upstreamExited = "upstream exited"
	backendExited  = "backend exited"
)

type createClientCall struct {
	done chan struct{}
	res  *client
	err  error
}

type upstream struct {
	cfg    *config
	hosts  *host.Set
	logger log.Logger
	stats  *proc.UpstreamStats
	hkc    *hotkey.Collector

	clients           atomic.Value // map[string]*client
	clientsMu         sync.Mutex
	createClientCalls sync.Map // map[string]*createClientCall

	slots               [slotNum]*instance
	slotsRefreshCh      chan struct{}
	slotsRefTriggerHook func() // only used to testing
	slotsLastUpdateTime time.Time

	quit chan struct{}
	done chan struct{}
}

func newUpstream(cfg *config, hosts []*host.Host, logger log.Logger, stats *proc.UpstreamStats) *upstream {
	u := &upstream{
		cfg:            cfg,
		hosts:          host.NewSet(hosts...),
		logger:         logger,
		stats:          stats,
		hkc:            hotkey.NewCollector(50),
		slotsRefreshCh: make(chan struct{}, 1),
		quit:           make(chan struct{}),
		done:           make(chan struct{}),
	}
	u.clients.Store(make(map[string]*client))
	return u
}

func (u *upstream) Serve() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		u.loopRefreshSlots()
	}()
	go func() {
		defer wg.Done()
		u.hkc.Run(u.quit)
	}()
	wg.Wait()

	// stop all clients
	u.clientsMu.Lock()
	clients := u.loadClients()
	for _, c := range clients {
		c.Stop()
	}
	u.clientsMu.Unlock()
	close(u.done)
}

func (u *upstream) Stop() {
	close(u.quit)
	<-u.done
}

func (u *upstream) Hosts() []*host.Host {
	return u.hosts.Healthy()
}

func (u *upstream) HotKeys() []hotkey.HotKey {
	return u.hkc.HotKeys()
}

func (u *upstream) MakeRequest(routingKey []byte, req *simpleRequest) {
	addr, err := u.chooseHost(routingKey, req)
	if err != nil {
		req.SetResponse(newError(err.Error()))
		return
	}
	u.MakeRequestToHost(addr, req)
}

func (u *upstream) chooseHost(routingKey []byte, req *simpleRequest) (string, error) {
	hash := crc16(hashtag(routingKey))
	inst := u.slots[hash&(slotNum-1)]

	// There is no redis instance which is responsible for the specified slot.
	// The possible reasons are as follows:
	// 1) The proxy has just started, and has not yet pulled routing information.
	// 2) The redis cluster has some temporary problems, and will recover automatically later.
	// To avoid these problems, could randomly choose one from the provided seed hosts.
	if inst == nil {
		return u.randomHost()
	}

	if !req.IsReadOnly() {
		return inst.Addr, nil
	}

	// read-only requests
	var candidates []string
	readStrategy := redis.ReadStrategy_MASTER
	if option := u.cfg.GetRedisOption(); option != nil {
		readStrategy = option.ReadStrategy
	}
	switch readStrategy {
	case redis.ReadStrategy_MASTER:
		candidates = append(candidates, inst.Addr)
	case redis.ReadStrategy_BOTH:
		candidates = append(candidates, inst.Addr)
		fallthrough
	case redis.ReadStrategy_REPLICA:
		for _, replica := range inst.Replicas {
			candidates = append(candidates, replica.Addr)
		}
	}

	if len(candidates) == 0 {
		candidates = append(candidates, inst.Addr)
	}
	i := 0
	l := len(candidates)
	if l > 1 {
		// The concurrency-safe rand source does not scale well to multiple cores,
		// so we use currrent unix time in nanoseconds to avoid it. As a result these
		// generated values are sequential rather than random, but are acceptable.
		i = int(time.Now().UnixNano()) % l
	}
	return candidates[i], nil
}

func (u *upstream) MakeRequestToHost(addr string, req *simpleRequest) {
	// request metrics
	u.stats.RqTotal.Inc()
	req.RegisterHook(func(req *simpleRequest) {
		if req.Response().Type == Error {
			u.stats.RqFailureTotal.Inc()
		} else {
			u.stats.RqSuccessTotal.Inc()
		}
		u.stats.RqDurationMs.Record(uint64(req.Duration() / time.Millisecond))
	})

	select {
	case <-u.quit:
		req.SetResponse(newError(upstreamExited))
		return
	default:
	}

	c, err := u.getClient(addr)
	if err != nil {
		req.SetResponse(newError(err.Error()))
		return
	}
	// TODO: detect client status
	c.Send(req)
}

func (u *upstream) getClient(addr string) (*client, error) {
	c, ok := u.loadClients()[addr]
	if ok {
		return c, nil
	}

	// NOTE: fail fast when the addr is unreachable
	v, loaded := u.createClientCalls.LoadOrStore(addr, &createClientCall{
		done: make(chan struct{}),
	})
	call := v.(*createClientCall)
	if loaded {
		<-call.done
		return call.res, call.err
	}
	c, err := u.createClient(addr)
	call.res, call.err = c, err
	close(call.done)
	return c, err
}

func (u *upstream) createClient(addr string) (*client, error) {
	u.clientsMu.Lock()
	defer u.clientsMu.Unlock()

	select {
	case <-u.quit:
		return nil, errors.New(upstreamExited)
	default:
	}
	c, ok := u.loadClients()[addr]
	if ok {
		return c, nil
	}

	conn, err := netutil.Dial("tcp", addr, *u.cfg.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	options := []clientOption{
		withKeyCounter(u.hkc.AllocCounter(addr)),
		withRedirectionCb(u.handleRedirection),
		withClusterDownCb(u.handleClusterDown),
	}
	c, err = newClient(conn, u.cfg, u.logger, options...)
	if err != nil {
		return nil, err
	}

	// start client
	go func() {
		c.Start()
		u.removeClient(addr)
	}()
	u.addClientLocked(addr, c)
	return c, nil
}

func (u *upstream) addClientLocked(addr string, c *client) {
	clone := u.cloneClients()
	clone[addr] = c
	u.updateClients(clone)
}

func (u *upstream) removeClient(addr string) {
	u.clientsMu.Lock()
	defer u.clientsMu.Unlock()
	u.removeClientLocked(addr)
}

func (u *upstream) removeClientLocked(addr string) {
	clone := u.cloneClients()
	delete(clone, addr)
	u.updateClients(clone)
}

func (u *upstream) resetAllClients() {
	old := u.loadClients()

	// set clients to empty
	u.clientsMu.Lock()
	u.updateClients(make(map[string]*client))
	u.clientsMu.Unlock()

	// stop all old clients.
	for _, client := range old {
		client.Stop()
	}
}

func (u *upstream) loadClients() map[string]*client {
	return u.clients.Load().(map[string]*client)
}

func (u *upstream) cloneClients() map[string]*client {
	clients := u.loadClients()
	cpy := make(map[string]*client, len(clients))
	for k, v := range clients {
		cpy[k] = v
	}
	return cpy
}

func (u *upstream) updateClients(clients map[string]*client) {
	u.clients.Store(clients)
}

func (u *upstream) handleRedirection(req *simpleRequest, resp *RespValue) {
	err := strings.Split(string(resp.Text), " ")
	hostAddr := err[2]
	switch strings.ToLower(err[0]) {
	case MOVED:
		u.stats.Counter("moved").Inc()
		u.MakeRequestToHost(hostAddr, req)
	case ASK:
		askingReq := newSimpleRequest(newArray(
			*newBulkString(ASKING),
		))
		u.MakeRequestToHost(hostAddr, askingReq)
		u.MakeRequestToHost(hostAddr, req)
	}
	u.triggerSlotsRefresh()
}

func (u *upstream) handleClusterDown(req *simpleRequest, resp *RespValue) {
	// Usually the cluster is able to recover itself after a CLUSTERDOWN
	// error, so try to request the new slots info.
	u.triggerSlotsRefresh()
	// TODO: add some retry
	req.SetResponse(resp)
}

func (u *upstream) triggerSlotsRefresh() {
	select {
	case u.slotsRefreshCh <- struct{}{}:
	default:
	}

	if u.slotsRefTriggerHook != nil {
		u.slotsRefTriggerHook()
	}
}

var (
	slotsRefFreq = time.Minute * 2
	// slots refresh minum rate is used to prevent excessive refresh requests.
	slotsRefMinRate = 5 * time.Second
)

func (u *upstream) loopRefreshSlots() {
	u.triggerSlotsRefresh() // trigger slots refresh immediately
	for {
		select {
		case <-u.quit:
			return
		case <-time.After(slotsRefFreq):
		case <-u.slotsRefreshCh:
		}

		u.refreshSlots()

		t := time.NewTimer(slotsRefMinRate)
		select {
		case <-t.C:
		case <-u.quit:
			t.Stop()
			return
		}
	}
}

func (u *upstream) refreshSlots() {
	scope := u.stats.NewChild("slots_refresh")
	scope.Counter("total").Inc()
	err := u.doSlotsRefresh()
	if err == nil {
		u.logger.Debugf("refresh slots success")
		scope.Counter("success_total").Inc()
		u.slotsLastUpdateTime = time.Now()
		return
	}

	scope.Counter("failure_total").Inc()
	u.logger.Warnf("fail to refresh slots: %v, will retry...", err)
	// retry
	u.triggerSlotsRefresh()
	return
}

func (u *upstream) randomHost() (string, error) {
	h := u.hosts.Random()
	if h == nil {
		return "", errors.New("no available host")
	}
	return h.Addr, nil
}

func (u *upstream) doSlotsRefresh() error {
	v := newArray(
		*newBulkString("cluster"),
		*newBulkString("nodes"),
	)
	req := newSimpleRequest(v)

	addr, err := u.randomHost()
	if err != nil {
		return err
	}
	u.MakeRequestToHost(addr, req)

	// wait done
	req.Wait()
	resp := req.Response()
	if resp.Type == Error {
		return errors.New(string(resp.Text))
	}
	if resp.Type != BulkString {
		return errInvalidClusterNodes
	}
	insts, err := parseClusterNodes(string(resp.Text))
	if err != nil {
		return err
	}

	// update slots
	for _, inst := range insts {
		for _, slot := range inst.Slots {
			if slot < 0 || slot >= slotNum {
				continue
			}
			// NOTE: it's safe in x86-64 platform.
			u.slots[slot] = inst
		}
	}
	return nil
}

func (u *upstream) OnHostAdd(hosts ...*host.Host) error {
	if len(hosts) == 0 {
		return nil
	}

	u.hosts.Add(hosts...)
	u.triggerSlotsRefresh()
	return nil
}

func (u *upstream) OnHostRemove(hosts ...*host.Host) error {
	if len(hosts) == 0 {
		return nil
	}

	u.hosts.Remove(hosts...)
	// remove the corresponding clients
	clients := u.loadClients()
	for _, h := range hosts {
		client, ok := clients[h.Addr]
		if !ok {
			continue
		}
		client.Stop()
	}

	u.triggerSlotsRefresh()
	return nil
}

func (u *upstream) OnHostReplace(hosts []*host.Host) error {
	if len(hosts) == 0 {
		return nil
	}
	u.hosts.ReplaceAll(hosts)
	u.resetAllClients()
	u.triggerSlotsRefresh()
	return nil
}

type clientOption func(c *client)

func withKeyCounter(counter *hotkey.Counter) clientOption {
	return func(c *client) {
		c.keyCounter = counter
	}
}

func withRedirectionCb(cb func(req *simpleRequest, resp *RespValue)) clientOption {
	return func(c *client) {
		c.onRedirection = cb
	}
}

func withClusterDownCb(cb func(req *simpleRequest, resp *RespValue)) clientOption {
	return func(c *client) {
		c.onClusterDown = cb
	}
}

type client struct {
	cfg    *config
	logger log.Logger
	conn   net.Conn
	enc    *encoder
	dec    *decoder

	filter         *FilterChain
	keyCounter     *hotkey.Counter
	pendingReqs    chan *simpleRequest
	processingReqs chan *simpleRequest
	onRedirection  func(req *simpleRequest, resp *RespValue)
	onClusterDown  func(req *simpleRequest, resp *RespValue)

	quitOnce sync.Once
	quit     chan struct{}
	done     chan struct{}
}

func newClient(conn net.Conn, cfg *config, logger log.Logger, options ...clientOption) (*client, error) {
	// UserTimeout is used to make the client fail fast when the remote peer
	// of eastablised connection crashes without sending FIN packet. It only works
	// on linux platform currently, the underlying mechanism is use TCP_USER_TIMEOUT
	// option to limit the tcp packet retransmission time.
	//
	// 10 seconds is enough in most scenarios, maybe it could be configured in the future.
	userTimeout := time.Second * 10
	if err := syscall.SetTCPUserTimeout(conn, userTimeout); err != nil {
		return nil, err
	}

	c := &client{
		cfg:            cfg,
		logger:         logger,
		conn:           conn,
		enc:            newEncoder(conn, 4096),
		dec:            newDecoder(conn, 8192),
		pendingReqs:    make(chan *simpleRequest, 1024),
		processingReqs: make(chan *simpleRequest, 1024),
		quit:           make(chan struct{}),
		done:           make(chan struct{}),
	}

	// set provided options
	for _, option := range options {
		option(c)
	}

	if err := c.initFilters(); err != nil {
		return nil, err
	}
	// Normally replica nodes will redirect clients to the authoritative master for
	// the hash slot involved in a given command. If want to read queries from these
	// replica nodes, clients must issue READONLY command firstly. Also the READONLY
	// command is harmless to the master node, more details see: https://redis.io/commands/readonly
	readOnlyReq := newSimpleRequest(newStringArray("readonly"))
	c.Send(readOnlyReq)
	return c, nil
}

func (c *client) initFilters() error {
	filters := []Filter{
		newHotKeyFilter(c.keyCounter),
		newCompressFilter(c.cfg),
	}

	chain := newRequestFilterChain()
	for _, filter := range filters {
		chain.AddFilter(filter)
	}
	c.filter = chain
	return nil
}

func (c *client) Start() {
	writeDone := make(chan struct{})
	go func() {
		c.loopWrite()
		c.conn.Close()
		close(writeDone)
	}()

	c.loopRead()
	c.conn.Close()
	c.quitOnce.Do(func() {
		close(c.quit)
	})
	<-writeDone
	c.drainRequests()
	close(c.done)
}

func (c *client) Send(req *simpleRequest) {
	select {
	case <-c.quit:
		req.SetResponse(newError(backendExited))
	default:
		c.pendingReqs <- req
	}
}

func (c *client) loopWrite() {
	var (
		req *simpleRequest
		err error
	)
	for {
		select {
		case <-c.quit:
			return
		case req = <-c.pendingReqs:
		}

		switch c.filter.Do(req) {
		case Continue:
		case Stop:
			continue
		}

		err = c.enc.Encode(req.Body())
		if err != nil {
			goto FAIL
		}

		if len(c.pendingReqs) == 0 {
			if err = c.enc.Flush(); err != nil {
				goto FAIL
			}
		}
		c.processingReqs <- req
	}

FAIL:
	// req and error must not be nil
	req.SetResponse(newError(err.Error()))
	c.logger.Warnf("loop write exit: %v", err)
}

func (c *client) loopRead() {
	for {
		resp, err := c.dec.Decode()
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				c.logger.Warnf("loop read exit: %v", err)
			}
			return
		}

		req := <-c.processingReqs
		c.handleResp(req, resp)
	}
}

func (c *client) handleResp(req *simpleRequest, v *RespValue) {
	if v.Type != Error {
		req.SetResponse(v)
		return
	}

	i := bytes.Index(v.Text, []byte(" "))
	var errPrefix []byte
	if i != -1 {
		errPrefix = v.Text[:i]
	}
	switch {
	case bytes.EqualFold(errPrefix, []byte(MOVED)),
		bytes.EqualFold(errPrefix, []byte(ASK)):
		if c.onRedirection != nil {
			c.onRedirection(req, v)
			return
		}
	case bytes.EqualFold(errPrefix, []byte(CLUSTERDOWN)):
		if c.onClusterDown != nil {
			c.onClusterDown(req, v)
			return
		}
	}

	// set error as response
	req.SetResponse(v)
}

func (c *client) drainRequests() {
	for {
		select {
		case req := <-c.pendingReqs:
			req.SetResponse(newError(backendExited))
		case req := <-c.processingReqs:
			req.SetResponse(newError(backendExited))
		default:
			return
		}
	}
}

func (c *client) Stop() {
	c.quitOnce.Do(func() {
		close(c.quit)
	})
	c.conn.Close()
	<-c.done
	c.filter.Reset()
}
