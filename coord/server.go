// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/xdr/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

// XXX: how do we cleanup stale cards? blocked on xdr defs

type CoordinatorService struct {
	upid proto.UPID
	lk   sync.Mutex
	db   map[proto.UPID]proto.DbCard
	exec map[proto.UPID]proto.ExecutorCard
}

const UPID_SEED_OFFSET = 0x1eef2eef3eef4eef

func NewCoordinatorService(ctx context.Context) *CoordinatorService {
	upid, err := GenerateUPID(runtime.NewRand(UPID_SEED_OFFSET).Uint64())
	must.NoError(ctx, err)
	return &CoordinatorService{
		upid: upid,
		db:   map[proto.UPID]proto.DbCard{},
		exec: map[proto.UPID]proto.ExecutorCard{},
	}
}

func (s *CoordinatorService) Card() proto.CoordinatorCard {
	return proto.CoordinatorCard{Upid: s.upid}
}

func (s *CoordinatorService) Null() {}

func (s *CoordinatorService) Intro_db(card proto.DbCard) proto.IntroDbResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.db[card.Upid] = card
	return proto.IntroDbResult{Cc: s.Card()}
}

func (s *CoordinatorService) Intro_executor(card proto.ExecutorCard) proto.IntroExecutorResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.exec[card.Upid] = card
	return proto.IntroExecutorResult{Cc: s.Card()}
}

func (s *CoordinatorService) List_db() proto.ListDbResult {
	s.lk.Lock()
	defer s.lk.Unlock()
	shards := s.findReadyDb()
	return proto.ListDbResult{
		Cc:       s.Card(),
		Is_ready: len(shards) > 0,
		Shards:   shards,
	}
}

// findReadyDb returns a list of db shards of highest block height that form a ready cluster.
// It returns nil if there is no ready db cluster.
// A set of shards is deemed ready if:
// - it is not empty
// - all shards report the same block height and total shard count
// - the shard count matches the reported count
func (s *CoordinatorService) findReadyDb() xdrutil.DbCards {
	byHeight := map[uint32]xdrutil.DbCards{} // block_height -> shards
	for _, dbc := range s.db {
		byHeight[dbc.Block_height] = append(byHeight[dbc.Block_height], dbc)
	}
	var foundReady bool
	var foundHeight uint32
	var foundShards xdrutil.DbCards
	for height, shards := range byHeight {
		shards.Sort()
		if isDbReadyFixedHeight(shards) {
			if !foundReady || foundHeight < height {
				foundReady = true
				foundHeight = height
				foundShards = shards
			}
		}
	}
	return foundShards
}

func isDbReadyFixedHeight(shards xdrutil.DbCards) bool {
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

func StartServer(ctx context.Context, addr string) {
	go DoServer(ctx, addr)
}

// DoServer runs the coordinator server and blocks forever.
// TODO: make an abortable server for test purposes
func DoServer(ctx context.Context, addr string) {
	s := NewCoordinatorService(ctx)

	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	defer l.Close()

	for {
		sock, err := l.Accept()
		must.NoError(ctx, err)
		driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
		driver.Lock = nil // process all calls in parallel
		driver.Log = log.Writer()
		driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
		driver.Register(proto.Coordinator1_Server{Srv: s})
		go driver.Go()
	}
}
