package hotkey

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterHit(t *testing.T) {
	c := newCounter(3, nil)

	c.Hit("key1")
	c.Hit("key2")
	c.Hit("key2")
	c.Hit("key3")
	// exceed capacity
	for i := 0; i < 3; i++ {
		c.Hit("key4")
	}

	res := c.Latch()
	assert.Len(t, res, 3)
	assert.EqualValues(t, 2, res["key2"])
	assert.EqualValues(t, 1, res["key3"])
	assert.EqualValues(t, 3, res["key4"])
}

func TestCounterLatch(t *testing.T) {
	c := newCounter(3, nil)
	for i := 1; i < 4; i++ {
		c.Hit("key" + strconv.Itoa(i))
	}
	c.Latch()
	c.Hit("key5")
	res := c.Latch()
	assert.Len(t, res, 1)
	assert.EqualValues(t, 1, res["key5"])
}

func TestCounterFree(t *testing.T) {
	var called bool
	freeCb := func() { called = true }
	c := newCounter(3, freeCb)
	c.Free()
	assert.True(t, called)
}
