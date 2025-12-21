package baton

import (
	"sync"
	"time"

	"github.com/groundhogchain/lib-go/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

type blockChrono struct {
	height        proto.BlockHeight
	nominateStart time.Time
	nominateEnd   time.Time
	commitStart   time.Time
	commitEnd     time.Time
}

type chrono struct {
	lk                sync.Mutex
	block             [2]blockChrono
	lastNominateStart time.Time
	lastNominateEnd   time.Time
	lastCommitStart   time.Time
	lastCommitEnd     time.Time
}

func (x *chrono) at(height proto.BlockHeight) *blockChrono {
	k := &x.block[int(height)%len(x.block)]
	if k.height != height {
		*k = blockChrono{height: height}
	}
	return k
}

func (b *Baton) stampNominateStart(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).nominateStart = now
	b.chrono.lastNominateStart = now

	if !b.chrono.lastCommitStart.IsZero() {
		liblog.Debugf("commit_start_to_nominate_start_latency=%v",
			b.chrono.lastNominateStart.Sub(b.chrono.lastCommitStart))
	}
}

func (b *Baton) stampNominateEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).nominateEnd = now
	b.chrono.lastNominateEnd = now
}

func (b *Baton) stampCommitStart(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).commitStart = now
	b.chrono.lastCommitStart = now

	if !b.chrono.lastNominateStart.IsZero() {
		liblog.Debugf("nominate_start_to_commit_start_latency=%v",
			b.chrono.lastCommitStart.Sub(b.chrono.lastNominateStart))
	}
}

func (b *Baton) stampCommitEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).commitEnd = now
	b.chrono.lastCommitEnd = now
}

// nominate_start -> commit_start
// commit_start -> nominate_start
