package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	Onto   batonapi.Tip
	Cancel context.CancelFunc
}

type nominated struct {
	time.Time
	Nomination *proto.Nomination
}

func (b *Baton) canNominate(tipToNominateOn *batonapi.Tip, nominating *nominating, nominated *nominated) bool {
	if tipToNominateOn == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return tipToNominateOn.Height+1 > nominated.Nomination.Block_height
	case nominating != nil && nominated == nil:
		return tipToNominateOn.Height > nominating.Onto.Height
	case nominating != nil && nominated != nil:
		return tipToNominateOn.Height > nominating.Onto.Height &&
			tipToNominateOn.Height+1 > nominated.Nomination.Block_height
	}
	panic("unreachable")
}

func (b *Baton) nominate(ctx context.Context, tip batonapi.Tip) (*proto.Nomination, proto.BlockID, error) {

	schedNomResult, err := b.handler.Schedulercoord_nominate(ctx, tip.Proto())
	if err != nil {
		return nil, 0, err
	}
	if schedNomResult.Outcome != proto.SchedulerNominationOK {
		return nil, 0, fmt.Errorf("scheduler nomination error")
	}
	schedNom := *schedNomResult.Scheduler_nomination()

	nom := &schedNom.Nomination
	nom.Proposer = b.handler.Signer()

	b.handler.Emit(ctx, batonapi.EventNominating{
		Onto:   tip,
		NumTx:  len(nom.Transaction_hashes),
		Height: nom.Block_height,
		Hash:   nom.Block_hash,
	})

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{Block_hash: nom.Prev_block_hash}
	if ok := b.handler.Nominate(ctx, scpapi.Slot(nom.Block_height), nom, &prevSummary); ok {
		b.handler.Emit(ctx, batonapi.EventNominated{
			Onto:   tip,
			NumTx:  len(nom.Transaction_hashes),
			Height: nom.Block_height,
			Hash:   nom.Block_hash,
		})
	} else {
		liblog.Infof("nomination rejected, onto tip %v", tip.String())
	}

	return nom, schedNom.Block_id, nil
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {

	if b.nominating == nil || b.nominating.Onto.Height+1 > height {
		return
	}

	b.nominating.Cancel()
	b.nominating = nil
}
