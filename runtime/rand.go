package runtime

import (
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
)

var (
	lk       sync.Mutex
	rootSeed int64
	rootRand *rand.Rand
)

func SetRootSeed(s int64) {
	lk.Lock()
	defer lk.Unlock()
	if rootRand != nil {
		panic("setting root randomness seed twice")
	}
	rootSeed = s
	rootRand = rand.New(rand.NewSource(rootSeed))
	log.Infof("root randomness seed %x", rootSeed)
}

func GetRootSeed() int64 {
	lk.Lock()
	defer lk.Unlock()
	return rootSeed
}

func RootRandInt63() int64 {
	lk.Lock()
	defer lk.Unlock()
	return rootRand.Int63()
}

// sources

func NewSourceFromRoot() rand.Source64 {
	return rand.NewSource(RootRandInt63()).(rand.Source64)
}

// rands

func NewRandFromRoot() *rand.Rand {
	return rand.New(NewSourceFromRoot())
}

func NewRandFromRand(parent *rand.Rand) *rand.Rand {
	return rand.New(rand.NewSource(parent.Int63()))
}
