package baton

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libruntime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

var testMarshaler = libxdr.NewXdrMarshaler()

func TestMain(m *testing.M) {
	liblog.SetLogLevel(zerolog.DebugLevel, true)
	runtime.SetRootSeed(0)
	libruntime.InstallStackTraceOnInterrupt()
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

func TestValidateValueRejectsUnsupportedProtocol(t *testing.T) {
	ctx, rig, baton, _ := testSetup()
	block := rig.GenNomination(ctx, 1)
	block.ConsensusValue.Protocol = proto.Protocol(1)

	baton.Start(ctx)
	level := <-rig.ValidateNominationValue(
		ctx,
		scpkit.Slot(block.ConsensusValue.Proposal_header.Block_height),
		testMarshaler.MarshalXdr(&block.ConsensusValue).Bytes(),
	)
	if level != scpkit.InvalidValue {
		t.Fatalf("validation level = %v, want invalid", level)
	}
}

// TestNominateCommit tests committing of blocks that we have nominated.
func TestNominateCommit(t *testing.T) {

	reader := installTestMeterProvider(t)
	ctx, rig, baton, tick := testSetup()
	baseHeight := rig.GetTipHeight(ctx)

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	if nominating_1.ConsensusValue.Protocol != proto.Protocol(0) {
		t.Fatalf("consensus protocol = %d, want 0", nominating_1.ConsensusValue.Protocol)
	}
	rig.PlaceConsensusValue(ctx, *nominating_1.ConsensusBasis, *nominating_1.ConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx))                                                                                // txs for stage 3
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nominating_1.ConsensusValue).Bytes()) // trigger commit, trigger stage 3
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

	data := collectBatonMetrics(t, reader)
	if height := findInt64Gauge(t, data, "coordinator_current_block_height"); height != baseHeight+1 {
		t.Fatalf("coordinator height = %d, want %d", height, baseHeight+1)
	}
	if committed := findInt64Counter(t, data, "coordinator_committed_blocks"); committed != 1 {
		t.Fatalf("committed blocks = %d, want 1", committed)
	}

}

// TestValidateExternalized tests that nodes can validate already externalized values.
func TestValidateExternalized(t *testing.T) {

	ctx, rig, baton, tick := testSetup()
	baseHeight := rig.GetTipHeight(ctx)

	// stage 1: nominate [1]
	rig.AddTxs(ctx, rig.GenSignedTx(ctx)) // user submits tx
	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.PlaceConsensusValue(ctx, *nominating_1.ConsensusBasis, *nominating_1.ConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nominating_1.ConsensusValue).Bytes()) // trigger commit, trigger stage 3
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
		testMarshaler.MarshalXdr(nominating_1.ConsensusValue).Bytes(),
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
	baseHeight := rig.GetTipHeight(ctx)

	// stage 1: validate [1]
	block := rig.GenNomination(ctx, 2)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)
	rig.PassTime(tick) // ensure nomination time is before validation time

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)

	// consume nomination
	nomination := rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.PlaceConsensusValue(ctx, *nomination.ConsensusBasis, *nomination.ConsensusValue)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)
	liblog.Debugf(ctx, "STAGE 1: nomination succeeded")

	vlevel := rig.ValidateNominationValue(
		ctx,
		proto.SCPSlotIndex(block.ConsensusValue.Proposal_header.Block_height),
		testMarshaler.MarshalXdr(nomination.ConsensusValue).Bytes(),
	)
	// validation is skipped, as the nomination is already cached
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
	// _ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
	if <-vlevel != scpkit.FullyValidValue {
		t.Fatalf("expecting fully valid level")
	}
	liblog.Debugf(ctx, "STAGE 1: validation succeeded")
	rig.PassTime(tick)
	rig.WaitBackfeed(ctx)

	// stage 2: commit already validated [1]
	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(nomination.ConsensusValue).Bytes())
	committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
	if committed.DbgValidatable.AlreadyValidated() == nil {
		t.Fatalf("expecting commit of already validated block")
	}
}

