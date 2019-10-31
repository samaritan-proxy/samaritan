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

package utils

import (
	"net"
	"sync"
	"testing"
)

// TestServerHandler handles incomming connections.
type TestServerHandler func(*net.TCPConn)

// TestServer is a TCP server for testing
type TestServer struct {
	*net.TCPListener
	t       testing.TB
	addr    string
	handle  TestServerHandler
	serveWG sync.WaitGroup
	started chan bool
}

// Start starts listening on given address
func (ts *TestServer) Start() error {
	var tcpAddr *net.TCPAddr
	var err error
	// Using existing addr if any
	if ts.TCPListener != nil {
		tcpAddr = ts.TCPListener.Addr().(*net.TCPAddr)
	}
	if tcpAddr == nil {
		tcpAddr, err = net.ResolveTCPAddr("tcp", ts.addr)
		if err != nil {
			return err
		}
	}

	if ts.TCPListener, err = net.ListenTCP("tcp", tcpAddr); err != nil {
		return err
	}
	go ts.serve()
	<-ts.started
	return nil
}
func (ts *TestServer) defaultHandler(c *net.TCPConn) {
	c.Close()
	ts.t.Logf("Connection closed: %v", c)
}

// serve accepts incoming connections.
func (ts *TestServer) serve() {
	handle := ts.handle
	if handle == nil {
		// Use default handler if not provided
		handle = ts.defaultHandler
	}
	ts.serveWG.Add(1)
	defer ts.serveWG.Done()
	ts.started <- true
	for {
		ts.t.Logf("Waiting for connection.")
		c, err := ts.TCPListener.AcceptTCP()
		if err != nil {
			return
		}
		ts.t.Logf("Connection accepted: %v", c)
		go handle(c)
	}
}

// Port returns the listening port number.
func (ts *TestServer) Port() int {
	if ts.TCPListener != nil {
		return ts.TCPListener.Addr().(*net.TCPAddr).Port
	}
	return -1
}

// Close stops listening and wait for all connections to complete
func (ts *TestServer) Close() error {
	ts.t.Logf("Closing server.")
	err := ts.TCPListener.Close()
	if err != nil {
		return err
	}
	ts.serveWG.Wait()
	ts.t.Logf("Server closed.")
	return err
}

// StartTestServer launches a TCP server at given address
func StartTestServer(t testing.TB, addr string, handlers ...TestServerHandler) *TestServer {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	var handle TestServerHandler
	if len(handlers) > 0 {
		handle = handlers[0]
	}
	ts := &TestServer{t: t, addr: addr, handle: handle, started: make(chan bool)}
	if err := ts.Start(); err != nil {
		t.Fatal("Launching test server failed: ", err)
	}
	t.Logf("Test server listens at %s", ts.Addr())
	return ts
}
