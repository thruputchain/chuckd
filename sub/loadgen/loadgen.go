package loadgen

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type Config struct {
	NumAccts       int
	NumTxsPerBlock int
	NumBlocks      int
}

type LoadGen struct {
	Cfg Config
	//
	rand   *rand.Rand
	inside libnet.Network
	//
	genesisCh chan *proto.GenesisConfiguration
	initTxsCh chan []*proto.SignedTransaction
	workTxsCh chan proto.ParallelSignedTransactionBatch
	done      chan struct{}
}

func NewLoadGen(
	ctx context.Context,
	rand *rand.Rand,
	inside libnet.Network,
	cfg Config,
) *LoadGen {
	x := &LoadGen{
		Cfg:       cfg,
		rand:      rand,
		inside:    inside,
		genesisCh: make(chan *proto.GenesisConfiguration, 1),
		initTxsCh: make(chan []*proto.SignedTransaction),
		workTxsCh: make(chan proto.ParallelSignedTransactionBatch),
		done:      make(chan struct{}),
	}
	return x
}

func (x *LoadGen) Start(ctx context.Context, loadgenAddr string) {

	lgConn, err := DialLoadgenClientRetry(ctx, x.inside, loadgenAddr)
	must.NoError(ctx, err)

	liblog.Infof(ctx, "num_accounts=%d, num_txs_per_block=%d", x.Cfg.NumAccts, x.Cfg.NumTxsPerBlock)

	go x.obtainTxs(ctx, lgConn)
}

func (x *LoadGen) WaitGenesis(ctx context.Context) *proto.GenesisConfiguration {
	return <-x.genesisCh
}

func (x *LoadGen) WaitInitTxs(ctx context.Context) []*proto.SignedTransaction {
	return <-x.initTxsCh
}

func (x *LoadGen) WaitWorkTxs(ctx context.Context) (proto.ParallelSignedTransactionBatch, bool) {
	txs, ok := <-x.workTxsCh
	return txs, ok
}

func (x *LoadGen) WaitDone() {
	<-x.done
}

func (x *LoadGen) Close() {
}

func (x *LoadGen) obtainTxs(ctx context.Context, lgConn *LoadgenClient) {
	defer lgConn.Close()
	defer close(x.done)

	cfg := proto.ExperimentConfig{}
	cfg.Experiment_type = proto.KExperimentTypeERC20BasicAuthPayments
	*cfg.Erc20_payments() = proto.ERC20PaymentsConfig{Num_addresses: uint64(x.Cfg.NumAccts)}
	r, err := lgConn.Generate_experiment(ctx, cfg)
	must.NoError(ctx, err)

	result := r.Success()
	ctxID := result.Context_id

	liblog.Infof(ctx, "delivering genesis block")
	x.genesisCh <- &result.Genesis_configuration
	close(x.genesisCh)

	liblog.Infof(ctx, "delivering init blocks")
	for _, blockInitTxs := range result.Initialization_txs {
		x.initTxsCh <- protoutil.Unpack(blockInitTxs)
	}
	close(x.initTxsCh)

	liblog.Infof(ctx, "delivering work blocks")
	for range x.Cfg.NumBlocks {
		r, err := lgConn.Extend_experiment(ctx, ctxID, uint64(x.Cfg.NumTxsPerBlock), 1e9)
		must.NoError(ctx, err)

		ext := r.Success()
		x.workTxsCh <- ext.Work_txs
	}
	close(x.workTxsCh)
}
