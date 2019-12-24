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
	case "eval", "cluster", "auth":
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
