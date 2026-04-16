package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/lib-go/liblog"
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

func (b *TestBus) SendEvent(ctx context.Context, s time.Time, event any) {
	env := TestEnvelope{Stamp: s, Event: event}
	liblog.Debugf(ctx, "%s", event)
	b.emitted <- env
}

func (b *TestBus) ReceiveEvent(ctx context.Context) TestEnvelope {
	return <-b.emitted
}