func TestStartRestoresNonGenesisTipWithCommittedCV(t *testing.T) {
	reader := installTestMeterProvider(t)
	ctx, rig, baton, _ := testSetup()

	block := rig.GenNomination(ctx, 1)
	rig.AddTxs(ctx, protoutil.Unpack(block.Txs)...)
	validated, err := rig.Schedulercoord_validate_block(ctx, rig.GetTip(ctx).Id, *protoutil.ProposalFromBasis(&block.ConsensusBasis))
	if err != nil {
		t.Fatalf("validate block: %v", err)
	}
	if validated.Outcome != proto.SchedulerValidationOK {
		t.Fatalf("validate block outcome: %v", validated.Outcome)
	}
	value := protoutil.BasisValue(&block.ConsensusBasis)
	if err := rig.CVPoolCommit(ctx, block.ConsensusBasis, *value); err != nil {
		t.Fatalf("commit cv: %v", err)
	}
	if outcome, err := rig.Schedulercoord_commit_block(ctx, validated.Block().Block_id, *value); err != nil || outcome != proto.SchedulerCommitOK {
		t.Fatalf("commit block: outcome=%v err=%v", outcome, err)
	}

	baton.Start(ctx)
	if protoutil.ValueHeight(&baton.tipDb.Value) != 1 {
		t.Fatalf("restored height = %d, want 1", protoutil.ValueHeight(&baton.tipDb.Value))
	}
	data := collectBatonMetrics(t, reader)
	if height := findInt64Gauge(t, data, "coordinator_current_block_height"); height != 1 {
		t.Fatalf("coordinator height = %d, want 1", height)
	}
}

func TestStartCommitsLocallyPersistedCatchupAboveDbTip(t *testing.T) {
	ctx, rig, baton, _ := testSetup()
	block := rig.GenNomination(ctx, 1)

	if err := rig.CVPoolCommit(ctx, block.ConsensusBasis, block.ConsensusValue); err != nil {
		t.Fatalf("commit consensus value: %v", err)
	}
	if err := rig.CBPoolAdd(ctx, block.Catchup, block.ConsensusValue); err != nil {
		t.Fatalf("add catchup block: %v", err)
	}
	if err := rig.CBPoolWaitPersist(ctx, block.ConsensusValue); err != nil {
		t.Fatalf("persist catchup block: %v", err)
	}
	rig.TestCatchupStaged = nil

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	committed := waitForCommittedEvent(t, rig)
	wantHeight := protoutil.ValueHeight(&block.ConsensusBasis.Value)
	if committed.Height != wantHeight {
		t.Fatalf("committed height = %d, want %d", committed.Height, wantHeight)
	}
	if height := rig.GetTipHeight(ctx); height != wantHeight {
		t.Fatalf("scheduler height = %d, want %d", height, wantHeight)
	}
	if queries := rig.catchupQueries.Load(); queries != 0 {
		t.Fatalf("network catchup queries = %d, want 0", queries)
	}
}

func TestRestoreLocalCatchupRejectsInvalidBlock(t *testing.T) {
	ctx, rig, baton, _ := testSetup()
	block := rig.GenNomination(ctx, 1)
	block.Catchup.Signed_transactions = proto.ParallelSignedTransactionBatch{}

	gotErr := must.Try(func() {
		baton.mustRestoreLocalCatchup(ctx, &block.Catchup, &block.ConsensusBasis)
	})
	if gotErr == nil || !strings.Contains(gotErr.Error(), "invalid local catchup block at height 1") {
		t.Fatalf("expected invalid local catchup failure, got %v", gotErr)
	}
}

func TestRestoreLocalCatchupRejectsConsensusMismatch(t *testing.T) {
	ctx, rig, baton, _ := testSetup()
	want := rig.GenNomination(ctx, 1)
	wrong := rig.GenNomination(ctx, 0)

	gotErr := must.Try(func() {
		baton.mustRestoreLocalCatchup(ctx, &wrong.Catchup, &want.ConsensusBasis)
	})
	if gotErr == nil || !strings.Contains(gotErr.Error(), "local catchup block disagrees with consensus at height 1") {
		t.Fatalf("expected local catchup consensus mismatch, got %v", gotErr)
	}
}

