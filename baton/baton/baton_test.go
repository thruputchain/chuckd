package baton

import (
	"context"
	"os"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	liblog.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts(true)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func testSetup() (context.Context, *TestRig, *Baton, time.Duration) {
	tick := time.Second
	ctx := context.Background()
	rig := NewTestRig(
		ctx,
		runtime.NewRandFromRoot(),
		time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
		3*tick, // txTTL
		5*tick, // committedTTL
	)
	callsPerExec := 2
	batchSize := 10
	baton := NewBaton(ctx, rig, callsPerExec, batchSize)
	rig.Bind(baton)
	return ctx, rig, baton, tick
}

func TestSubmittedExpiry(t *testing.T) {
	ctx, rig, _, tick := testSetup()

	block := rig.GenNomination(ctx, 1)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)
	rig.PassTime(tick * 5)
	goruntime.Gosched()
	time.Sleep(time.Millisecond) // XXX: flaky
	_, ok := rig.LookupTxs(ctx, []proto.SignedTransactionHash{block.Nomination.Transaction_hashes[0]})
	if ok {
		t.Errorf("expecting txs to be missing")
	}
}

// TestNominateCommit tests committing of blocks that we have nominated.
func TestNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	rig.StageExecs(1)
	baton.Start(ctx, tick/5)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonapi.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominated)
	rig.PassTime(tick)

	// stage 2: commit already validated [1]
	rig.StageExecs(1)
	rig.AddTxs(ctx, rig.GenSignedTx(ctx))                                      // txs for stage 3
	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(nominating_1.Nomination)) // trigger commit, trigger stage 3
	committed := rig.ReceiveEvent(ctx).Event.(batonapi.EventCommitted)
	if committed.AlreadyValidatedBlockID == nil {
		t.Fatalf("expecting commit of already validated block")
	}

	// stage 3: nominate [2]
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominated)
	rig.PassTime(tick)

}

// TestValidateCommit tests committing of previously proposed and already validated blocks.
func TestValidateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	// stage 1: validate [1]
	rig.StageExecs(1)
	baton.Start(ctx, tick/5)
	block := rig.GenNomination(ctx, 2)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)

	// consume nomination
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominated)

	vlevel := rig.ValidateValue(ctx, block.Nomination.Block_height, xdrutil.MarshalXdr(&block.Nomination), true)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidated)
	if <-vlevel != scpapi.FullyValidValue {
		t.Fatalf("expecting fully valid level")
	}
	rig.PassTime(tick)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(&block.Nomination))
	committed := rig.ReceiveEvent(ctx).Event.(batonapi.EventCommitted)
	if committed.AlreadyValidatedBlockID == nil {
		t.Fatalf("expecting commit of already validated block")
	}
	rig.PassTime(3 * tick)
	goruntime.Gosched()
	time.Sleep(time.Millisecond) //XXX: flaky -race
	_, ok := rig.LookupTxs(ctx, block.Nomination.Transaction_hashes)
	if ok {
		t.Fatalf("txs not removed after expiry")
	}

}

// TestCatchupMissingBlockCommit tests catchup for missing blocks between tip and externalizations.
func TestCatchupMissingBlockCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	block1 := rig.GenNomination(ctx, 1)
	block2 := rig.GenNominationOntoBlock(ctx, 1, block1)
	rig.AddTxs(ctx, protoutil.Unpack(block2.Txs)...) // prevent catchup request for [2]
	rig.StageCatchup(block1.Catchup)

	baton.Start(ctx, tick/5)
	rig.StageExecs(1) // for validation and commit

	// consume nomination
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominated)

	rig.ExternalizedValue(ctx, 2, xdrutil.MarshalXdr(&block2.Nomination))
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidated)
	committed := rig.ReceiveEvent(ctx).Event.(batonapi.EventCommitted)
	if committed.BlockHeight != 1 {
		t.Errorf("expecting committed height 1")
	}
}

// TestCatchupMissingTxsCommit tests catchup for externalizations with missing txs.
func TestCatchupMissingTxsCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	block1 := rig.GenNomination(ctx, 1)
	rig.StageCatchup(block1.Catchup)

	baton.Start(ctx, tick/5)
	rig.StageExecs(1) // for validation and commit
	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(&block1.Nomination))
	// catchup provisions the block
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidated)
	committed := rig.ReceiveEvent(ctx).Event.(batonapi.EventCommitted)
	if committed.BlockHeight != 1 {
		t.Errorf("expecting committed height 1")
	}

}

func TestAlternatingNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	maxHeight := proto.BlockHeight(15)

	baton.Start(ctx, tick/5)

	lastCommitted := testutil.TipHandle(rig.GetTip())
	for height := proto.BlockHeight(1); height < maxHeight; height++ {

		theirBlock := rig.GenNominationOnto(ctx, 1, lastCommitted)

		// prepare stage
		rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // submit tx
		baton.Kick()                          // wake baton after submitting transactions

		_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventNominating)
		rig.WaitNomination(ctx)
		rig.SignalNomination(true)
		nominated := rig.ReceiveEvent(ctx).Event.(batonapi.EventNominated)

		// kick off validation
		rig.AddTxs(ctx, protoutil.Unpack(theirBlock.Txs)...)
		rig.ValidateValue(ctx, height, xdrutil.MarshalXdr(&theirBlock.Nomination), true)
		_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidating)
		_ = rig.ReceiveEvent(ctx).Event.(batonapi.EventValidated)

		if height%2 == 1 { // nominate [our], validate [their], committ [our]
			// kick off commit of nominated
			rig.ExternalizedValue(ctx, height, xdrutil.MarshalXdr(nominated.Nomination))
		} else { // nominate [our], validate [their], committ [their]
			// kick off commit of validated
			rig.ExternalizedValue(ctx, height, xdrutil.MarshalXdr(&theirBlock.Nomination))
		}
		committed := rig.ReceiveEvent(ctx).Event.(batonapi.EventCommitted)
		lastCommitted = testutil.VBHandle(committed.ValidatableBlock)

		rig.PassTime(tick)
	}

}
