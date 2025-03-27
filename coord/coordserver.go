// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/xdr/xdrdef"
	"github.com/groundhogchain/chuckd/xdr/xdrutil"
)

// XXX: how do we cleanup stale cards?
// XXX: determine db cluster is ready?

type Coordinator struct {
	upid xdrdef.UPID
	lk   sync.Mutex
	db   map[xdrdef.UPID]xdrdef.DbCard
	exec map[xdrdef.UPID]xdrdef.ExecutorCard
}

const UPID_SEED_OFFSET = 0x1eef2eef3eef4eef

func NewCoordinator(ctx context.Context) *Coordinator {
	upid, err := GenerateUPID(runtime.NewRand(UPID_SEED_OFFSET).Uint64())
	must.NoError(ctx, err)
	return &Coordinator{
		upid: upid,
		db:   map[xdrdef.UPID]xdrdef.DbCard{},
		exec: map[xdrdef.UPID]xdrdef.ExecutorCard{},
	}
}

func (s *Coordinator) Card() xdrdef.CoordinatorCard {
	return xdrdef.CoordinatorCard{Upid: s.upid}
}

func (s *Coordinator) Null() {}

func (s *Coordinator) Intro_db(card xdrdef.DbCard) xdrdef.IntroDbResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.db[card.Upid] = card
	return xdrdef.IntroDbResult{Cc: s.Card()}
}

func (s *Coordinator) Intro_executor(card xdrdef.ExecutorCard) xdrdef.IntroExecutorResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.exec[card.Upid] = card
	return xdrdef.IntroExecutorResult{Cc: s.Card()}
}

func (s *Coordinator) List_db() xdrdef.ListDbResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	dbs := xdrutil.DbCards{}
	for _, db := range s.db {
		dbs = append(dbs, db)
	}
	dbs.Sort()
	return xdrdef.ListDbResult{
		Cc:        s.Card(),
		Readiness: xdrdef.READY, //XXX: when are we ready?
		Shards:    dbs,
	}
}
