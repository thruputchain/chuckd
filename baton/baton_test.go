package baton

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/xdrutil"
)

func TestMain(m *testing.M) {
	log.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
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
		3*tick, // batchTTL
		3*tick, // txTTL
		5*tick, // committedTTL
	)
	baton := NewBaton(ctx, rig)
	rig.Bind(baton)
	return ctx, rig, baton, tick
}

func TestNominateCommit(t *testing.T) {

	ctx, rig, baton, tick := testSetup()

	// stage 1: nominate 1
	rig.AddTx(ctx, rig.GenSignedTx(ctx)) // user submits tx
	rig.StageExecs(1)
	baton.Start(ctx)
	nominating_1 := rig.ReceiveEvent(ctx).Event.(EventNominating)
	rig.SignalNomination(true)
	rig.WaitNextNomination(ctx)
	_ = rig.ReceiveEvent(ctx).Event.(EventNominated)
	rig.PassTime(tick)

	// stage 2: commit already validated 1
	rig.StageExecs(1)
	rig.AddTx(ctx, rig.GenSignedTx(ctx))                                     // txs for stage 3
	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(nominating_1.Proposal)) // trigger commit, trigger stage 3
	_ = rig.ReceiveEvent(ctx).Event.(EventCommitted)

	// stage 3: nominate 2
	rig.SignalNomination(true)
	rig.WaitNextNomination(ctx)
	_ = rig.ReceiveEvent(ctx).Event.(EventNominating)
	_ = rig.ReceiveEvent(ctx).Event.(EventNominated)
	rig.PassTime(tick)

}

// func TestValidateCommit(t *testing.T) {

// 	ctx, rig, baton, tick := testSetup()

// 	// stage 1: validate 1
// 	rig.StageExecs(1)
// 	baton.Start(ctx)

// 	proposal := rig.MakeProposal(ctx)
// 	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(proposal))

// }
