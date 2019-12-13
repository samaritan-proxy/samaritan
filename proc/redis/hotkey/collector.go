package hotkey

import (
	"math/rand"
	"sort"
	"sync"
	"time"
)

const (
	// TODO: adjust the value
	logFactor = 10
	// max logarithmic counter value
	maxLogCounterValue = 255
)

// logCounter is a logarithmic counter which is consistent with the redis lfu counter
// implementation. Refer to: https://github.com/antirez/redis/blob/5.0/src/evict.c#L260
// http://antirez.com/news/109
type logCounter struct {
	lut int64 // last update time in minute
	val uint8
}

// Value returns the counter value.
func (c *logCounter) Value() uint8 {
	return c.val
}

// Incrby increments the counter value.
func (c *logCounter) Incrby(delta uint64) {
	for i := uint64(0); i < delta; i++ {
		if c.val == maxLogCounterValue {
			break
		}
		c.incr()
	}
	c.lut = time.Now().Unix() / 60
}

func (c *logCounter) incr() {
	if c.val == maxLogCounterValue {
		return
	}

	r := rand.Float64()
	p := 1 / float64(c.val*logFactor*10+1)
	if r < p {
		c.val++
	}
	return
}

// Decr decrements the counter value.
func (c *logCounter) Decr() {
	if c.val == 0 {
		return
	}
	c.val--
	c.lut = time.Now().Unix() / 60
}

// LastUpdateTime returns the last update time in minutes of counter.
func (c *logCounter) LastUpdateTime() int64 {
	return c.lut
}

// HotKey represents a hot key.
type HotKey struct {
	Name    string
	Counter *logCounter
}

var (
	defaultCollectInterval = time.Second * 10
	defaultEvictInterval   = time.Minute
)

// Collector is used to collect hot keys.
type Collector struct {
	rwmu            sync.RWMutex
	capacity        uint8
	collectInterval time.Duration
	evictInterval   time.Duration

	// keys cache the most visited keys recently, and they're in descending
	// order by the key's counter value.
	keys []HotKey
	// counters is used to record the actual hit count of a key over a period
	// of time. To reduce the impact on performance, each redis client have its own.
	counters map[string]*Counter
}

// NewCollector creates a hot keys collector with given parameters.
func NewCollector(capacity uint8) *Collector {
	c := &Collector{
		capacity:        capacity,
		collectInterval: defaultCollectInterval,
		evictInterval:   defaultEvictInterval,
		counters:        make(map[string]*Counter),
	}
	return c
}

// Run runs the collector unitl receive the stop signal.
func (c *Collector) Run(stop <-chan struct{}) {
	collectTicker := time.NewTicker(c.collectInterval)
	evictTicker := time.NewTicker(c.evictInterval)
	defer func() {
		collectTicker.Stop()
		evictTicker.Stop()
	}()

	for {
		select {
		case <-stop:
			return
		case <-collectTicker.C:
			c.collect()
		case <-evictTicker.C:
			c.evictStale()

			// TODO: print hot keys regularly
		}
	}
}

// AllocCounter allocates a counter which is used to record the actual hit count
// of visited keys. The allocated counter is not goroutine-safe, every redis client
// should have its own.
func (c *Collector) AllocCounter(name string) *Counter {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()
	counter, ok := c.counters[name]
	if ok {
		return counter
	}

	counter = newCounter(c.capacity)
	c.counters[name] = counter
	return counter
}

// HotKeys returns the collected hot keys which are sorted by logarithmic hit count.
func (c *Collector) HotKeys() []HotKey {
	c.rwmu.RLock()
	defer c.rwmu.RUnlock()
	return c.keys
}

func (c *Collector) collect() {
	// get all keys and acutal hit count in the current period.
	c.rwmu.RLock()
	allHitKeys := make(map[string]uint64)
	for _, counter := range c.counters {
		for key, hitCount := range counter.Latch() {
			allHitKeys[key] += hitCount
		}
	}
	c.rwmu.RUnlock()
	if len(allHitKeys) == 0 {
		return
	}

	// make map of the current hot key counters.
	c.rwmu.RLock()
	curHotKeys := make(map[string]*logCounter, len(c.keys))
	for _, key := range c.keys {
		curHotKeys[key.Name] = key.Counter
	}
	c.rwmu.RUnlock()

	// generate the new hot keys.
	newHotKeys := make([]HotKey, 0, c.capacity)
	tryInsert := func(key HotKey) {
		l := len(newHotKeys)
		i := sort.Search(l, func(i int) bool {
			return newHotKeys[i].Counter.Value() <= key.Counter.Value()
		})

		if i < l {
			copy(newHotKeys[i+1:], newHotKeys[i:])
			newHotKeys[i] = key
		} else if uint8(l) < c.capacity {
			newHotKeys = append(newHotKeys, key)
		}
	}

	for keyName, hitCount := range allHitKeys {
		counter := curHotKeys[keyName]
		if counter == nil {
			counter = new(logCounter)
		}
		counter.Incrby(hitCount)
		key := HotKey{
			Name:    keyName,
			Counter: counter,
		}
		tryInsert(key)
	}

	// update the cached hot keys
	c.rwmu.Lock()
	c.keys = newHotKeys
	c.rwmu.Unlock()
}

func (c *Collector) evictStale() {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()

	// decrement counter
	curTimeInMinute := time.Now().Unix() / 60
	for _, key := range c.keys {
		counter := key.Counter
		if curTimeInMinute > counter.LastUpdateTime() {
			counter.Decr()
		}
	}

	// remove stale
	keys := make([]HotKey, 0, len(c.keys))
	for _, key := range c.keys {
		if key.Counter.Value() != 0 {
			keys = append(keys, key)
		}
	}
	c.keys = keys
}
