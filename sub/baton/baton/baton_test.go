package baton

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
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
		Skew_msec:          0, //!
		Heartbeat_msec:     roundTTL.Milliseconds() / 5,
		Patience_msec:      roundTTL.Milliseconds(),
		Max_catchup_height: 10000,
	}
	baton := NewBaton(ctx, cfg, rig)
	rig.Bind(baton)
	return ctx, rig, baton, roundTTL
}

// TestNominateCommit tests committing of blocks that we have nominated.
func TestNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	baseHeight := rig.GetTip(ctx).Height

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.PlaceConsensusValue(ctx, *nominating_1.DbgConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx))                                                                                      // txs for stage 3
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nominating_1.DbgConsensusValueCue).Bytes()) // trigger commit, trigger stage 3
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated() == nil {
		t.Fatalf("expecting commit of already validated block")
	}

	// stage 3: nominate [2]
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)

}

// TestValidateExternalized tests that nodes can validate already externalized values.
func TestValidateExternalized(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	baseHeight := rig.GetTip(ctx).Height

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.PlaceConsensusValue(ctx, *nominating_1.DbgConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nominating_1.DbgConsensusValueCue).Bytes()) // trigger commit, trigger stage 3
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated() == nil {
		t.Fatalf("expecting commit of already validated block")
	}

	// consume next externalization
	go func() {
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
		rig.WaitNomination(ctx)
		rig.SignalNomination(true)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	}()

	// request validation of already externalized block
	vlevel := rig.ValidateNominationValue(
		ctx,
		proto.SCPSlotIndex(nominating_1.Height),
		testMarshaler.MarshalXdr(nominating_1.DbgConsensusValueCue).Bytes(),
	)
	// validation is skipped, as the nomination is already cached
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	if <-vlevel != scpkit.FullyValidValue {
		t.Fatalf("expecting fully valid level")
	}
}

// TestValidateCommit tests committing of previously proposed and already validated blocks.
func TestValidateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	baseHeight := rig.GetTip(ctx).Height

	// stage 1: validate [1]
	block := rig.GenNomination(ctx, 2)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)
	rig.PassTime(tick) // ensure nomination time is before validation time

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)

	// consume nomination
	nomination := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.PlaceConsensusValue(ctx, *nomination.DbgConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	liblog.Debugf("STAGE 1: nomination succeeded")

	vlevel := rig.ValidateNominationValue(
		ctx,
		proto.SCPSlotIndex(block.ConsensusValue.Nomination.Block_height),
		testMarshaler.MarshalXdr(nomination.DbgConsensusValueCue).Bytes(),
	)
	// validation is skipped, as the nomination is already cached
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	if <-vlevel != scpkit.FullyValidValue {
		t.Fatalf("expecting fully valid level")
	}
	liblog.Debugf("STAGE 1: validation succeeded")
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nomination.DbgConsensusValueCue).Bytes())
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated() == nil {
		t.Fatalf("expecting commit of already validated block")
	}
}

// TestCatchupMissingBlockCommit tests catchup for missing blocks between tip and externalizations.
// The validator will request catchup twice, because the test catchup system returns one block at a time,
// regardless of how many were requested.
func TestCatchupMissingBlockCommit(t *testing.T) {

	ctx, rig, baton, _ := testSetup()
	baseHeight := rig.GetTip(ctx).Height

	block2 := rig.GenNomination(ctx, 1) // db is initialized at block height 1
	block3 := rig.GenNominationOntoBlock(ctx, 1, block2)

	rig.AddTxs(ctx, protoutil.Unpack(block3.Txs)...) // prevent catchup request for [2]

	rig.StageCatchup(block2.Catchup)
	rig.StageCatchup(block3.Catchup)

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)

	// consume nomination
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+2), testMarshaler.MarshalXdr(&block3.ConsensusValueCue).Bytes())
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	committed2 := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed2.Height != baseHeight+1 {
		t.Errorf("unexpected committed height")
	}

	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	committed3 := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed3.Height != baseHeight+2 {
		t.Errorf("unexpected committed height")
	}
}

// TestCatchupMissingTxsCommit tests catchup for externalizations with missing txs.
func TestCatchupMissingTxsCommit(t *testing.T) {

	ctx, rig, baton, _ := testSetup()
	baseHeight := rig.GetTip(ctx).Height

	block1 := rig.GenNomination(ctx, 1)
	rig.StageCatchup(block1.Catchup)

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(&block1.ConsensusValueCue).Bytes())
	// catchup provisions the block
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.Height != baseHeight+1 {
		t.Errorf("unexpected committed height")
	}
}

func TestAlternatingNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	numRounds := 15

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)

	lastCommitted := testutil.BlockHandleFromTip(rig.GetTipCV(ctx))
	for round := 0; round < numRounds; round++ {

		theirBlock := rig.GenNominationOnto(ctx, 1, lastCommitted)

		// prepare stage
		rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // submit tx
		baton.Kick()                          // wake baton after submitting transactions

		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
		rig.WaitNomination(ctx)
		rig.SignalNomination(true)
		nominated := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

		// kick off validation
		rig.PassTime(tick)
		rig.WaitBackfeed(ctx)
		rig.AddTxs(ctx, protoutil.Unpack(theirBlock.Txs)...)
		rig.PlaceConsensusValue(ctx, theirBlock.ConsensusValue)
		validatedTheirs := rig.ValidateNominationValue(
			ctx,
			proto.SCPSlotIndex(theirBlock.ConsensusValue.Nomination.Block_height),
			testMarshaler.MarshalXdr(&theirBlock.ConsensusValueCue).Bytes(),
		)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
		<-validatedTheirs

		rig.PassTime(tick)
		rig.WaitBackfeed(ctx)
		if round%2 == 1 { // nominate [our], validate [their], committ [our]
			// kick off commit of nominated
			rig.PlaceConsensusValue(ctx, *nominated.DbgConsensusValue)
			rig.ExternalizedValue(
				ctx,
				proto.SCPSlotIndex(nominated.DbgConsensusValue.Nomination.Block_height),
				testMarshaler.MarshalXdr(nominated.DbgConsensusValueCue).Bytes(),
			)
		} else { // nominate [our], validate [their], committ [their]
			// kick off commit of validated
			rig.ExternalizedValue(
				ctx,
				proto.SCPSlotIndex(theirBlock.ConsensusValue.Nomination.Block_height),
				testMarshaler.MarshalXdr(&theirBlock.ConsensusValueCue).Bytes(),
			)
		}
		committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
		lastCommitted = testutil.BlockHandleFromCommittableBlock(committed.DbgValidatable)

		rig.PassTime(tick)
		rig.WaitBackfeed(ctx)
	}

}
