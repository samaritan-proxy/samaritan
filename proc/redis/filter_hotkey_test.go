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
