package baton

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

var testMarshaler = xdrutil.NewXdrMarshaler()

func TestMain(m *testing.M) {
	liblog.LogLevel(zerolog.DebugLevel)
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts(true)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func testSetup() (context.Context, *TestRig, *Baton, time.Duration) {
	roundTTL := time.Second
	ctx := context.Background()
	rig := NewTestRig(
		ctx,
		runtime.NewRandFromRoot(),
		time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	cfg := proto.BatonConfig{
		Heartbeat_msec:     proto.Milliseconds(roundTTL) / 5,
		Max_catchup_height: 10000,
	}
	baton := NewBaton(ctx, cfg, rig)
	rig.Bind(baton)
	return ctx, rig, baton, roundTTL
}

// TestNominateCommit tests committing of blocks that we have nominated.
func TestNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	baton.Start(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)

	// stage 2: commit already validated [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx))                                                       // txs for stage 3
	rig.ExternalizedValue(ctx, 1, testMarshaler.MarshalXdr(nominating_1.DbgNomination).Bytes()) // trigger commit, trigger stage 3
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated == nil {
		t.Fatalf("expecting commit of already validated block")
	}

	// stage 3: nominate [2]
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)

}

// TestValidateCommit tests committing of previously proposed and already validated blocks.
func TestValidateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	// stage 1: validate [1]
	baton.Start(ctx)
	block := rig.GenNomination(ctx, 2)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)

	// consume nomination
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

	vlevel := rig.ValidateNominationValue(
		ctx,
		proto.SCPSlotIndex(block.Nomination.Block_height),
		testMarshaler.MarshalXdr(&block.Nomination).Bytes(),
	)
	// validation is skipped, as the nomination is already cached
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	if <-vlevel != scpapi.FullyValidValue {
		t.Fatalf("expecting fully valid level")
	}
	rig.PassTime(tick)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, 1, testMarshaler.MarshalXdr(&block.Nomination).Bytes())
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated == nil {
		t.Fatalf("expecting commit of already validated block")
	}
}

// TestCatchupMissingBlockCommit tests catchup for missing blocks between tip and externalizations.
func TestCatchupMissingBlockCommit(t *testing.T) {

	ctx, rig, baton, _ := testSetup()

	block1 := rig.GenNomination(ctx, 1)
	block2 := rig.GenNominationOntoBlock(ctx, 1, block1)

	rig.AddTxs(ctx, protoutil.Unpack(block2.Txs)...) // prevent catchup request for [2]

	rig.StageCatchup(block1.Catchup)
	rig.StageCatchup(block2.Catchup)

	baton.Start(ctx)

	// consume nomination
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

	rig.ExternalizedValue(ctx, 2, testMarshaler.MarshalXdr(&block2.Nomination).Bytes())
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.Height != 1 {
		t.Errorf("expecting committed height 1")
	}
}

// TestCatchupMissingTxsCommit tests catchup for externalizations with missing txs.
func TestCatchupMissingTxsCommit(t *testing.T) {

	ctx, rig, baton, _ := testSetup()

	block1 := rig.GenNomination(ctx, 1)
	rig.StageCatchup(block1.Catchup)

	baton.Start(ctx)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.ExternalizedValue(ctx, 1, testMarshaler.MarshalXdr(&block1.Nomination).Bytes())
	// catchup provisions the block
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated) //XXX: hanging; trying repair txs, but should be trying catchup
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.Height != 1 {
		t.Errorf("expecting committed height 1")
	}

}

func TestAlternatingNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	maxHeight := proto.BlockHeight(15)

	baton.Start(ctx)

	lastCommitted := testutil.TipHandle(rig.getTip(ctx))
	for height := proto.BlockHeight(1); height < maxHeight; height++ {

		theirBlock := rig.GenNominationOnto(ctx, 1, lastCommitted)

		// prepare stage
		rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // submit tx
		baton.Kick()                          // wake baton after submitting transactions

		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
		rig.WaitNomination(ctx)
		rig.SignalNomination(true)
		nominated := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

		// kick off validation
		rig.AddTxs(ctx, protoutil.Unpack(theirBlock.Txs)...)
		validatedTheirs := rig.ValidateNominationValue(
			ctx,
			proto.SCPSlotIndex(height),
			testMarshaler.MarshalXdr(&theirBlock.Nomination).Bytes(),
		)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
		<-validatedTheirs

		if height%2 == 1 { // nominate [our], validate [their], committ [our]
			// kick off commit of nominated
			rig.ExternalizedValue(ctx, proto.SCPSlotIndex(height), testMarshaler.MarshalXdr(nominated.DbgNomination).Bytes())
		} else { // nominate [our], validate [their], committ [their]
			// kick off commit of validated
			rig.ExternalizedValue(ctx, proto.SCPSlotIndex(height), testMarshaler.MarshalXdr(&theirBlock.Nomination).Bytes())
		}
		committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
		lastCommitted.Hash, lastCommitted.Height = committed.Hash, committed.Height

		rig.PassTime(tick)
	}

}
