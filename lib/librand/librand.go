package librand

import (
	"math/rand"
	"sync"
)

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(rand.Int63()).(rand.Source64))
}

func NewRandFrom(parent *rand.Rand) *rand.Rand {
	return rand.New(rand.NewSource(parent.Int63()))
}

func NewSyncRand() *SyncRand {
	return &SyncRand{rand: NewRand()}
}

func NewSyncRandWrapping(r *rand.Rand) *SyncRand {
	return &SyncRand{rand: r}
}

type SyncRand struct {
	lk   sync.Mutex
	rand *rand.Rand
}

func (x *SyncRand) Int63() int64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.rand.Int63()
}
