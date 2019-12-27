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
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	"github.com/samaritan-proxy/samaritan/proc/internal/syscall"
	"github.com/samaritan-proxy/samaritan/stats"
)

func newTestClient(t *testing.T, conn net.Conn, cfg *config, options ...clientOption) *client {
	t.Helper()

	if cfg == nil {
		// TODO: use the default config
	}

	logger := log.New("[test]")
	c, err := newClient(conn, cfg, logger, options...)
	if err != nil {
		t.Fatal(err)
	}
	// drain the initial auth and readonly requests.
	c.drainRequests()
	return c
}

func TestUpstreamClientTCPUserTimeout(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	conn, err := net.Dial("tcp", lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	c := newTestClient(t, conn, nil)

	// assert tcp user timeout
	opt, err := syscall.GetTCPUserTimeout(c.conn)
	if err != nil {
		t.Fatal(err)
	}
	if opt < 0 {
		t.Skipf("skipping test on unsupported platform")
	}
	assert.NotZero(t, opt)
}

func TestUpstreamClientReadError(t *testing.T) {
	cconn, sconn := net.Pipe()
	c := newTestClient(t, cconn, nil)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	time.AfterFunc(time.Millisecond*100, func() {
		sconn.Close()
	})
	<-done
}

func TestUpstreamClientWriteError(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	tcpConn := conn.(*net.TCPConn)
	c := newTestClient(t, tcpConn, nil)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	time.AfterFunc(time.Millisecond*100, func() {
		tcpConn.CloseWrite()
		c.Send(newSimpleRequest(newStringArray("ping")))
	})
	<-done
}

func TestUpstreamClientStop(t *testing.T) {
	cconn, _ := net.Pipe()
	c := newTestClient(t, cconn, nil)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	time.AfterFunc(time.Millisecond*100, func() {
		c.Stop()
	})
	<-done
}

func TestUpstreamClientDrainRequests(t *testing.T) {
	cconn, _ := net.Pipe()
	c := newTestClient(t, cconn, nil)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	time.AfterFunc(time.Millisecond*100, c.Stop)

	var reqs []*simpleRequest
loop:
	for {
		select {
		case <-done:
			break loop
		default:
		}
		req := newSimpleRequest(newStringArray("ping"))
		c.Send(req)
		reqs = append(reqs, req)
	}

	// assert
	for _, req := range reqs {
		req.Wait()
	}
}

func TestUpstreamClientHandleRedirection(t *testing.T) {
	cconn, sconn := net.Pipe()
	options := []clientOption{
		withRedirectionCb(func(req *simpleRequest, v *RespValue) {
			req.SetResponse(newBulkString("1"))
		}),
	}
	c := newTestClient(t, cconn, nil, options...)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	defer func() {
		c.Stop()
		<-done
	}()

	t.Run("moved", func(t *testing.T) {
		go func() {
			sconn.Read(make([]byte, 8192))
			sconn.Write([]byte("-moved 3999 127.0.0.1:6380\r\n"))
		}()
		req := newSimpleRequest(newArray(
			*newBulkString("get"),
			*newBulkString("a"),
		))
		c.Send(req)
		req.Wait()
		assert.Equal(t, BulkString, req.Response().Type)
	})

	t.Run("ask", func(t *testing.T) {
		go func() {
			sconn.Read(make([]byte, 8192))
			sconn.Write([]byte("-ask 3999 127.0.0.1:6380\r\n"))
		}()
		req := newSimpleRequest(newArray(
			*newBulkString("get"),
			*newBulkString("a"),
		))
		c.Send(req)
		req.Wait()
		assert.Equal(t, BulkString, req.Response().Type)
	})
}

func TestUpstreamClientHandleClusterDown(t *testing.T) {
	cconn, sconn := net.Pipe()
	options := []clientOption{
		withClusterDownCb(func(req *simpleRequest, v *RespValue) {
			req.SetResponse(newBulkString("1"))
		}),
	}
	c := newTestClient(t, cconn, nil, options...)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	defer func() {
		c.Stop()
		<-done
	}()

	go func() {
		sconn.Read(make([]byte, 8192))
		sconn.Write([]byte("-CLUSTERDOWN the cluster is down\r\n"))
	}()
	req := newSimpleRequest(newArray(
		*newBulkString("get"),
		*newBulkString("a"),
	))
	c.Send(req)
	req.Wait()
	assert.Equal(t, BulkString, req.Response().Type)
}

func TestUpstreamClientHandleResp(t *testing.T) {
	c := newTestClient(t, nil, nil)
	c.filter = newRequestFilterChain()

	// normal response
	req := newSimpleRequest(newSimpleString("ping"))
	c.handleResp(req, respOK)
	req.Wait()
	assert.Equal(t, SimpleString, req.Response().Type)

	// error response
	req = newSimpleRequest(newArray(
		*newBulkString("get"),
		*newBulkString("a"),
	))
	c.handleResp(req, newError("internal error"))
	req.Wait()
	assert.Equal(t, Error, req.Response().Type)
}

func newTestUpstream(cfg *config, hosts ...*host.Host) *upstream {
	logger := log.New("[test]")
	stats := proc.NewUpstreamStats(stats.CreateScope("test"))

	if cfg == nil {
		// TODO: use the default config
		cfg = newConfig(&service.Config{
			ConnectTimeout: utils.DurationPtr(time.Second),
		})
	}

	if hosts == nil {
		hosts = []*host.Host{}
	}

	return newUpstream(cfg, hosts, logger, stats)
}

func TestUpstreamCreateClient(t *testing.T) {
	u := newTestUpstream(nil)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	c, err := u.createClient(l.Addr().String())
	assert.NoError(t, err)
	assert.NotNil(t, c)
	defer c.Stop()
	assert.Equal(t, 1, len(u.loadClients()))

	// same addr
	o, err := u.createClient(l.Addr().String())
	assert.NoError(t, err)
	assert.Equal(t, c, o)
}

func TestUpstreamCreateClientParallel(t *testing.T) {
	u := newTestUpstream(nil)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	n := runtime.GOMAXPROCS(0)
	if n < 2 {
		n = 2
	}
	cs := make([]*client, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _ := u.createClient(l.Addr().String())
			cs[i] = c
		}(i)
	}

	wg.Wait()
	for i := 1; i < n; i++ {
		assert.NotNil(t, cs[i])
		assert.Equal(t, cs[0], cs[i])
		cs[i].Stop()
	}
}

