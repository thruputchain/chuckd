package coord

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (svc *CoordinatorService) Coordinator_intro_exec(ctx context.Context, card proto.ExecCard) (proto.IntroExecResult, error) {
	svc.lk.Lock()
	svc.exec[card.Upid] = card
	svc.lk.Unlock()
	shards := svc.WaitDbReady()
	return proto.IntroExecResult{
		Cc:     svc.Card(),
		Shards: shards,
	}, nil
}
