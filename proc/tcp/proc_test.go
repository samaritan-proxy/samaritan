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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/lb"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	"github.com/samaritan-proxy/samaritan/stats"
	"github.com/samaritan-proxy/samaritan/utils"
)

func TestCloseRead(t *testing.T) {
	tcpConn := new(net.TCPConn)
	err := closeRead(tcpConn)
	assert.Equal(t, err, syscall.EINVAL)
	udpConn := new(net.UDPConn)
	err = closeRead(udpConn)
	assert.Nil(t, err)
}

func TestCloseWrite(t *testing.T) {
	tcpConn := new(net.TCPConn)
	err := closeWrite(tcpConn)
	assert.Equal(t, err, syscall.EINVAL)
	udpConn := new(net.UDPConn)
	err = closeWrite(udpConn)
	assert.Nil(t, err)
}

func newServiceConfig() *service.Config {
	return &service.Config{
		HealthCheck: &hc.HealthCheck{
			Interval:      time.Second,
			Timeout:       time.Second,
			FallThreshold: 3,
			RiseThreshold: 3,
			Checker: &hc.HealthCheck_TcpChecker{
				TcpChecker: &hc.TCPChecker{},
			},
		},
		Listener: &service.Listener{
			Address: &common.Address{
				Ip:   "127.0.0.1",
				Port: 0,
			},
			ConnectionLimit: 0,
		},
		ConnectTimeout: utils.DurationPtr(time.Second),
		IdleTimeout:    utils.DurationPtr(time.Second),
		LbPolicy:       service.LoadBalancePolicy_ROUND_ROBIN,
		Protocol:       protocol.TCP,
		ProtocolOptions: &service.Config_TcpOption{
			TcpOption: &protocol.TCPOption{},
		},
	}
}

func TestNewProc(t *testing.T) {
	cfg := newServiceConfig()
	stats := proc.NewStats(stats.CreateScope("blabla"))
	p, err := newProc("blabla", cfg, nil, stats, nil)
	assert.Nil(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, p.Name(), "blabla")
}

func TestBuilder(t *testing.T) {
	b := new(builder)
	cfg := newServiceConfig()
	stats := proc.NewStats(stats.CreateScope("blabla"))
	logger := log.New("blabla")
	p, err := b.Build(proc.BuildParams{
		Name: "blabla", Cfg: cfg, Stats: stats, Logger: logger,
	})
	assert.Nil(t, err)
	assert.NotNil(t, p)
}

func testProc(t *testing.T, cfg *service.Config, hosts []*host.Host, fn func(p *tcpProc)) {
	if cfg == nil {
		cfg = newServiceConfig()
	}
	scopeName := fmt.Sprintf("service.%s.", strings.Replace("test", ".", "_", -1))
	s := proc.NewStats(stats.CreateScope(scopeName))

	p, err := newProc("test", cfg, hosts, s, log.New("[test]"))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	// proc start need time, should wait a little while
	time.Sleep(time.Millisecond * 100)
	fn(p)
}

func TestNoAvailableSvcNode(t *testing.T) {
	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)
		_, err = conn.Read(make([]byte, 1))
		assert.NotNil(t, err)
	}
	testProc(t, nil, nil, fn)
}

func TestConnectSvcNodeTimeout(t *testing.T) {
	fn := func(p *tcpProc) {
		saved := dialTimeout
		dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, errors.New("i/o timeout")
		}
		defer func() { dialTimeout = saved }()

		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)
		_, err = conn.Read(make([]byte, 1))
		assert.NotNil(t, err)
	}

	ln, err := newLocalListener()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	cfg := newServiceConfig()
	hostAddr := ln.Addr().String()
	hosts := []*host.Host{
		host.New(hostAddr),
	}
	testProc(t, cfg, hosts, fn)
}

func TestSvcIdleTimeout(t *testing.T) {
	ls, err := newLocalServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.tearDown()
	ls.buildup(func(ls *localServer) {
		conn, err := ls.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		for {
			_, err = conn.Read(make([]byte, 64))
			if err != nil {
				break
			}
		}
		assert.Equal(t, io.EOF, err)
	})

	cfg := newServiceConfig()
	h := host.New(ls.Address())
	hosts := []*host.Host{h}
	cfg.IdleTimeout = utils.DurationPtr(time.Second)

	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)
		defer conn.Close()

		readDone := make(chan bool)
		writeDone := make(chan bool)
		go func() {
			for {
				select {
				case <-readDone:
					close(writeDone)
					return
				default:
					conn.Write(make([]byte, 64))
				}
			}
		}()
		_, err = conn.Read(make([]byte, 64))
		assert.Equal(t, io.EOF, err)
		close(readDone)
		<-writeDone
	}
	testProc(t, cfg, hosts, fn)
}

