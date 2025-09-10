package coord

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *CoordinatorService) DialDb(ctx context.Context) (batonapi.DbCoordConn, error) {
	shard := x.pickShard()
	liblog.Infof("dialing shard %v", form.Display(shard.Address))
	return DialDbCoordClient(ctx, x.net, shard.Address)
}

func (x *CoordinatorService) pickShard() proto.ShardCard {
	x.dbLk.Lock()
	defer x.dbLk.Unlock()
	return x.dbShards[rand.Intn(len(x.dbShards))]
}

func (x *CoordinatorService) DialExecs(ctx context.Context) ([]batonapi.ExecCoordConn, error) {
	conns := []batonapi.ExecCoordConn{}
	for _, card := range x.getExecs() {
		conn, err := DialExecutorCoordClient(ctx, x.net, card.Address)
		if err != nil {
			liblog.Errorf("dialing executor: %v", err)
			continue
		}
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("no executors")
	}
	return conns, nil
}

func (x *CoordinatorService) getExecs() []proto.ExecCard {
	x.lk.Lock()
	defer x.lk.Unlock()
	r := []proto.ExecCard{}
	for _, card := range x.exec {
		r = append(r, card)
	}
	return r
}
