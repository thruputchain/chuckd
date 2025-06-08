package loadgen

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/baton/validate"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type LoadGen struct {
	tpb          int // txs per block
	genPar       int
	callsPerExec int
	batchSize    int
	rand         *rand.Rand
	inside       libnet.Network
	coord        *coord.CoordinatorService
	handler      *LiveHandler
	insideSrv    *system.Server
	//
	nominateMsec *StatFloat64
	validateMsec *StatFloat64
}

func NewLoadGenBench(
	ctx context.Context,
	rand *rand.Rand,
	inside libnet.Network,
	tpb int, // txs per block
	genPar int,
	callsPerExec int,
	batchSize int,
) *LoadGen {
	x := &LoadGen{
		tpb:          tpb,
		genPar:       genPar,
		callsPerExec: callsPerExec,
		batchSize:    batchSize,
		rand:         rand,
		inside:       inside,
		nominateMsec: NewStatFloat64(),
		validateMsec: NewStatFloat64(),
	}
	x.coord = coord.NewCoordinatorService(ctx, inside, runtime.NewRandFromRand(rand))
	x.insideSrv = system.NewServer(ctx, proto.Coordinator1_Server{Srv: x.coord})
	x.handler = NewLiveHandler(x.coord, runtime.NewRandFromRand(rand))
	return x
}

func (x *LoadGen) Start(ctx context.Context, inAddr string) {
	x.insideSrv.Start(ctx, x.inside, inAddr)
	go x.drive(ctx)
}

func (x *LoadGen) Close() {
	x.insideSrv.Close()
}

func (x *LoadGen) drive(ctx context.Context) {

	x.coord.WaitDbReady() // returns after db restored

	c, err := x.coord.DialDb(ctx)
	must.NoError(ctx, err)
	defer c.Close()

	blockTxsCh := make(chan []*proto.SignedTransaction)
	go x.generateTxsWithoutPool(ctx, runtime.NewRandFromRand(x.rand), blockTxsCh, x.tpb)
	txs := <-blockTxsCh
	i := 0
	for {

		tip, err := getDbTip(ctx, c)
		must.NoError(ctx, err)

		liblog.Infof("proposing block at height %d", tip.Height+1)

		x0 := time.Now()
		// txs := <-blockTxsCh
		genDurMsec := float64(time.Now().Sub(x0).Milliseconds())

		t0 := time.Now()

		// gcPrevent := debug.SetGCPercent(-1) //XXX: disable GC

		nomination, err := nominate.PrepareNomination(ctx, x.handler, tip, txs, x.callsPerExec, x.batchSize)
		must.NoError(ctx, err)

		t1 := time.Now()
		nominateDurMsec := float64(t1.Sub(t0).Milliseconds())
		x.nominateMsec.Add(nominateDurMsec)

		vb := &batonapi.ValidatableBlock{
			BlockHeight:          nomination.BlockHeight,
			BlockHash:            nomination.BlockHash,
			PrevBlockHash:        nomination.PrevBlockHash,
			ProposedTransactions: protoutil.UnpackProposedTransactions(nomination.ProposedTransactions()),
			DbTip:                tip,
		}

		t2 := time.Now()

		liblog.Infof("validating block at height %d", tip.Height+1)

		_, err = validate.ValidateClose(ctx, x.handler, vb, x.callsPerExec, x.batchSize)
		must.NoError(ctx, err)

		t3 := time.Now()
		validateDurMsec := float64(t3.Sub(t2).Milliseconds())
		x.validateMsec.Add(validateDurMsec)

		liblog.Infof("done at height %d", tip.Height+1)

		// debug.SetGCPercent(gcPrevent) //XXX: reenable GC

		fmt.Printf("gen_wait_msec=%v | nom_tps=%v val_tps=%v | avg_nom_tps=%v avg_val_tps=%v\n",
			genDurMsec,
			float64(len(txs))/(nominateDurMsec/1e3),
			float64(len(txs))/(validateDurMsec/1e3),
			float64(len(txs))/(x.nominateMsec.Average()/1e3),
			float64(len(txs))/(x.validateMsec.Average()/1e3),
		)
		i += 1
	}
}

func (x *LoadGen) generateTxs(
	ctx context.Context,
	rand *rand.Rand,
	txCh chan<- []*proto.SignedTransaction,
	tpb int,
) {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)
	gen := testutil.NewTestGen(ctx, libtime.RealTime{}, rand, user.Secret_key)
	for {
		blockTxs := make([]*proto.SignedTransaction, tpb)
		for i := range blockTxs {
			blockTxs[i] = gen.GenSignedTx(ctx)
		}
		select {
		case txCh <- blockTxs:
		case <-ctx.Done():
			close(txCh)
			return
		}
	}
}

func (x *LoadGen) generateTxsWithPool(
	ctx context.Context,
	rand *rand.Rand,
	txCh chan<- []*proto.SignedTransaction,
	tpb int,
) {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)

	k := x.genPar

	gen := make([]*testutil.TestGen, k)
	for i := range gen {
		gen[i] = testutil.NewTestGen(ctx, libtime.RealTime{}, runtime.NewRandFromRand(rand), user.Secret_key)
	}

	pool, err := ants.NewPool(k)
	must.NoError(ctx, err)
	for {
		blockTxs := make([]*proto.SignedTransaction, tpb)

		t0 := time.Now()

		chunk := util.CeilDivSigned(tpb, k)
		var wg sync.WaitGroup
		for j := 0; j < k; j++ {
			wg.Add(1)
			err = pool.Submit(
				func() {
					for i := j * chunk; i < min(tpb, (j+1)*chunk); i++ {
						blockTxs[i] = gen[j].GenSignedTx(ctx)
					}
					wg.Done()
				},
			)
			must.NoError(ctx, err)
		}
		wg.Wait()

		dur := time.Now().Sub(t0)
		liblog.Infof("generation %v tps", float64(tpb)/(float64(dur.Milliseconds())/1e3))

		select {
		case txCh <- blockTxs:
		case <-ctx.Done():
			close(txCh)
			return
		}
	}
}

func (x *LoadGen) generateTxsWithoutPool(
	ctx context.Context,
	rand *rand.Rand,
	txCh chan<- []*proto.SignedTransaction,
	tpb int,
) {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)

	k := x.genPar

	gen := make([]*testutil.TestGen, k)
	for i := range gen {
		gen[i] = testutil.NewTestGen(ctx, libtime.RealTime{}, runtime.NewRandFromRand(rand), user.Secret_key)
	}

	for {
		blockTxs := make([]*proto.SignedTransaction, tpb)

		t0 := time.Now()

		chunk := util.CeilDivSigned(tpb, k)
		var wg sync.WaitGroup
		for j := 0; j < k; j++ {
			wg.Add(1)
			go func() {
				for i := j * chunk; i < min(tpb, (j+1)*chunk); i++ {
					blockTxs[i] = gen[j].GenSignedTx(ctx)
				}
				wg.Done()
			}()
		}
		wg.Wait()

		dur := time.Now().Sub(t0)
		liblog.Infof("generation %v tps", float64(tpb)/(float64(dur.Milliseconds())/1e3))

		select {
		case txCh <- blockTxs:
		case <-ctx.Done():
			close(txCh)
			return
		}
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
