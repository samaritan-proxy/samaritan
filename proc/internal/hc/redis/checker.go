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
	"time"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/atcp"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
)

// Redis packets to be sent or expected.
var (
	PING = []byte("*1\r\n$4\r\nPING\r\n")
	PONG = []byte("+PONG\r\n")
)

// Checker does health checking with Redis protocol.
type Checker struct{}

// NewChecker creates RedisChecker.
func NewChecker(config *hc.RedisChecker) *Checker {
	return &Checker{}
}

// Check checks given host with timeout.
// TODO: add auth
func (c *Checker) Check(addr string, timeout time.Duration) error {
	var deadline = time.Now().Add(timeout)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err = atcp.SendBytesWithDeadline(conn, PING, deadline); err != nil {
		return err
	}
	if err = atcp.ExpectBytesWithDeadline(conn, PONG, deadline); err != nil {
		return err
	}
	return nil
}
