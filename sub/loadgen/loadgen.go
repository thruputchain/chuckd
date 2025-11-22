package loadgen

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	genesisCh  chan *proto.GenesisBlock
	blockTxsCh chan []*proto.SignedTransaction
	done       chan struct{}
}

func NewLoadGen(
	ctx context.Context,
	rand *rand.Rand,
	inside libnet.Network,
	cfg Config,
) *LoadGen {
	x := &LoadGen{
		Cfg:        cfg,
		rand:       rand,
		inside:     inside,
		genesisCh:  make(chan *proto.GenesisBlock, 1),
		blockTxsCh: make(chan []*proto.SignedTransaction),
		done:       make(chan struct{}),
	}
	return x
}

func (x *LoadGen) Start(ctx context.Context, loadgenAddr string) {

	lgConn, err := DialLoadgenClient(ctx, x.inside, loadgenAddr)
	must.NoError(ctx, err)

	liblog.Infof("num_accounts=%d, num_txs_per_block=%d", x.Cfg.NumAccts, x.Cfg.NumTxsPerBlock)

	go x.obtainTxs(ctx, lgConn)
}

func (x *LoadGen) WaitGenesis(ctx context.Context) *proto.GenesisBlock {
	return <-x.genesisCh
}

func (x *LoadGen) WaitBlockTxs(ctx context.Context) []*proto.SignedTransaction {
	return <-x.blockTxsCh
}

func (x *LoadGen) WaitDone() {
	<-x.done
}

func (x *LoadGen) Close() {
}

func (x *LoadGen) obtainTxs(ctx context.Context, lgConn *LoadgenClient) {
	defer lgConn.Close()
	defer close(x.done)
	defer close(x.blockTxsCh)
	defer close(x.genesisCh)

	cfg := proto.ExperimentConfig{}
	*cfg.Erc20_payments() = proto.ERC20PaymentsConfig{Num_accounts: uint64(x.Cfg.NumAccts)}
	r, err := lgConn.Generate_experiment(ctx, cfg)
	must.NoError(ctx, err)

	result := r.Success()
	ctxID := result.Context_id

	liblog.Infof("delivering genesis block")
	x.genesisCh <- &result.Genesis_block

	liblog.Infof("delivering init blocks")
	for _, blockInitTxs := range result.Initialization_txs {
		x.blockTxsCh <- protoutil.Unpack(blockInitTxs)
	}

	liblog.Infof("delivering work blocks")
	for i := 0; i < x.Cfg.NumBlocks; i++ {
		r, err := lgConn.Extend_experiment(ctx, ctxID, uint64(x.Cfg.NumTxsPerBlock), 1e9)
		must.NoError(ctx, err)

		ext := r.Success()
		x.blockTxsCh <- protoutil.Unpack(ext.Work_txs)
	}
}
