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
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	"github.com/samaritan-proxy/samaritan/stats"
)

func init() {
	proc.RegisterBuilder(protocol.Redis, new(builder))
}

type builder struct{}

func (*builder) Build(params proc.BuildParams) (proc.Proc, error) {
	return newRedisProc(params.Name, params.Cfg, params.Hosts,
		params.Stats, params.Logger)
}

type redisProc struct {
	name   string
	stats  *proc.Stats
	logger log.Logger

	l        proc.Listener
	u        *upstream
	cmdHdlrs map[string]*commandHandler
	wg       sync.WaitGroup

	cfg      *config
	cfgHooks []func(*config)
}

func newRedisProc(svcName string, svcCfg *service.Config, svcHosts []*host.Host,
	stats *proc.Stats, logger log.Logger) (*redisProc, error) {
	p := &redisProc{
		name:     svcName,
		cfg:      svcCfg,
		stats:    stats,
		logger:   logger,
		cmdHdlrs: make(map[string]*commandHandler),
	}

	l, err := proc.NewListener(p.cfg.Listener, p.stats.Downstream, logger, p.handleConn)
	if err != nil {
		return nil, err
	}

	p.l = l
	p.u = newUpstream(p.cfg, svcHosts, p.logger, p.stats.Upstream)
	p.initCommandHandlers()
	return p, nil
}

func (p *redisProc) registerConfigHook(hook func(*config)) {
	p.cfgHooks = append(p.cfgHooks, hook)
}

func (p *redisProc) initCommandHandlers() {
	scope := p.stats.NewChild("redis")
	for _, cmd := range simpleCommands {
		p.addHandler(scope, cmd, handleSimpleCommand)
	}

	for _, cmd := range sumResultCommands {
		p.addHandler(scope, cmd, handleSumResultCommand)
	}

	p.addHandler(scope, "eval", handleEval)
	p.addHandler(scope, "mset", handleMSet)
	p.addHandler(scope, "mget", handleMGet)

	p.addHandler(scope, "ping", handlePing)
	p.addHandler(scope, "quit", handleQuit)
	p.addHandler(scope, "info", handleInfo)
	p.addHandler(scope, "time", handleTime)
	p.addHandler(scope, "select", handleSelect)
}

func (p *redisProc) addHandler(scope *stats.Scope, cmd string, fn commandHandleFunc) {
	hdlr := &commandHandler{
		stats:  newCommandStats(scope, cmd),
		handle: fn,
	}
	p.cmdHdlrs[cmd] = hdlr
}

func (p *redisProc) findHandler(cmd string) (*commandHandler, bool) {
	hdlr, ok := p.cmdHdlrs[strings.ToLower(cmd)]
	return hdlr, ok
}

func (p *redisProc) Start() error {
	// TODO(kirk91): use errgroup to cooridnate different goroutines.
	p.wg.Add(1)
	go func() {
		p.u.Serve()
		p.wg.Done()
	}()
	p.wg.Add(1)
	go func() {
		p.l.Serve()
		p.wg.Done()
	}()
	return nil
}

func (p *redisProc) Name() string {
	return p.name
}

func (p *redisProc) OnSvcHostAdd(hosts []*host.Host) error {
	return p.u.OnHostAdd(hosts...)
}

func (p *redisProc) OnSvcHostRemove(hosts []*host.Host) error {
	return p.u.OnHostRemove(hosts...)
}

func (p *redisProc) OnSvcAllHostReplace(hosts []*host.Host) error {
	return p.u.OnHostReplace(hosts)
}

func (p *redisProc) OnSvcConfigUpdate(newCfg *service.Config) error {
	if err := newCfg.Validate(); err != nil {
		return err
	}
	p.cfg = newCfg
	for _, hook := range p.cfgHooks {
		hook(p.cfg)
	}
	return nil
}

func (p *redisProc) Config() *service.Config {
	return p.cfg
}

func (p *redisProc) Address() string {
	return p.l.Address()
}

func (p *redisProc) StopListen() error {
	return p.l.Drain()
}

func (p *redisProc) Stop() error {
	p.l.Stop()
	p.u.Stop()
	p.wg.Wait()
	return nil
}

func (p *redisProc) handleConn(conn net.Conn) {
	s := newSession(p, conn)
	s.Serve()
}

func (p *redisProc) handleRequest(req *rawRequest) {
	// rawRequest metrics
	p.stats.Downstream.RqTotal.Inc()
	req.RegisterHook(func(req *rawRequest) {
		switch req.Response().Type {
		case Error:
			p.stats.Downstream.RqFailureTotal.Inc()
		default:
			p.stats.Downstream.RqSuccessTotal.Inc()
		}
		p.stats.Downstream.RqDurationMs.Record(uint64(req.Duration() / time.Millisecond))
	})

	// check
	if !req.IsValid() {
		req.SetResponse(newError(invalidRequest))
		return
	}

	// find corresponding _handler
	cmd := string(req.Body().Array[0].Text)
	hdlr, ok := p.findHandler(cmd)
	if !ok {
		// unsupported command
		req.SetResponse(newError(fmt.Sprintf("unsupported command %s", cmd)))
		return
	}

	cmdStats := hdlr.stats
	cmdStats.Total.Inc()
	req.RegisterHook(func(req *rawRequest) {
		switch req.Response().Type {
		case Error:
			cmdStats.Error.Inc()
		default:
			cmdStats.Success.Inc()
		}
		latency := uint64(req.Duration() / time.Microsecond)
		cmdStats.LatencyMicros.Record(latency)
	})
	hdlr.handle(p.u, req)
}
