// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/xdr/xdrdef"
	"github.com/groundhogchain/chuckd/xdr/xdrutil"
	"github.com/xdrpp/goxdr/rpc"
)

// XXX: how do we cleanup stale cards? blocked on xdr defs

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
		Cc:       s.Card(),
		Is_ready: s.isDbReady(),
		Shards:   dbs,
	}
}

// isDbReady returns true if the db cluster is ready to use.
// The cluster is deemed ready if:
// - one or more shards have introduced themselves
// - all introduced shards report the same total shard count and block height
// - the total shard count matches the number of shards we know
func (s *Coordinator) isDbReady() bool {
	if len(s.db) == 0 {
		return false
	}
	shardCount, blockHeight := int64(-1), int64(-1)
	for _, dbc := range s.db {
		if shardCount < 0 {
			shardCount = int64(dbc.Shard_count)
		}
		if shardCount != int64(dbc.Shard_count) {
			return false
		}
		if blockHeight < 0 {
			blockHeight = int64(dbc.Block_height)
		}
		if blockHeight != int64(dbc.Block_height) {
			return false
		}
	}
	if shardCount != int64(len(s.db)) {
		return false
	}
	return true
}

// DoServer runs the coordinator server and blocks forever.
func DoServer(ctx context.Context, addr string) {
	s := NewCoordinator(ctx)

	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	defer l.Close()

	for {
		sock, err := l.Accept()
		must.NoError(ctx, err)
		driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
		driver.Lock = nil // process all calls in parallel
		driver.Log = log.Writer()
		driver.Register(xdrdef.Coordinator1_Server{Srv: s})
		go driver.Go() // XXX: driver.Go handles panics, but doesn't display the attached error/trace info
	}
}
