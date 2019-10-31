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

package net

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/stats"
)

func newLocalListener(t *testing.T) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestSetReadTimeout(t *testing.T) {
	l := newLocalListener(t)
	defer l.Close()

	rawConn, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	conn := New(rawConn)
	defer conn.Close()
	conn.SetReadTimeout(time.Millisecond * 50)
	_, err = conn.Read(make([]byte, 64))
	assert.Error(t, err)
}

func TestSetWriteTimeout(t *testing.T) {
	l := newLocalListener(t)
	defer l.Close()

	rawConn, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	conn := New(rawConn)
	defer conn.Close()
	conn.SetWriteTimeout(time.Millisecond * 50)

	// write until the buffer is full.
	for {
		_, err = conn.Write([]byte("WRITE TIMEOUT TEST"))
		if err != nil {
			break
		}
	}
	assert.Error(t, err)
}

func TestSetStats(t *testing.T) {
	l := newLocalListener(t)
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		conn.Write([]byte("hello"))
		conn.Read(make([]byte, 64))
	}()

	rawConn, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	conn := New(rawConn)
	hookCalled := 0
	conn.recordDurationHook = func() {
		hookCalled++
	}
	assert.NotEqual(t, conn.createdAt, time.Time{})
	defer func() {
		conn.Close()
		assert.Equal(t, hookCalled, 1)
	}()

	scope := stats.CreateScope("")
	connStats := &Stats{
		WriteTotal: scope.Counter("cx_write_total"),
		ReadTotal:  scope.Counter("cx_read_total"),
		Duration:   scope.Histogram("cx_length"),
	}
	conn.SetStats(connStats)

	conn.Read(make([]byte, 64))
	conn.Write([]byte("hello"))
	assert.Equal(t, connStats.ReadTotal.Value(), uint64(5))
	assert.Equal(t, connStats.WriteTotal.Value(), uint64(5))

	conn.Close()
}

func TestCloseWrite(t *testing.T) {
	l := newLocalListener(t)
	defer l.Close()

	rawConn, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn := New(rawConn)
	defer conn.Close()

	conn.CloseWrite()
	_, err = conn.Write([]byte("hello"))
	assert.Error(t, err)
}

func TestCloseRead(t *testing.T) {
	l := newLocalListener(t)
	defer l.Close()

	rawConn, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn := New(rawConn)
	defer conn.Close()

	conn.CloseRead()
	_, err = conn.Read(make([]byte, 64))
	assert.Error(t, err)
}

func TestNewConnIdempotent(t *testing.T) {
	rawConn := new(net.TCPConn)
	c := New(rawConn)
	c1 := New(c)
	if c1 != c {
		t.Error("New dit not detect underlying Conn")
	}
}
