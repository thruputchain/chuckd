package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) SCPHandler() scpkit.SCPHandler {
	return scpHandler{b}
}

type scpHandler struct {
	baton *Baton
}

func (x scpHandler) CombineCandidates(ctx context.Context, slot scpkit.Slot, values []scpkit.Value) scpkit.Value {
	<-x.baton.ready

	if len(values) == 0 {
		return nil
	} else {
		scpkit.Values(values).Sort()
		return values[0]
	}
}

func (x scpHandler) ValidateBallotValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	return x.ValidateCue(ctx, slot, value, false)
}

func (x scpHandler) ValidateNominationValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	return x.ValidateCue(ctx, slot, value, true)
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) *scpkit.Value {
	<-x.baton.ready
	return &value
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) {
	<-x.baton.ready

	var cv proto.ConsensusValue
	if err := xdrutil.UnmarshalXdr(&cv, value); err != nil {
		return
	}

	// check that SCP slot equals nomination's block height
	if slot != proto.SCPSlotIndex(cv.Nomination.Block_height) {
		liblog.Errorf("scp slot index does not equal proposed block height")
		return
	}

	x.baton.lock()
	defer x.baton.unlock()

	ext := x.baton.newExternalized(&cv)
	if isNewExternalization(ctx, x.baton.tipDb, ext, x.baton.externalized) {
		x.baton.handler.PersistConsensusValue(ctx, cv)
		x.baton.externalized = append(x.baton.externalized, ext)
	}
}

// isNewExternalization checks that an externalization has not already been seen;
// scp can produce duplicate externalizations.
func isNewExternalization(
	ctx context.Context,
	tip proto.DbTip,
	ext externalized,
	priorExts []externalized,
) bool {
	if ext.Height() <= tip.Height {
		return false
	}
	if len(priorExts) == 0 {
		return true // no prior externalizations
	}
	lastExt := priorExts[len(priorExts)-1]
	if ext.Height() <= lastExt.Height() {
		return false
	}
	must.Assertf(ctx,
		ext.Height() != lastExt.Height()+1 || ext.PrevBlockHash() == lastExt.BlockHash(),
		"scp externalization is invalid",
	)
	return true
}
