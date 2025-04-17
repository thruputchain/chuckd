package broadcast

import (
	"sync"
	"time"
)

type TreeCache struct {
	ttl   time.Duration
	lk    sync.Mutex
	trees map[MulticastTreeHash]*treeCacheEntry // cached multicast trees
}

type treeCacheEntry struct {
	lastUse   time.Time
	numRefs   int
	treeNodes MulticastTreeNodes
}

func NewTreeCache(ttl time.Duration) *TreeCache {
	x := &TreeCache{
		ttl:   ttl,
		trees: map[MulticastTreeHash]*treeCacheEntry{},
	}
	go x.collect()
	return x
}

func (x *TreeCache) collect() {
	for {
		time.Sleep(x.ttl)
		now := time.Now()
		x.lk.Lock()
		for h, e := range x.trees {
			if e.numRefs == 0 && now.Sub(e.lastUse) > x.ttl {
				delete(x.trees, h)
			}
		}
		x.lk.Unlock()
	}
}

func (x *TreeCache) Find(h MulticastTreeHash) (MulticastTreeNodes, bool) {
	x.lk.Lock()
	defer x.lk.Unlock()

	entry := x.trees[h]
	if entry == nil {
		return nil, false
	}
	entry.lastUse = time.Now()
	return entry.treeNodes, true
}

func (x *TreeCache) Ref(h MulticastTreeHash, nodes MulticastTreeNodes) {
	x.lk.Lock()
	defer x.lk.Unlock()

	entry := x.trees[h]
	if entry == nil {
		entry = &treeCacheEntry{
			lastUse:   time.Now(),
			numRefs:   1,
			treeNodes: nodes,
		}
		x.trees[h] = entry
	} else {
		entry.lastUse = time.Now()
		entry.numRefs += 1
	}
}

func (x *TreeCache) Deref(h MulticastTreeHash) {
	x.lk.Lock()
	defer x.lk.Unlock()

	entry := x.trees[h]
	if entry == nil || entry.numRefs <= 0 {
		panic("bug")
	}
	entry.lastUse = time.Now()
	entry.numRefs -= 1
}
