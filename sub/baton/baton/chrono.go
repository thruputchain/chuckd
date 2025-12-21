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
	lk    sync.Mutex
	block [2]blockChrono
	//
	lastNominateStart time.Time
	lastNominateEnd   time.Time
	lastCommitStart   time.Time
	lastCommitEnd     time.Time
	//
	lastNominateStartHeight proto.BlockHeight
	lastNominateEndHeight   proto.BlockHeight
	lastCommitStartHeight   proto.BlockHeight
	lastCommitEndHeight     proto.BlockHeight
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
	b.chrono.lastNominateStartHeight = height

	if !b.chrono.lastCommitStart.IsZero() {
		liblog.Debugf("commit_start_to_nominate_start_latency=%v | commit_height=%d, nominate_height=%d",
			b.chrono.lastNominateStart.Sub(b.chrono.lastCommitStart),
			b.chrono.lastCommitStartHeight, b.chrono.lastNominateStartHeight,
		)
	}
}

func (b *Baton) stampNominateEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).nominateEnd = now
	b.chrono.lastNominateEnd = now
	b.chrono.lastNominateEndHeight = height
}

func (b *Baton) stampCommitStart(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).commitStart = now
	b.chrono.lastCommitStart = now
	b.chrono.lastCommitStartHeight = height

	if !b.chrono.lastNominateStart.IsZero() {
		liblog.Debugf("nominate_start_to_commit_start_latency=%v | nominate_height=%d, commit_height=%d",
			b.chrono.lastCommitStart.Sub(b.chrono.lastNominateStart),
			b.chrono.lastNominateStartHeight, b.chrono.lastCommitStartHeight,
		)
	}
}

func (b *Baton) stampCommitEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).commitEnd = now
	b.chrono.lastCommitEnd = now
	b.chrono.lastCommitEndHeight = height
}
