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
	"testing"

	"github.com/samaritan-proxy/samaritan/proc/redis/hotkey"
	"github.com/stretchr/testify/assert"
)

func TestHotKeyFilterDo(t *testing.T) {
	c := hotkey.NewCounter(4, nil)
	f := newHotKeyFilter(c)
	defer f.Destroy()

	scenarios := []struct {
		cmd string
		rv  *RespValue
	}{
		// commands without params
		{cmd: "ping", rv: newStringArray("ping")},
		// special commands with params
		{cmd: "eval", rv: newStringArray("eval", "xxx")},
		{cmd: "auth", rv: newStringArray("auth", "passwd")},
		{cmd: "cluster", rv: newStringArray("cluster", "nodes")},
		// normal commands
		{cmd: "get", rv: newStringArray("get", "a")},
		{cmd: "set", rv: newStringArray("set", "a", "1")},
	}
	for _, scenario := range scenarios {
		req := newSimpleRequest(scenario.rv)
		f.Do(scenario.cmd, req)
	}

	// assert keys
	keys := c.Latch()
	assert.Len(t, keys, 1)
	assert.EqualValues(t, 2, keys["a"])
}
