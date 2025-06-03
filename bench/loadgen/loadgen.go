package loadgen

import (
	"context"
	"fmt"
	"math/rand"

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
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type LoadGen struct {
	tpb       int // txs per block
	rand      *rand.Rand
	inside    libnet.Network
	coord     *coord.CoordinatorService
	handler   *LiveHandler
	insideSrv *system.Server
}

func NewLoadGenBench(
	ctx context.Context,
	rand *rand.Rand,
	inside libnet.Network,
	tpb int, // txs per block
) *LoadGen {
	x := &LoadGen{
		tpb:    tpb,
		rand:   rand,
		inside: inside,
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

	txCh := make(chan *proto.SignedTransaction, x.tpb)
	go x.generateTxs(ctx, runtime.NewRandFromRand(x.rand), txCh)
	for {

		tip, err := getDbTip(ctx, c)
		must.NoError(ctx, err)

		liblog.Infof("proposing block at height %d", tip.Height+1)

		txs := make([]*proto.SignedTransaction, 0, x.tpb)
		for i := 0; i < x.tpb; i++ {
			txs = append(txs, <-txCh)
		}

		nomination, err := nominate.PrepareNomination(ctx, x.handler, tip, txs)
		must.NoError(ctx, err)

		vb := &batonapi.ValidatableBlock{
			BlockHeight:          nomination.BlockHeight,
			BlockHash:            nomination.BlockHash,
			PrevBlockHash:        nomination.PrevBlockHash,
			ProposedTransactions: protoutil.UnpackProposedTransactions(nomination.ProposedTransactions()),
			DbTip:                tip,
		}

		liblog.Infof("validating block at height %d", tip.Height+1)

		_, err = validate.ValidateCommit(ctx, x.handler, vb)
		must.NoError(ctx, err)

		liblog.Infof("committed block at height %d", tip.Height+1)

	}
}

func (x *LoadGen) generateTxs(ctx context.Context, rand *rand.Rand, txCh chan<- *proto.SignedTransaction) {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)
	gen := testutil.NewTestGen(ctx, libtime.RealTime{}, rand, user.Secret_key)
	for {
		tx := gen.GenSignedTx(ctx)
		select {
		case txCh <- tx:
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
