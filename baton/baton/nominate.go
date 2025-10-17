package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	Onto   proto.DbTip
	Cancel context.CancelFunc
}

type nominated struct {
	time.Time
	ConsensusValue *proto.ConsensusValue
}

func (b *Baton) canNominate(tipToNominateOn *proto.DbTip, nominating *nominating, nominated *nominated) bool {
	if tipToNominateOn == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return tipToNominateOn.Height+1 > nominated.ConsensusValue.Nomination.Block_height
	case nominating != nil && nominated == nil:
		return tipToNominateOn.Height > nominating.Onto.Height
	case nominating != nil && nominated != nil:
		return tipToNominateOn.Height > nominating.Onto.Height &&
			tipToNominateOn.Height+1 > nominated.ConsensusValue.Nomination.Block_height
	}
	panic("unreachable")
}

func (b *Baton) nominate(ctx context.Context, tip proto.DbTip) (*proto.ConsensusValue, proto.BlockID, error) {

	schedNomResult, err := b.handler.Schedulercoord_nominate(ctx, tip)
	// we cannot continue if nomination fails
	must.NoError(ctx, err)
	must.Assert(ctx, schedNomResult.Outcome == proto.SchedulerNominationOK)

	schedNom := *schedNomResult.Scheduler_nomination()

	nom := &schedNom.Nomination
	nom.Proposer = b.handler.Signer() // XXX: Proposer should move to ConsensusValue

	cv := &proto.ConsensusValue{
		Timestamp:  protoutil.TimepointFromTime(b.handler.Now()),
		Nomination: *nom,
		Upgrades:   nil, // XXX: placeholder for upgrades
	}

	b.handler.Emit(ctx, batonkit.EventNominating{
		Onto:   tip,
		NumTx:  len(nom.Transaction_hashes),
		Height: nom.Block_height,
		Hash:   nom.Block_hash,
		//
		DbgConsensusValue: cv,
	})

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{Block_hash: nom.Prev_block_hash}
	if ok := b.handler.Nominate(ctx, scpapi.Slot(nom.Block_height), cv, &prevSummary); ok {
		b.handler.Emit(ctx, batonkit.EventNominated{
			Onto:   tip,
			NumTx:  len(nom.Transaction_hashes),
			Height: nom.Block_height,
			Hash:   nom.Block_hash,
			//
			DbgConsensusValue: cv,
		})
	} else {
		liblog.Infof("nomination rejected by scp, onto tip %v", protoutil.DbTipString(tip))
	}

	return cv, schedNom.Block_id, nil
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {

	if b.nominating == nil || b.nominating.Onto.Height+1 > height {
		return
	}

	b.nominating.Cancel()
	b.nominating = nil
}
