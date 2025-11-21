package baton

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	OntoDb  proto.DbTip
	OntoExt externalizedConsensus
	Cancel  context.CancelFunc
}

type nominated struct {
	time.Time
	Rejected       *proto.BlockHeight    // set if scp rejected nomination
	ConsensusValue *proto.ConsensusValue // set if scp accepted nomination
}

func (x *nominated) BlockHeight() proto.BlockHeight {
	if x.Rejected != nil {
		return *x.Rejected
	} else {
		return x.ConsensusValue.Nomination.Block_height
	}
}

func (b *Baton) canNominate(tipToNominateOn *proto.DbTip, nominating *nominating, nominated *nominated) bool {
	if tipToNominateOn == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return tipToNominateOn.Height+1 > nominated.BlockHeight()
	case nominating != nil && nominated == nil:
		return tipToNominateOn.Height > nominating.OntoDb.Height
	case nominating != nil && nominated != nil:
		return tipToNominateOn.Height > nominating.OntoDb.Height &&
			tipToNominateOn.Height+1 > nominated.BlockHeight()
	}
	panic("unreachable")
}

var ErrNominationRejected = fmt.Errorf("nomination rejected by scp")

func (b *Baton) nominate(
	ctx context.Context,
	ontoDb proto.DbTip,
	ontoExt externalizedConsensus,
) (*proto.ConsensusValue, proto.BlockID, error) {

	outcome, err := b.handler.Schedulercoord_nominate(ctx, ontoDb)
	must.NoError(ctx, err)
	must.Assert(ctx, outcome == proto.SchedulerNominationOK)

	nom, nomBlockID, err := b.handler.WaitForNomination(ctx, ontoDb.Height+1)
	if err != nil {
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		return nil, 0, err
	}

	nom.Proposer = b.handler.Signer() // XXX: Proposer should move to ConsensusValue

	cv := &proto.ConsensusValue{
		Timestamp:    protoutil.TimepointFromTime(b.handler.Now()),
		Nomination:   nom,
		Upgrades:     nil, // XXX: placeholder for upgrades
		Prev_cv_hash: ontoExt.ConsensusValueHash(),
	}

	cvc := b.cueFromConsensusValue(cv)

	b.handler.Emit(ctx, batonkit.EventNominating{
		Onto:   ontoDb,
		NumTx:  len(nom.Transaction_hashes),
		Height: nom.Block_height,
		Hash:   nom.Block_hash,
		//
		DbgConsensusValue:    cv,
		DbgConsensusValueCue: cvc,
	})

	// in a single-node setup, the externalization resulting from calling scp.nominate (handler.Nominate below)
	// may come before the dissemination of the cv (DisseminateCV below), in which case the local node will
	// not be able to externalize the value (and look for catchup when no other nodes exist).
	// to fix this, we populate the cv in the local pool optimistically.
	b.handler.ConsensusValuePoolAdd(ctx, *cv)

	// similarly, save the validated nomination (needed for commit) in the event that
	// an externalization occurs before the call to scp nominate returns.
	b.lock()
	b.setValidated(cv, nomBlockID)
	b.unlock()

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{Block_hash: nom.Prev_block_hash}
	if ok := b.handler.Nominate(ctx, scpkit.Slot(nom.Block_height), cvc, &prevSummary); ok {

		// if nomination accepted, broadcast consensus value
		liblog.Debugf("disseminating consensus value for slot %d with hash %s",
			nom.Block_height,
			form.Display(cvc.Cv_hash))
		b.handler.DisseminateCV(ctx, *cv)

		b.handler.Emit(ctx, batonkit.EventNominated{
			Onto:   ontoDb,
			NumTx:  len(nom.Transaction_hashes),
			Height: nom.Block_height,
			Hash:   nom.Block_hash,
			//
			DbgConsensusValue:    cv,
			DbgConsensusValueCue: cvc,
		})
		return cv, nomBlockID, nil
	} else {
		liblog.Infof("nomination rejected by scp, onto tip %v", protoutil.DbTipString(ontoDb))
		return nil, 0, ErrNominationRejected
	}
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {

	if b.nominating == nil || b.nominating.OntoDb.Height+1 > height {
		return
	}

	b.nominating.Cancel()
	b.nominating = nil
}
