package hotkey

type Counter struct {
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

func (c *Counter) Hit(key string) {
	iNode, ok := c.items[key]
	if ok {
		// update the item's freq
		c.increment(iNode)
		return
	}

	// record the item
	if uint8(len(c.items)) > c.capacity {
		// evict the least frequently used item
		c.evictItem()
	}
	c.addItem(key)
}

func (c *Counter) increment(iNode *itemNode) {
	fNode := iNode.freqNode
	curFreq := fNode.freq

	var targetFreqNode *freqNode
	if fNode.next == nil || fNode.next.freq != curFreq+1 {
		targetFreqNode = &freqNode{freq: curFreq + 1}
		fNode.InsertAfterMe(targetFreqNode)
	} else {
		targetFreqNode = fNode.next
	}

	iNode.FreeMySelf()
	targetFreqNode.AppendItem(iNode)

	if fNode.iHead != nil {
		return
	}

	if c.freqHead == fNode {
		c.freqHead = targetFreqNode
	}
	fNode.RemoveMySelf()
}

func (c *Counter) addItem(key string) {
	iNode := &itemNode{
		key: key,
	}
	c.items[key] = iNode

	if c.freqHead != nil && c.freqHead.freq == 1 {
		c.freqHead.AppendItem(iNode)
		return
	}

	fNode := &freqNode{
		freq: 1,
	}
	fNode.AppendItem(iNode)

	if c.freqHead != nil {
		c.freqHead.InsertBeforeMe(fNode)
	}
	c.freqHead = fNode
}

func (c *Counter) evictItem() {
	fNode := c.freqHead
	iNode := fNode.iHead
	delete(c.items, iNode.key)
	fNode.PopItem()

	if fNode.iHead != nil {
		return
	}

	c.freqHead = fNode.next
	fNode.RemoveMySelf()
}

type freqNode struct {
	freq         uint32
	prev, next   *freqNode
	iHead, iTail *itemNode
}

func (n *freqNode) PopItem() *itemNode {
	if n.iHead == nil {
		return nil
	}

	// only have one item
	if n.iHead == n.iTail {
		iNode := n.iHead
		n.iHead = nil
		n.iTail = nil
		return iNode
	}

	iNode := n.iHead
	iNode.next.prev = nil
	n.iHead = iNode
	return iNode
}

func (n *freqNode) AppendItem(iNode *itemNode) {
	if n.iHead == nil {
		n.iHead = iNode
		n.iTail = iNode
		return
	}

	iNode.prev = n.iTail
	iNode.next = nil
	n.iTail.next = iNode
	n.iTail = iNode
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
	n.iHead, n.iTail = nil, nil
}

type itemNode struct {
	key        string
	freqNode   *freqNode
	prev, next *itemNode
}

func (n *itemNode) FreeMySelf() {
	fNode := n.freqNode
	if fNode.iHead == fNode.iTail {
		fNode.iHead, fNode.iTail = nil, nil
	} else if fNode.iHead == n {
		n.next.prev = nil
		fNode.iHead = n.next
	} else if fNode.iTail == n {
		n.prev.next = nil
		fNode.iTail = n.prev
	} else {
		n.prev.next = n.next
		n.next.prev = n.prev
	}

	n.prev = nil
	n.next = nil
	n.freqNode = nil
}
