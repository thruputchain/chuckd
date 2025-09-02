package coord

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (svc *CoordinatorService) Coordinator_null(ctx context.Context) error {
	return nil
}

func (svc *CoordinatorService) Coordinator_intro_shard(ctx context.Context, card proto.ShardCard) (proto.IntroShardResult, error) {
	svc.lk.Lock()
	svc.db[card.Upid] = card
	shards, height, ready := svc.findCompleteDbMinShardHeight()
	svc.lk.Unlock()

	if ready {
		go svc.prepareDb(shards, height)
	}

	return proto.IntroShardResult{Cc: svc.Card()}, nil
}

func (svc *CoordinatorService) prepareDb(shards xdrutil.ShardCards, height proto.BlockHeight) {
	ctx := svc.ctx
	must.Assertf(ctx, len(shards) > 0, "no shards")

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

				c, err := DialDbCoordClient(ctx, svc.net, s.Address)
				must.NoError(ctx, err)
				defer c.Close()
				_, err = c.Dbcoord_shard_restore(ctx, height)
				must.NoError(ctx, err)
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
