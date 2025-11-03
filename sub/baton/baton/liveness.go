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
}

func (b *Baton) sinceLastExternalization() *time.Duration {
	if b.liveness.lastExternalization.IsZero() {
		return nil
	}
	elapsed := b.handler.Now().Sub(b.liveness.lastExternalization)
	return &elapsed
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
