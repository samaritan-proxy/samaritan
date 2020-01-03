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
	"github.com/samaritan-proxy/samaritan/proc/redis/hotkey"
)

type hotKeyFilter struct {
	counter *hotkey.Counter
}

func newHotKeyFilter(counter *hotkey.Counter) *hotKeyFilter {
	return &hotKeyFilter{
		counter: counter,
	}
}

func (f *hotKeyFilter) Do(cmd string, req *simpleRequest) FilterStatus {
	key := f.extractKey(cmd, req.Body())
	if len(key) > 0 && f.counter != nil {
		f.counter.Incr(key)
	}
	return Continue
}

func (f *hotKeyFilter) extractKey(cmd string, v *RespValue) string {
	if len(v.Array) <= 1 {
		return ""
	}

	switch cmd {
	case "eval", "cluster", "auth", "scan":
		return ""
	default:
		// TODO: truncate key
		return string(v.Array[1].Text)
	}
}

func (f *hotKeyFilter) Destroy() {
	if f.counter != nil {
		f.counter.Free()
	}
}
