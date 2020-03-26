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

package hotkey

import (
	"sync"
)

// Counter is used to record the actual visits number of keys.
// To minimize the memory footprint, only the keys that are frequently
// accessed are recorded.
//
// It's not goroutine-safe, every redis client should have its own.
type Counter struct {
	mu       sync.Mutex
	capacity uint8
	freeCb   func()
	items    map[string]*itemNode
	freqHead *freqNode
}

// NewCounter creates a counter with given parameters.
// NOTE: It's only for test, don't rely on it directly.
func NewCounter(capacity uint8, freeCb func()) *Counter {
	return &Counter{
		capacity: capacity,
		freeCb:   freeCb,
		items:    make(map[string]*itemNode),
	}
}

// Incr increases the specified key visits.
func (c *Counter) Incr(key string) {
	c.mu.Lock()
	item, ok := c.items[key]
	if ok {
		// update the item's freq
		c.increment(item)
		c.mu.Unlock()
		return
	}

	// record
	if uint8(len(c.items)) >= c.capacity {
		// evict the least frequently used item
		c.evict()
	}
	item = &itemNode{key: key}
	c.add(item)
	c.mu.Unlock()
}

// Latch returns the cached key visits and reset it.
func (c *Counter) Latch() map[string]uint64 {
	c.mu.Lock()
	res := make(map[string]uint64, len(c.items))
	for key, item := range c.items {
		res[key] = item.freqNode.freq
	}
	c.reset()
	c.mu.Unlock()
	return res
}

func (c *Counter) reset() {
	// TODO: reuse the old datastructures.
	c.items = make(map[string]*itemNode)
	c.freqHead = nil
}

func (c *Counter) increment(item *itemNode) {
	curFreqNode := item.freqNode
	curFreq := curFreqNode.freq

	var targetFreqNode *freqNode
	if curFreqNode.next == nil || curFreqNode.next.freq != curFreq+1 {
		targetFreqNode = &freqNode{freq: curFreq + 1}
		curFreqNode.InsertAfterMe(targetFreqNode)
	} else {
		targetFreqNode = curFreqNode.next
	}

	item.Free()
	targetFreqNode.AppendItem(item)

	if curFreqNode.itemHead != nil {
		return
	}

	// remove current freq node if empty
	if c.freqHead == curFreqNode {
		c.freqHead = targetFreqNode
	}
	curFreqNode.Free()
}

func (c *Counter) add(item *itemNode) {
	c.items[item.key] = item

	if c.freqHead != nil && c.freqHead.freq == 1 {
		c.freqHead.AppendItem(item)
		return
	}

	fnode := &freqNode{freq: 1}
	fnode.AppendItem(item)
	if c.freqHead != nil {
		c.freqHead.InsertBeforeMe(fnode)
	}
	c.freqHead = fnode
}

func (c *Counter) evict() {
	fnode := c.freqHead
	item := fnode.itemHead
	delete(c.items, item.key)
	fnode.PopItem()

	if fnode.itemHead != nil {
		return
	}

	c.freqHead = fnode.next
	fnode.Free()
}

// Free frees the counter.
func (c *Counter) Free() {
	if c.freeCb != nil {
		c.freeCb()
	}

	c.mu.Lock()
	c.reset()
	c.mu.Unlock()
}

type freqNode struct {
	freq               uint64
	prev, next         *freqNode
	itemHead, itemTail *itemNode
}

func (n *freqNode) PopItem() *itemNode {
	if n.itemHead == nil {
		return nil
	}

	// only have one item
	if n.itemHead == n.itemTail {
		item := n.itemHead
		n.itemHead = nil
		n.itemTail = nil
		return item
	}

	item := n.itemHead
	item.next.prev = nil
	n.itemHead = item.next
	return item
}

func (n *freqNode) AppendItem(item *itemNode) {
	item.freqNode = n

	if n.itemHead == nil {
		n.itemHead = item
		n.itemTail = item
		return
	}

	item.prev = n.itemTail
	item.next = nil
	n.itemTail.next = item
	n.itemTail = item
}

func (n *freqNode) InsertBeforeMe(o *freqNode) {
	if n.prev != nil {
		n.prev.next = o
	}
	o.prev = n.prev
	o.next = n
	n.prev = o
}

func (n *freqNode) InsertAfterMe(o *freqNode) {
	o.next = n.next
	if n.next != nil {
		n.next.prev = o
	}
	n.next = o
	o.prev = n
}

func (n *freqNode) Free() {
	if n.prev != nil {
		n.prev.next = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	}
	n.prev = nil
	n.next = nil
	n.itemHead, n.itemTail = nil, nil
}

type itemNode struct {
	key        string
	freqNode   *freqNode
	prev, next *itemNode
}

func (n *itemNode) Free() {
	fnode := n.freqNode
	switch {
	case fnode.itemHead == fnode.itemTail:
		fnode.itemHead, fnode.itemTail = nil, nil
	case fnode.itemHead == n:
		n.next.prev = nil
		fnode.itemHead = n.next
	case fnode.itemTail == n:
		n.prev.next = nil
		fnode.itemTail = n.prev
	default:
		n.prev.next = n.next
		n.next.prev = n.prev
	}

	// remove all links
	n.prev = nil
	n.next = nil
	n.freqNode = nil
}
