package hotkey

import "sync"

// Counter is used to record the actual hit count of visited keys.
// To minimize the memory footprint, only the keys that are frequently
// accessed are recorded.
//
// It's not goroutine-safe, every redis client should have its own.
type Counter struct {
	rwmu     sync.RWMutex
	capacity uint8
	items    map[string]*itemNode
	freqHead *freqNode
}

func newCounter(capacity uint8) *Counter {
	return &Counter{
		capacity: capacity,
		items:    make(map[string]*itemNode),
	}
}

// Hit hits the specified key.
func (c *Counter) Hit(key string) {
	c.rwmu.Lock()
	item, ok := c.items[key]
	if ok {
		// update the item's freq
		c.increment(item)
		c.rwmu.Unlock()
		return
	}

	// record
	if uint8(len(c.items)) > c.capacity {
		// evict the least frequently used item
		c.evict()
	}
	item = &itemNode{key: key}
	c.add(item)
	c.rwmu.Unlock()
}

// Latch returns the cached key hit counts and reset it.
func (c *Counter) Latch() map[string]uint64 {
	c.rwmu.RLock()
	res := make(map[string]uint64, len(c.items))
	for key, item := range c.items {
		res[key] = item.freqNode.freq
	}
	c.reset()
	c.rwmu.RUnlock()
	return res
}

func (c *Counter) reset() {
	c.items = nil
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

	item.FreeMySelf()
	targetFreqNode.AppendItem(item)

	if curFreqNode.itemHead != nil {
		return
	}

	// remove current freq node if empty
	if c.freqHead == curFreqNode {
		c.freqHead = targetFreqNode
	}
	curFreqNode.RemoveMySelf()
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
	fnode.RemoveMySelf()
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
	n.itemHead = item
	return item
}

func (n *freqNode) AppendItem(item *itemNode) {
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

func (n *freqNode) RemoveMySelf() {
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

func (n *itemNode) FreeMySelf() {
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
