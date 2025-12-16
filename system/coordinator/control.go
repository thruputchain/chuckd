package coordinator

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/sched"
	proto "github.com/groundhogchain/protocol-go"
)

type Control struct {
	system       *System
	conn         sched.SchedulerCoordConn
	genesisReady chan struct{}
}

func (x Control) LoadGenesisConfiguration(ctx context.Context, gb proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	return x.conn.Schedcoord_load_genesis_configuration(ctx, gb)
}

func (x Control) SubmitTxs(ctx context.Context, txs []*proto.SignedTransaction) {
	x.system.UnvettedSubmitter.SubmitTxs(ctx, txs...)
}

func (x Control) GenesisIsReady() {
	close(x.genesisReady)
}

func (x Control) PauseNominating() {
	x.system.PauseNominating()
}

func (x Control) ResumeNominating() {
	x.system.ResumeNominating()
}
