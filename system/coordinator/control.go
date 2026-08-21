package coordinator

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
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

func (x Control) LoadGenesisConfigurationV2(ctx context.Context, gb proto.GenesisConfigurationV2) (proto.DbSchedResult, error) {
	return x.conn.Schedcoord_load_genesis_configuration_v2(ctx, gb)
}

func (x Control) CommitGenesisCeremony(ctx context.Context, request proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	return x.conn.Schedulercoord_commit_genesis_ceremony(ctx, request)
}

func (x Control) GetTip(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	return x.conn.Schedcoord_get_tip(ctx)
}

func (x Control) LookupTransactions(ctx context.Context, hashes proto.TransactionHashBatch) (proto.OptSignedTransactionBatch, error) {
	return x.conn.Schedcoord_lookup_transactions(ctx, hashes)
}

func (x Control) SubmitTxs(ctx context.Context, txs []*proto.SignedTransaction) {
	x.system.UnvettedSubmitter.SubmitTxs(ctx, txs...)
}

func (x Control) SubmitParallelTxs(ctx context.Context, txs proto.ParallelSignedTransactionBatch) {
	x.system.UnvettedSubmitter.SubmitParallelTxs(ctx, txs)
}

func (x Control) SignalGenesisIsReady() {
	x.system.signalGenesisIsReady()
}

func (x Control) SetLevers(levers batonkit.BatonLevers) {
	x.system.SetLevers(levers)
}

func (x Control) GetLevers() batonkit.BatonLevers {
	return x.system.GetLevers()
}

func (x Control) SetHooks(hooks batonkit.BatonHooks) {
	x.system.SetHooks(hooks)
}

func (x Control) PauseNominating() {
	x.system.PauseNominating()
}

func (x Control) ResumeNominating() {
	x.system.ResumeNominating()
}
