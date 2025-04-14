package broadcast

import (
	"sync"
	"time"
)

type TreeCache struct {
	lk    sync.Mutex
	trees map[MulticastTreeHash]*treeCacheEntry // cached multicast trees
}

type treeCacheEntry struct {
	lastUse   time.Time
	numRefs   int
	treeNodes MulticastTreeNodes
}

func NewTreeCache() *TreeCache {
	x := &TreeCache{
		trees: map[MulticastTreeHash]*treeCacheEntry{},
	}
	go x.collect()
	return x
}

const TreeCacheCollectionInterval = time.Minute

func (x *TreeCache) collect() {
	for {
		time.Sleep(TreeCacheCollectionInterval)
		now := time.Now()
		x.lk.Lock()
		for h, e := range x.trees {
			if e.numRefs == 0 && now.Sub(e.lastUse) > TreeCacheCollectionInterval {
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
