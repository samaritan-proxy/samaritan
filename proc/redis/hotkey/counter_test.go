package hotkey

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterIncr(t *testing.T) {
	c := NewCounter(3, nil)

	c.Incr("key1")
	c.Incr("key2")
	c.Incr("key2")
	c.Incr("key3")
	// exceed capacity
	for i := 0; i < 3; i++ {
		c.Incr("key4")
	}

	res := c.Latch()
	assert.Len(t, res, 3)
	assert.EqualValues(t, 2, res["key2"])
	assert.EqualValues(t, 1, res["key3"])
	assert.EqualValues(t, 3, res["key4"])
}

func TestCounterLatch(t *testing.T) {
	c := NewCounter(3, nil)
	for i := 1; i < 4; i++ {
		c.Incr("key" + strconv.Itoa(i))
	}
	c.Latch()
	c.Incr("key5")
	res := c.Latch()
	assert.Len(t, res, 1)
	assert.EqualValues(t, 1, res["key5"])
}

func TestCounterFree(t *testing.T) {
	var called bool
	freeCb := func() { called = true }
	c := NewCounter(3, freeCb)
	c.Free()
	assert.True(t, called)
}
