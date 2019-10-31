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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kavu/go_reuseport"
	"github.com/stretchr/testify/assert"
)

func TestCheckPort(t *testing.T) {
	ln, err := net.Listen("tcp", ":23333")
	assert.NoError(t, err)
	defer func() {
		if ln != nil {
			ln.Close()
		}
	}()
	assert.False(t, CheckPort(23333))
	assert.True(t, CheckPort(23334))
}

func TestCheckPortWithReusePort(t *testing.T) {
	cases := []struct {
		port  int
		reuse bool
		ln    net.Listener
	}{
		{port: 23333, reuse: false},
		{port: 23334, reuse: true},
	}
	for _, c := range cases {
		if c.reuse {
			ln, err := reuseport.Listen("tcp", fmt.Sprintf(":%d", c.port))
			assert.NoError(t, err)
			c.ln = ln
		} else {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
			assert.NoError(t, err)
			c.ln = ln
		}
		assert.Equal(t, c.reuse, CheckPortWithReusePort(c.port))
	}

	assert.True(t, CheckPortWithReusePort(23335))

	for _, c := range cases {
		if c.ln != nil {
			c.ln.Close()
		}
	}
}

func TestGetPortByRange(t *testing.T) {
	st := time.Now()
	port, err := GetPortByRange(10000, 10010, time.Second)
	dur := time.Since(st)
	assert.NoError(t, err)
	if dur > time.Second {
		t.Fatal("unexpected timeout")
	}
	if port < 10000 || port >= 10010 {
		t.Fatal("port out of range")
	}
}
