package baton

import (
	"time"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	proto "github.com/groundhogchain/protocol-go"
)

type liveness struct {
	started             time.Time
	lastCommit          time.Time // updated by commit
	lastExternalization time.Time // updated by scp externalize
	lastBackfeed        time.Time // updated by drive loop
	lastCatchupAttempt  time.Time // updated by attemptCatchup
}

func (b *Baton) sinceLastExternalization(now time.Time) time.Duration {
	return now.Sub(b.liveness.lastExternalization)
}

func (b *Baton) sinceLastBackfeed(now time.Time) time.Duration {
	return now.Sub(b.liveness.lastBackfeed)
}

func (b *Baton) noteStarted() {
	b.liveness.started = b.handler.Now()
}

func (b *Baton) noteExternalized(_ scpkit.Slot) {
	b.liveness.lastExternalization = b.handler.Now()
}

func (b *Baton) noteCommitted(_ proto.BlockHeight) {
	b.liveness.lastCommit = b.handler.Now()
}

func (b *Baton) noteBackfeed() {
	b.liveness.lastBackfeed = b.handler.Now()
}

func (b *Baton) noteCatchupAttempt() {
	b.liveness.lastCatchupAttempt = b.handler.Now()
}

func (b *Baton) sinceLastCatchupAttempt() time.Duration {
	return b.handler.Now().Sub(b.liveness.lastCatchupAttempt)
}