func TestClientIdleTimeout(t *testing.T) {
	ls, err := newLocalServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.tearDown()
	ls.buildup(func(ls *localServer) {
		conn, err := ls.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		readDone := make(chan bool)
		writeDone := make(chan bool)
		go func() {
			for {
				select {
				case <-readDone:
					close(writeDone)
					return
				default:
					conn.Write(make([]byte, 64))
				}
			}
		}()
		_, err = conn.Read(make([]byte, 64))
		assert.Equal(t, io.EOF, err)
		close(readDone)
		<-writeDone
	})

	cfg := newServiceConfig()
	h := host.New(ls.Address())
	hosts := []*host.Host{h}
	cfg.IdleTimeout = utils.DurationPtr(time.Second)

	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)
		defer conn.Close()
		for {
			if _, err = conn.Read(make([]byte, 64)); err != nil {
				break
			}
		}
		assert.Equal(t, io.EOF, err)
	}
	testProc(t, cfg, hosts, fn)
}

func TestDualReadWrite(t *testing.T) {
	ls, err := newLocalServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.tearDown()
	ls.buildup(func(ls *localServer) {
		conn, err := ls.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		checkWrite(t, conn, []byte("hello, world"))
		checkRead(t, conn, []byte("line 2"), nil)
		checkRead(t, conn, nil, io.EOF)
	})

	cfg := newServiceConfig()
	h := host.New(ls.Address())
	hosts := []*host.Host{h}

	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		if err != nil {
			t.Error(err)
			return
		}

		checkRead(t, conn, []byte("hello, world"), nil)
		checkWrite(t, conn, []byte("line 2"))
		time.Sleep(time.Millisecond * 50)

		conn.Close()
		time.Sleep(time.Millisecond * 50)
	}
	testProc(t, cfg, hosts, fn)
}

func checkWrite(t *testing.T, w io.Writer, data []byte) {
	n, err := w.Write(data)
	if err != nil {
		t.Error(err)
		return
	}
	if n != len(data) {
		t.Errorf("short write: %d != %d", n, len(data))
	}
}

func checkRead(t *testing.T, r io.Reader, data []byte, wantErr error) {
	buf := make([]byte, len(data)+10)
	n, err := r.Read(buf)
	if err != wantErr {
		t.Error(err)
		return
	}
	if n != len(data) || !bytes.Equal(buf[0:n], data) {
		t.Errorf("bad read: got %q", buf[0:n])
		return
	}
}

func deepCopy(cfg *service.Config, t *testing.T) *service.Config {
	b, err := cfg.Marshal()
	assert.NoError(t, err)
	newCfg := new(service.Config)
	assert.NoError(t, newCfg.Unmarshal(b))
	assert.True(t, newCfg.Equal(cfg))
	return newCfg
}

func TestResetConnectTimeout(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		newCfg := deepCopy(p.Config(), t)
		newCfg.ConnectTimeout = utils.DurationPtr(time.Minute * 31)
		assert.NoError(t, p.OnSvcConfigUpdate(newCfg))
		assert.Equal(t, *p.cfg.ConnectTimeout, time.Minute*31)
	})
}

func TestResetIdleTimeout(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		newCfg := deepCopy(p.Config(), t)
		newCfg.IdleTimeout = utils.DurationPtr(time.Minute * 31)
		assert.NoError(t, p.OnSvcConfigUpdate(newCfg))
		assert.Equal(t, *p.cfg.IdleTimeout, time.Minute*31)
	})
}

func TestResetLBPolicy(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		newCfg := deepCopy(p.Config(), t)
		newCfg.LbPolicy = service.LoadBalancePolicy_RANDOM
		assert.NoError(t, p.OnSvcConfigUpdate(newCfg))
		assert.Equal(t, p.cfg.LbPolicy, service.LoadBalancePolicy_RANDOM)
		assert.Equal(t, p.lb.Name(), lb.New(service.LoadBalancePolicy_RANDOM).Name())
	})
}

