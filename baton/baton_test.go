package baton

import (
	"context"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/xdrutil"
)

func TestExt(t *testing.T) {

	log.LogVerbosely()
	runtime.SetRootSeed(0)

	roundDuration := time.Second
	ctx := context.Background()
	rig := NewTestRig(
		ctx,
		runtime.NewRandFromRoot(),
		time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
		3*roundDuration, // batchTTL
		3*roundDuration, // txTTL
		5*roundDuration, // committedTTL
	)
	baton := NewBaton(ctx, rig)
	rig.Bind(baton)

	tick := time.Second

	rig.AddTx(ctx, rig.MakeSignedTx(ctx)) // user submits tx

	baton.Start(ctx)

	nominating := rig.ReceiveEvent(ctx).Event.(EventNominating)

	rig.PassTime(tick)

	rig.ExternalizedValue(ctx, 1, xdrutil.MarshalXdr(nominating.Proposal))

	select {}
}
