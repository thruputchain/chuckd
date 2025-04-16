// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/shard"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: how do we cleanup stale cards? blocked on xdr defs

type CoordinatorService struct {
	dialer libnet.AddressDialer
	//
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

func NewCoordinatorService(ctx context.Context, dialer libnet.AddressDialer, rr *rand.Rand) *CoordinatorService {
	log.Infof("starting coordinator service")
	upid, err := GenerateUPID(rr.Uint64())
	must.NoError(ctx, err)
	cs := &CoordinatorService{
		dialer: dialer,
		upid:   upid,
		db:     map[proto.UPID]proto.ShardCard{},
		exec:   map[proto.UPID]proto.ExecCard{},
	}
	cs.dbCond = sync.NewCond(&cs.dbLk)
	return cs
}

func (svc *CoordinatorService) Card() proto.CoordinatorCard {
	return proto.CoordinatorCard{Upid: svc.upid}
}

func (svc *CoordinatorService) Coordinator_null() {}

func (svc *CoordinatorService) Coordinator_intro_shard(card proto.ShardCard) proto.IntroShardResult {
	svc.lk.Lock()
	svc.db[card.Upid] = card
	shards := svc.findCompleteDb()
	svc.lk.Unlock()

	svc.prepareDb(shards)

	return proto.IntroShardResult{Cc: svc.Card()}
}

func (svc *CoordinatorService) prepareDb(shards xdrutil.ShardCards) {
	if len(shards) == 0 {
		return
	}

	svc.dbLk.Lock()
	defer svc.dbLk.Unlock()

	if svc.dbIsUp {
		return
	}
	svc.dbIsUp = true
	svc.dbShards = shards

	// trigger db restoration
	go func() {

		// restore shards in parallel
		var wg sync.WaitGroup
		wg.Add(len(shards))
		for _, sh := range shards {
			go func(s proto.ShardCard) {
				defer wg.Done()

				c, err := shard.DialShardClient(context.TODO(), svc.dialer, s.Address)
				must.NoError(context.TODO(), err)
				c.Shard_restore(s.Block_height) //XXX: errors
			}(sh)
		}
		wg.Wait()

		// update db state
		svc.dbLk.Lock()
		defer svc.dbLk.Unlock()
		svc.dbIsRestored = true
		svc.dbCond.Broadcast()
	}()
}

func (svc *CoordinatorService) Coordinator_intro_exec(card proto.ExecCard) proto.IntroExecResult {
	svc.lk.Lock()
	svc.exec[card.Upid] = card
	svc.lk.Unlock()
	shards := svc.waitForDbReady()
	return proto.IntroExecResult{
		Cc:     svc.Card(),
		Shards: shards,
	}
}

func (svc *CoordinatorService) waitForDbReady() xdrutil.ShardCards {
	svc.dbLk.Lock()
	defer svc.dbLk.Unlock()
	for !svc.dbIsReady() {
		svc.dbCond.Wait()
	}
	return svc.dbShards
}

func (svc *CoordinatorService) dbIsReady() bool {
	return svc.dbIsUp && svc.dbIsRestored
}

// findCompleteDb returns a list of db shards of highest block height that form a complete cluster.
// It returns nil if there is no complete db cluster.
// A set of shards is deemed complete if:
// - it is not empty
// - all shards report the same block height and total shard count
// - the shard count matches the reported count
func (svc *CoordinatorService) findCompleteDb() xdrutil.ShardCards {
	byHeight := map[uint32]xdrutil.ShardCards{} // block_height -> shards
	for _, dbc := range svc.db {
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