func TestCatchupWithoutProvidersWaitsForHeartbeatBeforeRetry(t *testing.T) {
	ctx, rig, baton, tick := testSetup()
	block := rig.GenNomination(ctx, 1)
	if err := rig.CVPoolCommit(ctx, block.ConsensusBasis, block.ConsensusValue); err != nil {
		t.Fatalf("commit consensus value: %v", err)
	}
	rig.TestCatchupStaged = nil

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	waitForCatchupQueries(t, rig, 1)
	waitForCatchupIdle(t, baton)
	if queries := rig.catchupQueries.Load(); queries != 1 {
		t.Fatalf("catchup queries before heartbeat = %d, want 1", queries)
	}

	rig.PassTime(tick)
	waitForCatchupQueries(t, rig, 2)
}

func waitForCatchupIdle(t *testing.T, baton *Baton) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		baton.lockNoDrive()
		idle := len(baton.catchupWaiting) == 0
		baton.unlockNoDrive()
		if idle {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for catchup attempt to finish")
}

func waitForCommittedEvent(t *testing.T, rig *TestRig) batonkit.EventCommitted {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case envelope := <-rig.emitted:
			if committed, ok := envelope.Event.(batonkit.EventCommitted); ok {
				return committed
			}
		case <-timer.C:
			t.Fatal("timed out waiting for committed event")
		}
	}
}

func waitForCatchupQueries(t *testing.T, rig *TestRig, want uint64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if rig.catchupQueries.Load() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("catchup queries = %d, want at least %d", rig.catchupQueries.Load(), want)
}

// TestCatchupMissingBlockCommit tests catchup for missing blocks between tip and externalizations.
// The validator will request catchup twice, because the test catchup system returns one block at a time,
// regardless of how many were requested.
func TestCatchupMissingBlockCommit(t *testing.T) {

	ctx, rig, baton, _ := testSetup()
	baseHeight := rig.GetTipHeight(ctx)

	block2 := rig.GenNomination(ctx, 1) // db is initialized at block height 1
	block3 := rig.GenProposalOntoBlock(ctx, 1, block2)

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

	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+2), testMarshaler.MarshalXdr(&block3.ConsensusValue).Bytes())
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
	baseHeight := rig.GetTipHeight(ctx)

	block1 := rig.GenNomination(ctx, 1)
	rig.StageCatchup(block1.Catchup)

	baton.Start(ctx)
	rig.WaitBackfeed(ctx)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominating)
	rig.WaitNomination(ctx)
	rig.SignalNomination(true)
	_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventNominated)

	rig.ExternalizedValue(ctx, scpkit.Slot(baseHeight+1), testMarshaler.MarshalXdr(&block1.ConsensusValue).Bytes())
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

	lastCommitted := testutil.BlockHandleFromTip(rig.GetTip(ctx))
	for round := 0; round < numRounds; round++ {

		theirBlock := rig.GenProposalOnto(ctx, 1, lastCommitted)

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
		rig.PlaceConsensusValue(ctx, theirBlock.ConsensusBasis, theirBlock.ConsensusValue)
		validatedTheirs := rig.ValidateNominationValue(
			ctx,
			proto.SCPSlotIndex(theirBlock.ConsensusValue.Proposal_header.Block_height),
			testMarshaler.MarshalXdr(&theirBlock.ConsensusValue).Bytes(),
		)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidating)
		_ = rig.ReceiveEvent(ctx).Event.(batonkit.EventValidated)
		<-validatedTheirs

		rig.PassTime(tick)
		rig.WaitBackfeed(ctx)
		if round%2 == 1 { // nominate [our], validate [their], committ [our]
			// kick off commit of nominated
			rig.PlaceConsensusValue(ctx, *nominated.ConsensusBasis, *nominated.ConsensusValue)
			rig.ExternalizedValue(
				ctx,
				proto.SCPSlotIndex(nominated.ConsensusValue.Proposal_header.Block_height),
				testMarshaler.MarshalXdr(nominated.ConsensusValue).Bytes(),
			)
		} else { // nominate [our], validate [their], committ [their]
			// kick off commit of validated
			rig.ExternalizedValue(
				ctx,
				proto.SCPSlotIndex(theirBlock.ConsensusValue.Proposal_header.Block_height),
				testMarshaler.MarshalXdr(&theirBlock.ConsensusValue).Bytes(),
			)
		}
		committed := rig.ReceiveEvent(ctx).Event.(batonkit.EventCommitted)
		lastCommitted = testutil.BlockHandleFromCommittableBlock(committed.DbgValidatable)

		rig.PassTime(tick)
		rig.WaitBackfeed(ctx)
	}

}
