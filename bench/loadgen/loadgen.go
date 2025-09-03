package loadgen

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/baton/validate"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type LoadGen struct {
	numAccts     int
	numTxs       int // txs per block
	genPar       int
	callsPerExec int
	batchSize    int
	rand         *rand.Rand
	inside       libnet.Network
	coord        *coord.CoordinatorService
	handler      *LiveHandler
	insideSrv    *coordinator.Server
	//
	nominateMsec *StatFloat64
	validateMsec *StatFloat64
}

func NewLoadGenBench(
	ctx context.Context,
	rand *rand.Rand,
	inside libnet.Network,
	numAccts int,
	numTxs int, // txs per block
	callsPerExec int,
	batchSize int,
) *LoadGen {
	x := &LoadGen{
		numAccts:     numAccts,
		numTxs:       numTxs,
		callsPerExec: callsPerExec,
		batchSize:    batchSize,
		rand:         rand,
		inside:       inside,
		nominateMsec: NewStatFloat64(),
		validateMsec: NewStatFloat64(),
	}
	x.coord = coord.NewCoordinatorService(ctx, inside, runtime.NewRandFromRand(rand))
	x.insideSrv = coordinator.NewServer(ctx, proto.Coordinator1_Server{Srv: x.coord})
	x.handler = NewLiveHandler(x.coord, runtime.NewRandFromRand(rand))
	return x
}

func (x *LoadGen) Start(ctx context.Context, inAddr string, loadgenAddr string) {
	x.insideSrv.Start(ctx, x.inside, inAddr)
	lg, err := DialLoadgenClient(ctx, x.inside, loadgenAddr)
	must.NoError(ctx, err)

	genesisCh := make(chan *proto.GenesisBlock, 1)
	blockTxsCh := make(chan []*proto.SignedTransaction)

	liblog.Infof("num_accounts=%d, num_txs_per_block=%d", x.numAccts, x.numTxs)

	go x.obtainTxs(ctx, lg, genesisCh, blockTxsCh, x.numTxs, x.numAccts)
	go x.execTxs(ctx, genesisCh, blockTxsCh)
}

func (x *LoadGen) Close() {
	x.insideSrv.Close()
}

func (x *LoadGen) obtainTxs(
	ctx context.Context,
	lg *LoadgenClient,
	genesisCh chan<- *proto.GenesisBlock,
	blockTxsCh chan<- []*proto.SignedTransaction,
	numTxs int,
	numAccts int,
) {

	cfg := proto.ExperimentConfig{}
	*cfg.Erc20_payments() = proto.ERC20PaymentsConfig{Num_txs: uint64(numTxs), Num_accounts: uint64(numAccts)}
	r, err := lg.Generate_experiment(ctx, cfg)
	must.NoError(ctx, err)

	result := r.Success()
	ctxID := result.Context_id

	liblog.Infof("delivering genesis block")
	genesisCh <- &result.Genesis_block

	liblog.Infof("delivering init blocks")
	for _, blockInitTxs := range result.Initialization_txs {
		blockTxsCh <- protoutil.Unpack(blockInitTxs)
	}

	liblog.Infof("delivering work blocks")
	for i := 0; ; i++ {
		r, err := lg.Extend_experiment(ctx, ctxID, uint64(i))
		must.NoError(ctx, err)

		ext := r.Success()
		blockTxsCh <- protoutil.Unpack(ext.Work_txs)
	}
}

func (x *LoadGen) execTxs(
	ctx context.Context,
	genesisCh <-chan *proto.GenesisBlock,
	blockTxsCh <-chan []*proto.SignedTransaction,
) {

	x.coord.WaitDbReady() // returns after db restored
	liblog.Infof("db restored; ready to use")

	db, err := x.coord.DialDb(ctx)
	must.NoError(ctx, err)
	defer db.Close()

	// execute genesis block
	genesisBlock := <-genesisCh
	liblog.Infof("loading genesis block")
	dbr, err := db.Dbcoord_load_genesis_block(ctx, *genesisBlock)
	must.NoError(ctx, err)
	must.Assert(ctx, dbr == proto.DbResultOK)

	// execute blocks
	validationMarshaler := xdrutil.NewXdrMarshaler()
	for {
		tip, err := getDbTip(ctx, db)
		must.NoError(ctx, err)

		liblog.Infof("proposing block at height %d", tip.Height+1)

		x0 := time.Now()
		txs := <-blockTxsCh
		obtainDelayMsec := float64(time.Since(x0).Milliseconds())

		liblog.Infof("proposing block at height %d", tip.Height+1)

		t0 := time.Now()

		nomination, err := nominate.PrepareNomination(ctx, x.handler, tip, txs, x.callsPerExec, x.batchSize)
		must.NoError(ctx, err)

		t1 := time.Now()
		nominateDurMsec := float64(t1.Sub(t0).Milliseconds())
		x.nominateMsec.Add(nominateDurMsec)

		liblog.Infof("validating block at height %d", tip.Height+1)

		vb := &batonapi.ValidatableBlock{
			BlockHeight:          nomination.BlockHeight,
			BlockHash:            nomination.BlockHash,
			PrevBlockHash:        nomination.PrevBlockHash,
			ProposedTransactions: protoutil.UnpackProposedTransactions(nomination.ProposedTransactions()),
			DbTip:                tip,
		}

		t2 := time.Now()

		_, err = validate.ValidateCommit(ctx, validationMarshaler, x.handler, vb, x.callsPerExec, x.batchSize)
		must.NoError(ctx, err)

		t3 := time.Now()
		validateDurMsec := float64(t3.Sub(t2).Milliseconds())
		x.validateMsec.Add(validateDurMsec)

		liblog.Infof("committed block at height %d", tip.Height+1)

		fmt.Printf("obtain_delay_msec=%v | nom_tps=%v val_tps=%v | avg_nom_tps=%v avg_val_tps=%v\n",
			obtainDelayMsec,
			float64(len(txs))/(nominateDurMsec/1e3),
			float64(len(txs))/(validateDurMsec/1e3),
			float64(len(txs))/(x.nominateMsec.Average()/1e3),
			float64(len(txs))/(x.validateMsec.Average()/1e3),
		)
	}
}

func getDbTip(ctx context.Context, c batonapi.DbCoordConn) (batonapi.Tip, error) {
	r, err := c.Dbcoord_get_tip(ctx)
	if err != nil {
		return batonapi.Tip{}, err
	}
	if r.Result != proto.DbResultOK {
		return batonapi.Tip{}, fmt.Errorf("db get tip error")
	}
	return batonapi.Tip{Height: r.Db_tip().Height, Hash: r.Db_tip().Hash, ID: r.Db_tip().Id}, nil
}
