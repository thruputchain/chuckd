// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/shard"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: how do we cleanup stale cards? blocked on xdr defs

type CoordinatorService struct {
	upid proto.UPID
	//
	lk   sync.Mutex
	db   map[proto.UPID]proto.ShardCard
	exec map[proto.UPID]proto.ExecCard
	//
	dbCond       *sync.Cond
	dbLk         sync.Mutex
	dbShards     xdrutil.ShardCards
	dbIsUp       bool
	dbIsRestored bool
}

const UPID_SEED_OFFSET = 0x1eef2eef3eef4eef

func NewCoordinatorService(ctx context.Context) *CoordinatorService {
	log.Infof("starting coordinator service")
	upid, err := GenerateUPID(runtime.NewRand(UPID_SEED_OFFSET).Uint64())
	must.NoError(ctx, err)
	cs := &CoordinatorService{
		upid: upid,
		db:   map[proto.UPID]proto.ShardCard{},
		exec: map[proto.UPID]proto.ExecCard{},
	}
	cs.dbCond = sync.NewCond(&cs.dbLk)
	return cs
}

func (s *CoordinatorService) Card() proto.CoordinatorCard {
	return proto.CoordinatorCard{Upid: s.upid}
}

func (s *CoordinatorService) Coordinator_null() {}

func (s *CoordinatorService) Coordinator_intro_shard(card proto.ShardCard) proto.IntroShardResult {
	s.lk.Lock()
	s.db[card.Upid] = card
	shards := s.findCompleteDb()
	s.lk.Unlock()

	s.prepareDb(shards)

	return proto.IntroShardResult{Cc: s.Card()}
}

func (s *CoordinatorService) prepareDb(shards xdrutil.ShardCards) {
	if len(shards) == 0 {
		return
	}

	s.dbLk.Lock()
	defer s.dbLk.Unlock()

	if s.dbIsUp {
		return
	}
	s.dbIsUp = true
	s.dbShards = shards

	// trigger db restoration
	go func() {

		// restore shards in parallel
		var wg sync.WaitGroup
		wg.Add(len(shards))
		for _, s := range shards {
			go func(s proto.ShardCard) {
				defer wg.Done()

				c := shard.NewShardClient(context.TODO(), s.Address)
				c.Shard_restore(s.Block_height)
			}(s)
		}
		wg.Wait()

		// update db state
		s.dbLk.Lock()
		defer s.dbLk.Unlock()
		s.dbIsRestored = true
		s.dbCond.Broadcast()
	}()
}

func (s *CoordinatorService) Coordinator_intro_exec(card proto.ExecCard) proto.IntroExecResult {
	s.lk.Lock()
	s.exec[card.Upid] = card
	s.lk.Unlock()
	shards := s.waitForDbReady()
	return proto.IntroExecResult{
		Cc:     s.Card(),
		Shards: shards,
	}
}

func (s *CoordinatorService) waitForDbReady() xdrutil.ShardCards {
	s.dbLk.Lock()
	defer s.dbLk.Unlock()
	for !s.dbIsReady() {
		s.dbCond.Wait()
	}
	return s.dbShards
}

func (s *CoordinatorService) dbIsReady() bool {
	return s.dbIsUp && s.dbIsRestored
}

// findCompleteDb returns a list of db shards of highest block height that form a complete cluster.
// It returns nil if there is no complete db cluster.
// A set of shards is deemed complete if:
// - it is not empty
// - all shards report the same block height and total shard count
// - the shard count matches the reported count
func (s *CoordinatorService) findCompleteDb() xdrutil.ShardCards {
	byHeight := map[uint32]xdrutil.ShardCards{} // block_height -> shards
	for _, dbc := range s.db {
		byHeight[dbc.Block_height] = append(byHeight[dbc.Block_height], dbc)
	}
	var foundReady bool
	var foundHeight uint32
	var foundShards xdrutil.ShardCards
	for height, shards := range byHeight {
		shards.Sort()
		if isShardCountComplete(shards) {
			if !foundReady || foundHeight < height {
				foundReady = true
				foundHeight = height
				foundShards = shards
			}
		}
	}
	return foundShards
}

func isShardCountComplete(shards xdrutil.ShardCards) bool {
	if len(shards) == 0 {
		return false
	}
	shardCount := int64(-1)
	for _, dbc := range shards {
		if shardCount < 0 {
			shardCount = int64(dbc.Shard_count)
		}
		if shardCount != int64(dbc.Shard_count) {
			return false
		}
	}
	if shardCount != int64(len(shards)) {
		return false
	}
	return true
}
