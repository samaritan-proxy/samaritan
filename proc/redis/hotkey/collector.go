package hotkey

import (
	"math/rand"
	"sort"
	"sync"
	"time"
)

// the acutal counter value that can be represented is 50k which is enough for a proxy.
const defaultLogFactor = 10

var nowInMinute = func() int64 {
	return time.Now().Unix() / 60
}

// logrithmCounter is a logarithmic counter which is insipred by redis lfu counter.
// The incremental implementation is the same as redis except default value
// and decay logic. In current implementation, the default value is zero and
// is halved over time.
//
// More redis lfu counter details see:
// 1. https://github.com/antirez/redis/blob/5.0/src/evict.c#L260
// 2. http://antirez.com/news/109
type logrithmCounter struct {
	rnd *rand.Rand
	lut int64 // last update time in minute
	val uint8
}

// Value returns the counter value.
func (c *logrithmCounter) Value() uint8 {
	return c.val
}

// Incr increments the counter value N times.
func (c *logrithmCounter) ReaptIncr(times uint64) {
	// lazy init rand.
	if c.rnd == nil {
		c.rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	for i := uint64(0); i < times; i++ {
		if c.val == 255 {
			break
		}
		r := c.rnd.Float64()
		p := 1.0 / float64(float64(c.val)*defaultLogFactor+1)
		if r < p {
			c.val++
		}
	}
	c.lut = nowInMinute()
}

// Halve halves the counter value.
func (c *logrithmCounter) Halve() {
	if c.val == 0 {
		return
	}

	c.val = c.val >> 1
	c.lut = nowInMinute()
}

// LastUpdateTimeInMinute returns the last update time in minutes of counter.
func (c *logrithmCounter) LastUpdateTimeInMinute() int64 {
	return c.lut
}

// HotKey represents a hot key.
type HotKey struct {
	Name    string
	Counter *logrithmCounter
}

type sortedHotKeys struct {
	capacity uint8
	data     []HotKey
}

func newSortedHotKeys(capacity uint8) *sortedHotKeys {
	return &sortedHotKeys{
		capacity: capacity,
		data:     make([]HotKey, 0, capacity),
	}
}

func (s *sortedHotKeys) Insert(key HotKey) (success bool) {
	l := len(s.data)
	i := sort.Search(l, func(i int) bool {
		return s.data[i].Counter.Value() <= key.Counter.Value()
	})

	if uint8(l) < s.capacity {
		s.data = append(s.data, key)
		success = true
	}
	if i < l {
		copy(s.data[i+1:], s.data[i:])
		s.data[i] = key
		success = true
	}
	return
}

func (s *sortedHotKeys) Data() []HotKey {
	return s.data
}

var (
	defaultCollectInterval = time.Second * 10
	defaultEvictInterval   = time.Minute
)

type collectorOption func(*Collector)

func withCollectInterval(interval time.Duration) collectorOption {
	return func(c *Collector) {
		c.collectInterval = interval
	}
}

func withEvictInterval(interval time.Duration) collectorOption {
	return func(c *Collector) {
		c.evictInterval = interval
	}
}

// Collector is used to collect hot keys.
type Collector struct {
	rwmu            sync.RWMutex
	capacity        uint8
	collectInterval time.Duration
	evictInterval   time.Duration

	// keys cache the most visited keys recently, and they're in descending
	// order by the key's counter value.
	keys []HotKey
	// hitter is used to record the actual hits of a key over a period
	// of time. To reduce the impact on performance, each redis client have its own.
	counters map[string]*Counter
}

// NewCollector creates a hot keys collector with given parameters.
func NewCollector(capacity uint8, options ...collectorOption) *Collector {
	c := &Collector{
		capacity:        capacity,
		collectInterval: defaultCollectInterval,
		evictInterval:   defaultEvictInterval,
		counters:        make(map[string]*Counter),
	}

	// apply options
	for _, option := range options {
		option(c)
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

// AllocCounter allocates a general counter which is used to record the actual visits of keys.
func (c *Collector) AllocCounter(name string) *Counter {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()
	counter, ok := c.counters[name]
	if ok {
		return counter
	}

	cb := func() {
		c.rwmu.Lock()
		delete(c.counters, name)
		c.rwmu.Unlock()
	}
	counter = NewCounter(c.capacity, cb)
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
	// get all keys and acutal visits in the current period.
	c.rwmu.RLock()
	accessedKeyNames := make(map[string]uint64)
	for _, counter := range c.counters {
		for key, hitCount := range counter.Latch() {
			accessedKeyNames[key] += hitCount
		}
	}
	c.rwmu.RUnlock()
	if len(accessedKeyNames) == 0 {
		return
	}

	// make map of the current hot key counters.
	c.rwmu.RLock()
	curHotKeys := make(map[string]*logrithmCounter, len(c.keys))
	for _, key := range c.keys {
		curHotKeys[key.Name] = key.Counter
	}
	c.rwmu.RUnlock()

	// generate the new hot keys.
	res := newSortedHotKeys(c.capacity)
	for keyName, counter := range curHotKeys {
		visits := accessedKeyNames[keyName]
		counter.ReaptIncr(visits)
		key := HotKey{Name: keyName, Counter: counter}
		res.Insert(key)
		delete(accessedKeyNames, keyName)
	}

	for keyName, hitCount := range accessedKeyNames {
		counter := new(logrithmCounter)
		counter.ReaptIncr(hitCount)
		key := HotKey{Name: keyName, Counter: counter}
		res.Insert(key)
	}

	// update the cached hot keys
	c.rwmu.Lock()
	c.keys = res.Data()
	c.rwmu.Unlock()
}

func (c *Collector) evictStale() {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()

	// halve counter
	curTimeInMinute := nowInMinute()
	for _, key := range c.keys {
		counter := key.Counter
		if curTimeInMinute > counter.LastUpdateTimeInMinute() {
			counter.Halve()
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
