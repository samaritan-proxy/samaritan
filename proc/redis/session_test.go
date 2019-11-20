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
	"net"
	"testing"
	"time"
)

func TestSessionClose(t *testing.T) {
	conn, _ := net.Pipe()
	p := newRedisTestProc(t)

	s := newSession(p, conn)
	done := make(chan struct{})
	go func() {
		s.Serve()
		close(done)
	}()

	time.Sleep(time.Millisecond * 500)
	s.Close()
	<-done
}

func TestSessionReadError(t *testing.T) {
	cconn, sconn := net.Pipe()
	p := newRedisTestProc(t)

	s := newSession(p, cconn)
	done := make(chan struct{})
	go func() {
		s.Serve()
		close(done)
	}()

	time.AfterFunc(time.Millisecond*100, func() {
		sconn.Close()
	})
	<-done
}

func TestSessionWriteError(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Error(err)
		return
	}
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	p := newRedisTestProc(t)
	s := newSession(p, conn)
	done := make(chan struct{})
	go func() {
		s.Serve()
		close(done)
	}()

	time.AfterFunc(time.Millisecond*100, func() {
		conn.(*net.TCPConn).CloseWrite()

		// make a request
		sconn, _ := l.Accept()
		sconn.Write(encode(newArray(
			*newBulkString("get"),
			*newBulkString("a"),
		)))
		<-done
		sconn.Close()
	})
	<-done
}