func TestUpstreamRemoveClient(t *testing.T) {
	u := newTestUpstream(nil)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	c, _ := u.createClient(l.Addr().String())
	defer c.Stop()
	assert.Equal(t, 1, len(u.loadClients()))
	u.removeClient(l.Addr().String())
	assert.Equal(t, 0, len(u.loadClients()))

	// remove non-existent client
	u.removeClient(l.Addr().String())
	assert.Equal(t, 0, len(u.loadClients()))
}

func TestUpstreamGetClientParallel(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	addr := l.Addr().String()

	u := newTestUpstream(nil)
	n := 8
	cs := make([]*client, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _ := u.createClient(addr)
			cs[i] = c
		}(i)
	}

	wg.Wait()
	for i := 1; i < n; i++ {
		assert.NotNil(t, cs[i])
		assert.Equal(t, cs[0], cs[i])
		cs[i].Stop()
	}
}

func TestUpstreamGetClientFailFast(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	addr := l.Addr().String()

	// fill the listener backlog
	// TODO(kirk91): set the listener backlog to 1
	var conns []net.Conn
	for {
		conn, err := net.DialTimeout("tcp", l.Addr().String(), time.Millisecond*500)
		if err != nil {
			break
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	u := newTestUpstream(nil)
	n := 100 // 100 times is enough
	wg := new(sync.WaitGroup)
	cs := make([]*client, n)
	begin := time.Now()
	dialTimeout := time.Second
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _ := u.getClient(addr)
			cs[i] = c
		}(i)
	}

	wg.Wait()
	// all client is nil
	for _, c := range cs {
		assert.Nil(t, c)
	}
	assert.Equal(t, true, time.Since(begin) < dialTimeout*2)
}

func randStr(n int) string {
	buff := make([]byte, n)
	rand.Read(buff)
	s := base64.StdEncoding.EncodeToString(buff)
	// Base 64 can be longer than len
	return s[:n]
}

