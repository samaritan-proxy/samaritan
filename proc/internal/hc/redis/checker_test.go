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
	"io"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/utils"
)

func TestRedisCheckerSendAndRecv(t *testing.T) {
	var wg sync.WaitGroup
	srv := utils.StartTestServer(t, "", func(conn *net.TCPConn) {
		var buf = make([]byte, len(PING))
		_, err := io.ReadFull(conn, buf)
		if err != nil {
			t.Fatal("Read error: ", err)
		}
		if !bytes.Equal(PING, buf) {
			t.Fatalf("Unexpected response: '%s'", buf)
		}
		t.Log("Expected response received")
		_, err = conn.Write(PONG)
		assert.NoError(t, err)

		t.Log("PONG was sent, expecting EOF from client")
		conn.SetDeadline(time.Now().Add(time.Second))
		buf, err = ioutil.ReadAll(conn)
		assert.Equal(t, nil, err)
		assert.Equal(t, 0, len(buf))

		t.Log("EOF received from client")
		wg.Done()
	})
	defer srv.Close()
	wg.Add(1)
	checker := NewChecker(nil)
	err := checker.Check(srv.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Waiting for server")
	wg.Wait()
}
