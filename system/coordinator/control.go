package coordinator

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/sched"
	proto "github.com/groundhogchain/protocol-go"
)

type Control struct {
	s            *System
	conn         sched.SchedulerCoordConn
	genesisReady chan struct{}
}

func (x Control) LoadGenesisBlock(ctx context.Context, gb proto.GenesisBlock) (proto.DbSchedResult, error) {
	return x.conn.Schedcoord_load_genesis_block(ctx, gb)
}

func (x Control) SubmitTxs(ctx context.Context, txs []*proto.SignedTransaction) {
	x.s.WalletSubmitter.SubmitTxs(ctx, txs...)
}

func (x Control) GenesisIsReady() {
	close(x.genesisReady)
}
