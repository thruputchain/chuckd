package baton

import (
	"time"

	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type liveness struct {
	started             time.Time
	lastCommit          time.Time // updated by commit
	lastExternalization time.Time // updated by scp externalize
	lastBackfeed        time.Time // updated by drive loop
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
