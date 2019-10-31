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
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	netutil "github.com/samaritan-proxy/samaritan/proc/internal/net"
	"github.com/samaritan-proxy/samaritan/stats"
)

func newListener(t *testing.T, cfg *service.Listener, connHandleFn ConnHandlerFunc) *listener {
	if cfg == nil {
		cfg = &service.Listener{
			Address: &common.Address{
				Ip:   "127.0.0.1",
				Port: 0,
			},
		}
	}
	scopeName := fmt.Sprintf("service.test")
	s := NewDownstreamStats(stats.CreateScope(scopeName))
	l, err := NewListener(cfg, s, log.New("test"), connHandleFn)
	if err != nil {
		t.Fatal(err)
	}
	return l.(*listener)
}

func TestHandleConn(t *testing.T) {
	connHandler := func(conn net.Conn) {
		conn.Read(make([]byte, 64))
		wconn, ok := conn.(*netutil.Conn)
		if !ok {
			t.Error("expected wrapper conn")
			return
		}
		assert.NotNil(t, wconn.Stats)
		assert.NotZero(t, wconn.Stats.WriteTotal)
		assert.NotZero(t, wconn.Stats.ReadTotal)
		assert.NotZero(t, wconn.Stats.Duration)
	}

	l := newListener(t, nil, connHandler)
	done := make(chan struct{})
	go func() {
		l.Serve()
		close(done)
	}()
	defer func() {
		l.Stop()
		<-done
	}()
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", l.Address())
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	time.Sleep(time.Millisecond * 30)
	assert.Equal(t, l.stats.CxTotal.Value(), uint64(1))
	assert.Equal(t, l.stats.CxActive.Value(), uint64(1))

	conn.Write([]byte("hello, world"))
	conn.Read(make([]byte, 64)) // read until the conn is closed.

	time.Sleep(time.Millisecond * 30)
	assert.Equal(t, l.stats.CxDestroyTotal.Value(), uint64(1))
	assert.Equal(t, l.stats.CxActive.Value(), uint64(0))
}

func TestDrain(t *testing.T) {
	connHandler := func(conn net.Conn) {
		conn.Read(make([]byte, 64))
	}

	l := newListener(t, nil, connHandler)
	done := make(chan struct{})
	go func() {
		l.Serve()
		close(done)
	}()
	defer func() {
		l.Stop()
		<-done
	}()
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", l.Address())
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	time.Sleep(time.Millisecond * 30)

	l.Drain()
	_, err = net.Dial("tcp", l.Address())
	assert.Error(t, err)             // listener has been closed.
	assert.Equal(t, len(l.conns), 1) // check if existing connections are active
}

func TestRetryBinding(t *testing.T) {
	oldLn, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}

	_, portStr, _ := net.SplitHostPort(oldLn.Addr().String())
	port, _ := strconv.Atoi(portStr)
	cfg := &service.Listener{
		Address: &common.Address{
			Ip:   "127.0.0.1",
			Port: uint32(port),
		},
	}
	l := newListener(t, cfg, nil)
	done := make(chan struct{})
	go func() {
		l.Serve()
		close(done)
	}()
	defer func() {
		l.Stop()
		<-done
	}()
	time.Sleep(time.Millisecond * 100)

	time.Sleep(time.Millisecond * 100)
	assert.Empty(t, l.Address())

	// close the old listener
	oldLn.Close()
	time.Sleep(time.Second)
	assert.NotEmpty(t, l.Address())
}

func TestConnsLimit(t *testing.T) {
	connHandleFn := func(conn net.Conn) {
		if _, err := conn.Read(make([]byte, 5)); err != nil {
			panic(err)
		}
		conn.Write([]byte("hello"))
		// block here
		conn.Read(make([]byte, 1))
	}

	l := newListener(t, nil, connHandleFn)
	l.cfg.ConnectionLimit = 100
	done := make(chan struct{})
	go func() {
		l.Serve()
		close(done)
	}()
	defer func() {
		l.Stop()
		<-done
	}()
	time.Sleep(time.Millisecond * 100)

	conns := make([]net.Conn, 0)
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()
	for {
		conn, err := net.Dial("tcp", l.Address())
		if err != nil {
			break
		}
		if n, err := conn.Write([]byte("hello")); err != nil || n != 5 {
			break
		}
		if n, err := conn.Read(make([]byte, 5)); err != nil || n != 5 {
			break
		}
		conns = append(conns, conn)
	}
	assert.EqualValues(t, l.cfg.ConnectionLimit, len(conns))
}
