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

package mysql

import (
	"bytes"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/utils"
)

const greeting = "4a0000000a352e362e333000277f1a00277e6b7d53647d6900fff70802007f801500000000000000000000763646404c3c2a5c7041306c006d7973716c5f6e61746976655f70617373776f726400"
const responseOK = "70000020000000200000"

// handshakeResponse is a MySQL packet with "health_check" as username.
const handshakeResponse = "2e00000100820000000000012100000000000000000000000000000000000000000000006865616c74685f636865636b0000"

var (
	responseOKBytes        []byte
	greetingBytes          []byte
	handshakeResponseBytes []byte
)

var mySQLConfig = &hc.MySQLChecker{Username: "health_check"}

func init() {
	var err error
	for _, c := range []struct {
		S string
		B *[]byte
	}{
		{greeting, &greetingBytes},
		{responseOK, &responseOKBytes},
		{handshakeResponse, &handshakeResponseBytes},
	} {
		if *c.B, err = hex.DecodeString(c.S); err != nil {
			panic(err)
		}
	}
}

func TestMySQLCheckerSent(t *testing.T) {
	var read = make(chan struct{})
	srv := utils.StartTestServer(t, "", func(conn *net.TCPConn) {
		defer close(read)
		expected := append(handshakeResponseBytes, ComQUIT...)
		buf := make([]byte, len(expected))
		if _, err := conn.Read(buf); err != nil {
			logger.Fatalf("Reading from checker error: %s", err)
		}
		if !bytes.Equal(buf, expected) {
			logger.Fatal("Unexpected MySQL health checking packets")
		}
	})
	defer srv.Close()

	checker, err := NewChecker(mySQLConfig)
	assert.NoError(t, err)
	err = checker.Check(srv.Addr().String(), time.Second)
	if err == nil {
		logger.Fatalf("Health check should have failed")
	}
	select {
	case <-read:
	case <-time.After(time.Second):
		logger.Fatal("Waiting for checker timed out")
	}
}

func TestMySQLCheckerNormal(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		// Emulate a Good MySQL server.
		conn.Write(greetingBytes)
		conn.Write(responseOKBytes)
	})
	assert.Nil(t, err, "Health check failed: %s", err)
}

func TestMySQLCheckerGreetingOnce(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		conn.Write(greetingBytes)
	})
	assert.NotNil(t, err, "Health check should have failed")
}

func TestMySQLCheckerKeepGreeting(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		var e error
		for e == nil {
			_, e = conn.Write(greetingBytes)
		}
	})
	assert.NotNil(t, err, "Health check should have failed")
}

func TestMySQLCheckerOKOnce(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		conn.Write(responseOKBytes)
	})
	assert.NotNil(t, err, "Health check should have failed")
}

// TODO: Uncomment the following after Greeting packet could be verified.
//
// func TestMySQLCheckerKeepOKing(t *testing.T) {
// 	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
// 		var e error
// 		for e == nil {
// 			_, e = conn.Write(responseOKBytes)
// 		}
// 	})
// 	assert.NotNil(t, err, "Health check should have failed")
// }

func TestMySQLCheckerNoResponse(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		// Do nothing.
	})
	assert.NotNil(t, err, "Health check should have failed")
}

func TestMySQLCheckerClose(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		conn.Close()
	})
	assert.NotNil(t, err, "Health check should have failed")
}

func TestMySQLCheckerCloseLater(t *testing.T) {
	err := testMySQLCheckerWithTestServer(t, func(conn *net.TCPConn) {
		time.Sleep(time.Millisecond * 800)
		conn.Close()
	})
	assert.NotNil(t, err, "Health check should have failed")
}

func testMySQLCheckerWithTestServer(t *testing.T, handler utils.TestServerHandler) error {
	srv := utils.StartTestServer(t, "", handler)
	defer srv.Close()
	checker, err := NewChecker(mySQLConfig)
	assert.NoError(t, err)
	return checker.Check(srv.Addr().String(), time.Second)
}