func TestResetHealthCheckPolicy(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		newCfg := deepCopy(p.Config(), t)
		newCfg.HealthCheck.FallThreshold = 5
		assert.NoError(t, p.OnSvcConfigUpdate(newCfg))
		assert.EqualValues(t, p.cfg.HealthCheck.FallThreshold, 5)
		// TODO(kirk91): check monitor called.
	})
}

func TestResetProtocolConfig(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		// TODO(kirk91): check monitor called.
	})
}

func TestAddHosts(t *testing.T) {
	testProc(t, nil, nil, func(p *tcpProc) {
		h := host.New("1.2.3.4:0")
		assert.NoError(t, p.OnSvcHostAdd([]*host.Host{h}))
		assert.Equal(t, p.hostSet.All(), []*host.Host{h})
	})
}

func TestRemoveHosts(t *testing.T) {
	h := host.New("1.2.3.4:0")
	hosts := []*host.Host{h}
	testProc(t, nil, hosts, func(p *tcpProc) {
		assert.Equal(t, len(p.hostSet.All()), 1)
		assert.NoError(t, p.OnSvcHostRemove([]*host.Host{h}))
		assert.Equal(t, len(p.hostSet.All()), 0)
	})
}

func TestReplaceAllHosts(t *testing.T) {
	h1 := host.New("1.2.3.4:1")
	h2 := host.New("1.2.3.4:2")
	testProc(t, nil, []*host.Host{h1}, func(p *tcpProc) {
		assert.Equal(t, len(p.hostSet.All()), 1)
		assert.NoError(t, p.OnSvcAllHostReplace([]*host.Host{h2}))
		assert.Equal(t, len(p.hostSet.All()), 1)
		assert.Equal(t, h2.Addr, p.hostSet.All()[0].Addr)
	})
}

func TestRemoveHost(t *testing.T) {
	ls, err := newLocalServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.tearDown()
	ls.buildup(func(ls *localServer) {
		conn, err := ls.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		checkWrite(t, conn, []byte("hello, world"))
		conn.Read(make([]byte, 8))
	})

	cfg := newServiceConfig()
	h := host.New(ls.Address())
	hosts := []*host.Host{h}
	cfg.IdleTimeout = utils.DurationPtr(time.Second)

	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)

		time.Sleep(time.Millisecond * 10) // wait the first connection is established.

		checkRead(t, conn, []byte("hello, world"), nil)

		p.hostSet.Remove(h)

		st := time.Now()
		_, err = conn.Read(make([]byte, 8))
		dur := time.Since(st)

		assert.Equal(t, io.EOF, err)
		// dur ≈ 4.99 if conn not close
		assert.True(t, dur < time.Second, dur.String())
		p.Stop()
	}
	testProc(t, cfg, hosts, fn)
}

func TestStopListen(t *testing.T) {
	ls, err := newLocalServer()
	if err != nil {
		t.Fatal(err)
	}
	defer ls.tearDown()
	ls.buildup(func(ls *localServer) {
		conn, err := ls.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		conn.Read(make([]byte, 64))
	})

	cfg := newServiceConfig()
	h := host.New(ls.Address())
	hosts := []*host.Host{h}

	fn := func(p *tcpProc) {
		conn, err := net.Dial("tcp", p.Address())
		assert.Nil(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn.Read(make([]byte, 64))
		}()

		time.Sleep(time.Millisecond * 10) // wait the first connection is established.
		p.StopListen()
		time.Sleep(time.Millisecond * 10) // wait to close the listener.

		_, err = net.Dial("tcp", p.Address())
		if !strings.Contains(err.Error(), "connection refused") {
			t.Error(err)
		}

		p.Stop()
		wg.Wait()
	}
	testProc(t, cfg, hosts, fn)
}

func TestCopyBuffer(t *testing.T) {
	wb := new(bytes.Buffer)
	rb := new(bytes.Buffer)
	rb.WriteString("hello, world.")
	copyBuffer(wb, rb, make([]byte, 1))
	assert.Equal(t, wb.String(), "hello, world.")
}

func TestCopyBufferNil(t *testing.T) {
	wb := new(bytes.Buffer)
	rb := new(bytes.Buffer)
	rb.WriteString("hello, world.")
	copyBuffer(wb, rb, nil)
	assert.Equal(t, wb.String(), "hello, world.")
}
