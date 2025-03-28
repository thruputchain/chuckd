package runtime

import (
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
)

var (
	lk   sync.Mutex
	seed int64
)

func SetSeed(s int64) {
	lk.Lock()
	defer lk.Unlock()
	seed = s
	log.Infof("randomness seed %x", seed)
}

func NewRand(offset int64) *rand.Rand {
	lk.Lock()
	defer lk.Unlock()
	return rand.New(rand.NewSource(seed ^ offset))
}
