// Package coord implements the Coordinator API which helps coordinate db and executors internally.
package coord

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: how do we cleanup stale cards? blocked on xdr defs

type CoordinatorService struct {
	net libnet.Network
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

func NewCoordinatorService(ctx context.Context, n libnet.Network, rr *rand.Rand) *CoordinatorService {
	log.Infof("starting coordinator service")
	upid, err := GenerateUPID(rr.Uint64())
	must.NoError(ctx, err)
	cs := &CoordinatorService{
		net:  n,
		upid: upid,
		db:   map[proto.UPID]proto.ShardCard{},
		exec: map[proto.UPID]proto.ExecCard{},
	}
	cs.dbCond = sync.NewCond(&cs.dbLk)
	return cs
}

func (svc *CoordinatorService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.Coordinator1_Server{Srv: svc})
}

func (svc *CoordinatorService) Card() proto.CoordinatorCard {
	return proto.CoordinatorCard{Upid: svc.upid}
}
