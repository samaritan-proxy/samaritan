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

package tcp

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc"
	"github.com/samaritan-proxy/samaritan/proc/internal/lb"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	netutil "github.com/samaritan-proxy/samaritan/proc/internal/net"
)

func init() {
	proc.RegisterBuilder(protocol.TCP, new(builder))
}

// builder is an implementation of proc.Builder.
type builder struct{}

func (*builder) Build(params proc.BuildParams) (proc.Proc, error) {
	return newProc(params.Name, params.Cfg, params.Hosts, params.Stats, params.Logger)
}

type tcpProc struct {
	log.Logger
	stats   *proc.Stats
	name    string
	cfg     *service.Config
	hostSet *host.Set

	ln proc.Listener
	lb lb.Balancer
	hm *hc.Monitor

	wg sync.WaitGroup
}

func newProc(name string, cfg *service.Config, hosts []*host.Host, stats *proc.Stats, logger log.Logger) (*tcpProc, error) {
	p := &tcpProc{
		Logger:  logger,
		stats:   stats,
		name:    name,
		cfg:     cfg,
		hostSet: host.NewSet(hosts...),
		lb:      lb.New(cfg.GetLbPolicy()),
	}

	var err error
	p.ln, err = proc.NewListener(cfg.Listener, stats.Downstream, logger, p.HandleConn)
	if err != nil {
		return nil, err
	}

	p.hm, err = hc.NewMonitor(cfg.HealthCheck, p.hostSet, logger)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *tcpProc) Start() error {
	p.hm.Start()
	p.wg.Add(1)
	go func() {
		p.ln.Serve()
		p.wg.Done()
	}()
	return nil
}

func (p *tcpProc) Name() string {
	return p.name
}

func (p *tcpProc) HandleConn(conn net.Conn) {
	// attach idle timeout to conn
	cconn := netutil.New(conn)
	cconn.SetReadTimeout(*p.cfg.IdleTimeout)
	healthyHosts := p.hostSet.Healthy()
	if len(healthyHosts) == 0 {
		p.Warnf("No available host")
		return
	}

	host := p.lb.PickHost(healthyHosts)
	sconn, err := p.dial(host)
	if err != nil {
		p.Warnf("Dial to host[%s] failed: %v", host, err)
		p.stats.Upstream.CxConnectFail.Inc()
		return
	}
	defer sconn.Close()

	// update stats
	host.IncConnCount()
	p.stats.Upstream.CxTotal.Inc()
	p.stats.Upstream.CxActive.Inc()
	defer func() {
		host.DecConnCount()
		p.stats.Upstream.CxDestroyTotal.Inc()
		p.stats.Upstream.CxActive.Dec()
	}()

	done := make(chan struct{})

	// close conn when host removed form host set
	go func() {
		select {
		case <-host.WaitRemoved():
			p.Infof("host: %s removed, conn will close...", host.Addr)
			sconn.Close()
			cconn.Close()
			return
		case <-done:
			return
		}
	}()
	go func() {
		p.pipeConn(cconn, sconn)
		close(done)
	}()
	p.pipeConn(sconn, cconn)
	<-done
}

type closeReader interface {
	CloseRead() error
}

type closeWriter interface {
	CloseWrite() error
}

func closeRead(conn net.Conn) error {
	if closer, ok := conn.(closeReader); ok {
		return closer.CloseRead()
	}
	return nil
}

func closeWrite(conn net.Conn) error {
	if closer, ok := conn.(closeWriter); ok {
		return closer.CloseWrite()
	}
	return nil
}

var (
	bufSize = 16 * 1024

	// *[]byte pool
	bufPool = &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, bufSize)
			return &buf
		},
	}
)

func getBuffer() []byte {
	return *(bufPool.Get().(*[]byte))
}

func putBuffer(buf []byte) {
	bufPool.Put(&buf)
}

func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if len(buf) != 0 {
		return io.CopyBuffer(dst, src, buf)
	}

	buf = getBuffer()
	written, err = io.CopyBuffer(dst, src, buf)
	putBuffer(buf)
	return
}

func (p *tcpProc) pipeConn(src, dst net.Conn) {
	_, err := copyBuffer(dst, src, nil)
	if err == nil {
		err = errors.New("read EOF")
	}
	p.Debugf("%s -> %s -> %s closed: %s", src.RemoteAddr(), p.Address(), dst.RemoteAddr(), err)
	if err := closeWrite(dst); err != nil {
		dst.Close()
	}
	if err := closeRead(src); err != nil {
		src.Close()
	}
}

var (
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout(network, address, timeout)
	}
)

func (p *tcpProc) dial(host *host.Host) (net.Conn, error) {
	rawConn, err := dialTimeout("tcp", host.Addr, *p.cfg.ConnectTimeout)
	if err != nil {
		if _, ok := err.(interface {
			Timeout() bool
		}); ok {
			p.stats.Upstream.CxConnectTimeout.Inc()
		}
		return nil, err
	}

	conn := netutil.New(rawConn)
	conn.SetReadTimeout(*p.cfg.IdleTimeout)
	stats := &netutil.Stats{
		ReadTotal:  p.stats.Upstream.CxRxBytesTotal,
		WriteTotal: p.stats.Upstream.CxTxBytesTotal,
		Duration:   p.stats.Upstream.CxLengthSec,
	}
	conn.SetStats(stats)
	conn.SetInBytesCounter(host.ConnBytesInCounter())
	conn.SetOutBytesCounter(host.ConnBytesOutCounter())
	return conn, nil
}

func (p *tcpProc) Address() string {
	return p.ln.Address()
}

func (p *tcpProc) Config() *service.Config {
	return p.cfg
}

func (p *tcpProc) OnSvcHostAdd(hosts []*host.Host) error {
	p.hostSet.Add(hosts...)
	return nil
}

func (p *tcpProc) OnSvcHostRemove(hosts []*host.Host) error {
	p.hostSet.Remove(hosts...)
	return nil
}

func (p *tcpProc) OnSvcAllHostReplace(hosts []*host.Host) error {
	p.hostSet.ReplaceAll(hosts)
	return nil
}

func (p *tcpProc) OnSvcConfigUpdate(c *service.Config) error {
	// update if strategy changes.
	if newHC := c.GetHealthCheck(); !p.cfg.GetHealthCheck().Equal(newHC) {
		var err error
		if p.hm == nil {
			p.hm, err = hc.NewMonitor(newHC, p.hostSet, p.Logger)
			if err == nil {
				p.hm.Start()
			}
		} else {
			err = p.hm.ResetHealthCheck(newHC)
		}
		if err != nil {
			return err
		}
	}

	// update balance policy
	if newPolicy := c.GetLbPolicy(); p.cfg.GetLbPolicy() != newPolicy {
		p.lb = lb.New(newPolicy)
	}
	if !p.cfg.Equal(c) {
		p.cfg = c
	}
	// TODO: update listener
	return nil
}

func (p *tcpProc) StopListen() (err error) {
	return p.ln.Drain()
}

func (p *tcpProc) Stop() error {
	p.hm.Stop()
	p.ln.Stop()
	p.wg.Wait()
	return nil
}
