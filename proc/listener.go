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

package proc

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	reuseport "github.com/kavu/go_reuseport"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	netutil "github.com/samaritan-proxy/samaritan/proc/internal/net"
)

type ConnHandlerFunc func(conn net.Conn)

var (
	defaultListenFunc = reuseport.NewReusablePortListener

	ErrIPNotSet = errors.New("ip not set")
)

type Listener interface {
	Address() string
	Serve() error
	Drain() error
	Stop() error
}

type listener struct {
	log.Logger
	cfg   *service.Listener
	stats *DownstreamStats
	mu    sync.Mutex

	ln           net.Listener
	conns        map[net.Conn]struct{}
	connsWg      sync.WaitGroup
	connHandleFn ConnHandlerFunc

	drainOnce sync.Once
	drain     chan struct{}
	quitOnce  sync.Once
	quit      chan struct{}
	done      chan struct{}
}

func NewListener(cfg *service.Listener, stats *DownstreamStats, logger log.Logger, connHandleFn ConnHandlerFunc) (Listener, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Address.GetIp() == "" {
		return nil, ErrIPNotSet
	}

	l := &listener{
		cfg:          cfg,
		Logger:       logger,
		stats:        stats,
		conns:        make(map[net.Conn]struct{}),
		connHandleFn: connHandleFn,
		drain:        make(chan struct{}),
		quit:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	return l, nil
}

func (l *listener) Serve() error {
	ip := l.cfg.GetAddress().GetIp()
	port := l.cfg.GetAddress().GetPort()
	address := fmt.Sprintf("%s:%d", ip, port)

	var ln net.Listener
	for {
		select {
		case <-l.quit:
			return nil
		case <-l.drain:
			return nil
		default:
		}

		var err error
		ln, err = defaultListenFunc("tcp", address)
		if err == nil {
			break
		}

		l.Warnf("listen on %s failed: %v, will keep trying...", address, err)
		// TODO: use backoff algorithm to calculate sleep time.
		t := time.NewTimer(time.Millisecond * 500)
		select {
		case <-t.C:
		case <-l.drain:
			return nil
		case <-l.quit:
			return nil
		}
	}

	l.ln = ln
	l.Infof("start serving at %s", ln.Addr().String())
	l.serve()
	l.Infof("stop serving at %s, waiting all conns done", ln.Addr().String())

	l.connsWg.Wait()
	l.Infof("all conns done")
	close(l.done)
	return nil
}

func (l *listener) serve() {
	var tempDelay time.Duration
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				l.Warnf("accept failed: %v; retrying in %s", err, tempDelay)
				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-l.quit:
					timer.Stop()
					return
				}
				continue
			}

			select {
			case <-l.drain:
				return
			case <-l.quit:
				return
			default:
			}
			l.Warnf("done serving; accept failed: %v", err)
			return
		}

		l.connsWg.Add(1)
		go func(conn net.Conn) {
			l.handleRawConn(conn)
			l.connsWg.Done()
		}(conn)
	}
}

func (l *listener) handleRawConn(rawConn net.Conn) {
	conn := l.wrapRawConn(rawConn)
	connCreatedAt := time.Now()

	if !l.addConn(conn) {
		conn.Close()
		return
	}

	l.Debugf("%s -> %s created", conn.RemoteAddr(), l.ln.Addr().String())
	defer func() {
		conn.Close()
		l.removeConn(conn)
		l.Debugf("%s -> %s finished, duration: %s", conn.RemoteAddr(), l.ln.Addr().String(), time.Since(connCreatedAt).String())
	}()

	if l.connHandleFn == nil {
		l.Warnf("conn handle fn is nil, will close conn immediately")
		return
	}
	l.connHandleFn(conn)
}

func (l *listener) wrapRawConn(rawConn net.Conn) net.Conn {
	c := netutil.New(rawConn)
	// TODO(kirk91): use hook to replace it.
	stats := &netutil.Stats{
		WriteTotal: l.stats.CxTxBytesTotal,
		ReadTotal:  l.stats.CxRxBytesTotal,
		Duration:   l.stats.CxLengthSec,
	}
	c.SetStats(stats)
	return c
}

func (l *listener) addConn(conn net.Conn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conns == nil {
		return false
	}
	if l.connsLimit() {
		l.stats.CxRestricted.Inc()
		l.Warnf("connections limit, %s -> %s, will close", conn.RemoteAddr().String(), l.ln.Addr().String())
		return false
	}
	l.conns[conn] = struct{}{}
	l.stats.CxTotal.Inc()
	l.stats.CxActive.Inc()
	return true
}

func (l *listener) removeConn(conn net.Conn) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conns == nil {
		return
	}
	if _, ok := l.conns[conn]; !ok {
		return
	}
	delete(l.conns, conn)
	l.stats.CxDestroyTotal.Inc()
	l.stats.CxActive.Dec()
}

func (l *listener) connsLimit() bool {
	limit := l.cfg.ConnectionLimit
	if limit == 0 || uint32(len(l.conns)) < limit {
		return false
	}
	return true
}

func (l *listener) Address() string {
	if l.ln == nil {
		return ""
	}
	return l.ln.Addr().String()
}

func (l *listener) Drain() error {
	l.drainOnce.Do(func() {
		close(l.drain)
	})
	if l.ln != nil {
		l.ln.Close()
	}
	return nil
}

func (l *listener) Stop() error {
	l.quitOnce.Do(func() {
		close(l.quit)
	})

	l.mu.Lock()
	conns := l.conns
	l.conns = nil
	l.mu.Unlock()

	if l.ln != nil {
		l.ln.Close()
	}
	for conn := range conns {
		conn.Close()
	}
	<-l.done
	return nil
}
