package baton

import (
	"context"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/runtime"
)

func TestExt(t *testing.T) {

	roundDuration := time.Second
	ctx := context.Background()
	runtime.SetRootSeed(0)
	rig := NewTestRig(
		ctx,
		runtime.NewRandFromRoot(),
		time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
		5*roundDuration, // batchTTL
		3*roundDuration, // txTTL
		5*roundDuration, // committedTTL
	)
	baton := NewBaton(ctx, rig)
	rig.Bind(baton)

	// add user tx
	rig.AddTx(ctx, rig.MakeSignedTx(ctx))

	baton.Start(ctx)
	// XXX
}
