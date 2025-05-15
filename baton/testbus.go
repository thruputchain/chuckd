package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
)

type TestBus struct {
	emitted chan TestEnvelope
}

type TestEnvelope struct {
	Stamp time.Time
	Event any
}

func NewTestBus() *TestBus {
	return &TestBus{
		emitted: make(chan TestEnvelope),
	}
}

func (b *TestBus) SendEvent(s time.Time, event any) {
	env := TestEnvelope{Stamp: s, Event: event}
	log.Debugf("%s", event)
	b.emitted <- env
}

func (b *TestBus) ReceiveEvent(ctx context.Context) TestEnvelope {
	return <-b.emitted
}