func newTestRedisInstance(t *testing.T, handler func(conn net.Conn)) (addr string, shutdown func()) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	addr = l.Addr().String()
	shutdown = func() { l.Close() }
	if handler == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		conn, err := l.Accept()
		if err != nil {
			l.Close()
			return
		}

		defer conn.Close()
		drainReadOnlyRequest(conn)
		handler(conn)
	}()

	shutdown = func() {
		l.Close()
		// TODO: add timeout check to prevent the test program block.
		<-done
	}
	return
}

//nolint:errcheck
func drainReadOnlyRequest(conn net.Conn) {
	l := len(encode(newStringArray("readonly")))
	conn.Read(make([]byte, l))
	conn.Write([]byte("+OK\r\n"))
}

func TestUpstreamRefreshSlots(t *testing.T) {
	addr, shutdown := newTestRedisInstance(t, func(conn net.Conn) {
		b := make([]byte, 1024)
		n, _ := conn.Read(b)
		// assert request
		expect := encode(newArray(
			*newBulkString("cluster"),
			*newBulkString("nodes"),
		))
		if !strings.EqualFold(string(expect), string(b[:n])) {
			conn.Close()
			return
		}
		data := fmt.Sprintf("%s %s master - 0 1528688887753 7 connected 0-16383\n", randStr(40), conn.LocalAddr().String())
		conn.Write(encode(newBulkString(data)))
	})
	defer shutdown()

	u := newTestUpstream(nil, host.New(addr))
	done := make(chan struct{})
	go func() {
		u.Serve()
		close(done)
	}()
	defer func() {
		u.Stop()
		<-done
	}()

	time.Sleep(time.Millisecond * 100)
	assert.NotEmpty(t, u.slotsLastUpdateTime)
	for i := 0; i < slotNum; i++ {
		assert.NotNil(t, u.slots[i])
	}
}

func TestUpstreamSlotsRefreshRetryOnFail(t *testing.T) {
	addr, shutdown := newTestRedisInstance(t, func(conn net.Conn) {
		// drainReadOnlyRequest(conn)
		// fail at first time
		conn.Read(make([]byte, 1024))
		conn.Write(encode(newError("internal error")))
		// success at second time
		conn.Read(make([]byte, 1024))
		data := fmt.Sprintf("%s %s master - 0 1528688887753 7 connected 0-16383\n", randStr(40), conn.LocalAddr().String())
		conn.Write(encode(newBulkString(data)))
	})
	defer shutdown()

	u := newTestUpstream(nil, host.New(addr))
	slotsRefMinRate = time.Second
	done := make(chan struct{})
	go func() {
		u.Serve()
		close(done)
	}()
	defer func() {
		u.Stop()
		<-done
	}()

	time.Sleep(slotsRefMinRate + time.Second)
	assert.NotEmpty(t, u.slotsLastUpdateTime)
	for i := 0; i < slotNum; i++ {
		assert.NotNil(t, u.slots[i])
	}
}

