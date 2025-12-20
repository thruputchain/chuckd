package baton

import (
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type blockChrono struct {
	height          proto.BlockHeight
	nominationStart time.Time
	nominationEnd   time.Time
	commitStart     time.Time
	commitEnd       time.Time
}

type chrono struct {
	lk    sync.Mutex
	block [2]blockChrono
}

func (x *chrono) at(height proto.BlockHeight) *blockChrono {
	k := &x.block[int(height)%len(x.block)]
	if k.height != height {
		*k = blockChrono{height: height}
	}
	return k
}

func (b *Baton) stampNominationStart(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).nominationStart = b.handler.Now()
}

func (b *Baton) stampNominationEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).nominationEnd = b.handler.Now()
}

func (b *Baton) stampCommitStart(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).commitStart = b.handler.Now()
}

func (b *Baton) stampCommitEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).commitEnd = b.handler.Now()
}
