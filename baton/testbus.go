package baton

import (
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/log"
)

type TestBus struct {
	lk      sync.Mutex
	emitted []TestEnvelope
}

type TestEnvelope struct {
	Stamp time.Time
	Event any
}

func NewTestBus() *TestBus {
	return &TestBus{}
}

func (b *TestBus) Add(s time.Time, event any) {
	env := TestEnvelope{Stamp: s, Event: event}
	log.Infof("envelope: %s", form.SprintJSON(env))
	b.lk.Lock()
	defer b.lk.Unlock()
	b.emitted = append(b.emitted, env)
}