func TestUpstreamSlotsRefreshOnExecessiveRequests(t *testing.T) {
	addr, shutdown := newTestRedisInstance(t, func(conn net.Conn) {
		// drainReadOnlyRequest(conn)
		conn.Read(make([]byte, 1024))
		data := fmt.Sprintf("%s %s master - 0 1528688887753 7 connected 0-16383\n", randStr(40), conn.LocalAddr().String())
		conn.Write(encode(newBulkString(data)))
	})
	defer shutdown()

	u := newTestUpstream(nil, host.New(addr))
	slotsRefMinRate = time.Second * 2
	done := make(chan struct{})
	go func() {
		u.Serve()
		close(done)
	}()
	defer func() {
		u.Stop()
		<-done
	}()

	// wait until slots updated
	for {
		if !u.slotsLastUpdateTime.IsZero() {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	slotsLastUpdateTime := u.slotsLastUpdateTime

	deadline := time.Now().Add(time.Second)
	// produce excessive slots refresh requests
	for {
		if time.Now().After(deadline) {
			break
		}
		u.triggerSlotsRefresh()
	}
	// slots hasn't been updated after the last.
	assert.Equal(t, slotsLastUpdateTime, u.slotsLastUpdateTime)
}

func TestUpstreamHandleRedirection(t *testing.T) {
	u := newTestUpstream(nil)
	done := make(chan struct{})
	go func() {
		u.Serve()
		close(done)
	}()
	defer func() {
		u.Stop()
		<-done
	}()

	t.Run("moved", func(t *testing.T) {
		req := newSimpleRequest(newArray(
			*newBulkString("get"),
			*newBulkString("a"),
		))

		addr, shutdown := newTestRedisInstance(t, func(conn net.Conn) {
			// assert request
			b := make([]byte, 1024)
			n, _ := conn.Read(b)
			assert.Equal(t, encode(req.Body()), b[:n])
		})
		defer shutdown()

		assertSlots := newSlotsRefreshTriggerAssert(t, u)
		defer assertSlots()

		resp := newError("moved 123 " + addr)
		u.handleRedirection(req, resp)
		req.Wait()
	})

	t.Run("ask", func(t *testing.T) {
		req := newSimpleRequest(newArray(
			*newBulkString("get"),
			*newBulkString("a"),
		))

		addr, shutdown := newTestRedisInstance(t, func(conn net.Conn) {
			// assert request
			dec := newDecoder(conn, 2048)
			v, err := dec.Decode()
			assert.NoError(t, err)
			assert.Equal(t, ASKING, string(v.Array[0].Text))
			v, _ = dec.Decode()
			assert.Equal(t, req.Body(), v)
		})
		defer shutdown()

		assertSlots := newSlotsRefreshTriggerAssert(t, u)
		defer assertSlots()

		resp := newError("ASK 123 " + addr)
		u.handleRedirection(req, resp)
		req.Wait()
	})
}

func newSlotsRefreshTriggerAssert(t *testing.T, u *upstream) func() {
	called := false
	u.slotsRefTriggerHook = func() { called = true }
	return func() {
		assert.True(t, called)
		u.slotsRefTriggerHook = nil
	}
}

func TestUpstreamHandleClusterDown(t *testing.T) {
	u := newTestUpstream(nil)
	done := make(chan struct{})
	go func() {
		u.Serve()
		close(done)
	}()
	defer func() {
		u.Stop()
		<-done
	}()

	assertSlots := newSlotsRefreshTriggerAssert(t, u)
	defer assertSlots()

	req := newSimpleRequest(newArray(
		*newBulkString("get"),
		*newBulkString("a"),
	))
	defer req.Wait()
	resp := newError("CLUSTERDOWN the cluster is down")
	u.handleClusterDown(req, resp)
}

func TestUpstreamOnHostAdd(t *testing.T) {
	addr, shutdown := newTestRedisInstance(t, nil)
	defer shutdown()

	u := newTestUpstream(nil)
	assert.NoError(t, u.OnHostAdd(host.New(addr)))
	assert.Equal(t, 1, u.hosts.Len())
	assert.True(t, u.hosts.Exist(addr))
}

func TestUpstreamOnHostRemove(t *testing.T) {
	addr, shutdown := newTestRedisInstance(t, nil)
	defer shutdown()

	u := newTestUpstream(nil)
	assert.NoError(t, u.OnHostAdd(host.New(addr)))
	assert.True(t, u.hosts.Exist(addr))
	c, _ := u.getClient(addr)
	defer c.Stop()

	assert.NoError(t, u.OnHostRemove(host.New(addr)))
	time.Sleep(time.Millisecond * 300) // wait client close

	assert.Equal(t, 0, u.hosts.Len())
	// TODO: replace it with c.IsStopped
	assert.Empty(t, u.loadClients())
}

func TestUpstreamOnHostReplace(t *testing.T) {
	u := newTestUpstream(nil)

	// create two listeners
	addr1, shutdown1 := newTestRedisInstance(t, nil)
	defer shutdown1()
	addr2, shutdown2 := newTestRedisInstance(t, nil)
	defer shutdown2()

	// add the host of addr1
	assert.NoError(t, u.OnHostAdd(host.New(addr1)))
	assert.True(t, u.hosts.Exist(addr1))
	c1, _ := u.getClient(addr1)
	defer c1.Stop()

	// replace all with the host of addr2
	assert.NoError(t, u.OnHostReplace([]*host.Host{host.New(addr2)}))
	time.Sleep(time.Millisecond * 300) // wait the client of addr1 close
	assert.False(t, u.hosts.Exist(addr1))
	assert.Nil(t, u.loadClients()[addr1])

	c2, _ := u.getClient(addr2)
	defer c2.Stop()
	assert.NotNil(t, u.loadClients()[addr2])
	assert.True(t, u.hosts.Exist(addr2))
}
