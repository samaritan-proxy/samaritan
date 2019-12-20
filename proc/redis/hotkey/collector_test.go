package hotkey

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogrithmCounter(t *testing.T) {
	for _, i := range []uint64{
		100,
		1000,
		10000,
		100000,
		500000,
	} {
		t.Run(fmt.Sprintf("%d hits", i), func(t *testing.T) {
			c := new(logrithmCounter)
			c.ReaptIncr(i)

			timeInMinute := time.Now().Unix() / 60
			lut := c.LastUpdateTimeInMinute()
			assert.Equal(t, timeInMinute, lut)

			// halve
			count := 0
			for {
				c.Halve()
				count++
				if c.Value() == 0 {
					break
				}
			}
			assert.True(t, count <= 8)
		})
	}
}

func repeatHit(c *Counter, key string, times int) {
	for i := 0; i < times; i++ {
		c.Hit(key)
	}
}

func TestCollectorCollectAndMerge(t *testing.T) {
	c := NewCollector(4, withCollectInterval(time.Millisecond*100))
	stopCh := make(chan struct{})
	go c.Run(stopCh)
	defer close(stopCh)

	// alloc counters
	counter1 := c.AllocCounter("node1")
	counter2 := c.AllocCounter("node2")
	defer func() {
		counter1.Free()
		counter2.Free()
	}()

	// k1:9901 k2:3000 k3:300 k4:50 k5:1
	repeatHit(counter1, "k1", 1)
	repeatHit(counter1, "k2", 3000)
	repeatHit(counter2, "k1", 9900)
	repeatHit(counter2, "k3", 300)
	repeatHit(counter2, "k4", 50)
	repeatHit(counter2, "k5", 1)

	assertKeys := func(t *testing.T, c *Collector, expectedKeyNames []string) {
		t.Helper()
		var keyNames []string
		for _, key := range c.HotKeys() {
			keyNames = append(keyNames, key.Name)
		}
		assert.Equal(t, expectedKeyNames, keyNames)
	}

	time.Sleep(time.Millisecond * 150) // wait collect and merge
	assertKeys(t, c, []string{"k1", "k2", "k3", "k4"})

	// update k4 hit count
	repeatHit(counter2, "k4", 100000)
	time.Sleep(time.Millisecond * 150) // wait collect and merge
	assertKeys(t, c, []string{"k4", "k1", "k2", "k3"})
}

func TestCollectorEvictStale(t *testing.T) {
	// mock time
	oldFn := nowInMinute
	defer func() { nowInMinute = oldFn }()
	var curTimeInMinute int64
	nowInMinute = func() int64 {
		curTimeInMinute++
		return curTimeInMinute
	}

	options := []collectorOption{
		withCollectInterval(time.Millisecond * 50),
		withEvictInterval(time.Millisecond * 150),
	}
	c := NewCollector(4, options...)
	counter := c.AllocCounter("node1")
	repeatHit(counter, "k1", 100)

	stopCh := make(chan struct{})
	go c.Run(stopCh)
	defer close(stopCh)

	time.Sleep(time.Millisecond * 80)
	assert.Len(t, c.HotKeys(), 1)

	// wait evict k1
	time.Sleep(time.Millisecond * 500)
	assert.Len(t, c.HotKeys(), 0)
}
