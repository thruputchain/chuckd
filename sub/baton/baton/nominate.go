package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	OntoDb  proto.DbTip
	OntoExt externalizedConsensus
	Cancel  context.CancelFunc
}

func (b *Baton) canNominate(tipToNominateOn *proto.DbTip, nominating *nominating) bool {
	if tipToNominateOn == nil {
		return false
	}
	if nominating == nil {
		return true
	} else {
		return tipToNominateOn.Height > nominating.OntoDb.Height
	}
}

var ErrNominationRejected = fmt.Errorf("nomination rejected by scp")

func (b *Baton) nominate(
	ctx context.Context,
	ontoDb proto.DbTip,
	ontoExt externalizedConsensus,
) (*proto.ConsensusValue, *proto.ConsensusValueCue, proto.BlockID, error) {

	b.stampNominateStart(ontoDb.Height + 1)
	defer b.stampNominateEnd(ontoDb.Height + 1)

	// --- prepare nomination ---

	t0 := b.handler.Now()
	result, err := b.handler.Schedulercoord_nominate(ctx, ontoDb)
	must.NoError(ctx, err)
	must.Assert(ctx, result.Outcome == proto.SchedulerNominationOK)

	nominatedBlock := result.Nominated_block()
	nomination := nominatedBlock.Nomination
	numTxs := xdrutil.SizeOfBatch(nomination.Transactions_hash)

	tMempoolNominateEnd := b.handler.Now()
	b.stampMempoolNominateEnd(nominatedBlock.Nomination.Block_height, tMempoolNominateEnd)
	liblog.Debugf("mempool_nomination_delay=%v | height=%d, num_txs=%d",
		tMempoolNominateEnd.Sub(t0),
		nominatedBlock.Nomination.Block_height,
		numTxs,
	)

	// --- fire nomination ---

	t1 := b.handler.Now()

	nomination.Proposer = b.handler.Signer() // XXX: Proposer should move to ConsensusValue

	// wait until `now - prevStamp >= 2*clock skew`
	prevStamp := protoutil.TimeFromTimepoint(ontoExt.cv.Timestamp)
	stamp := b.handler.Now()
	for stamp.Sub(prevStamp) < 2*b.skew {
		liblog.Debugf("delaying block %d by %v to overcome clock skew", nomination.Block_height, b.skew)
		b.handler.Sleep(b.skew)
		stamp = b.handler.Now()
	}

	cv := &proto.ConsensusValue{
		Timestamp:    protoutil.TimepointFromTime(stamp.Add(-b.skew)),
		Nomination:   nomination,
		Upgrades:     nil, // XXX: placeholder for upgrades
		Prev_cv_hash: ontoExt.ConsensusValueHash(),
	}
	cb := &proto.CatchupBlock{
		Consensus_value:     *cv,
		Signed_transactions: nominatedBlock.Txs,
	}

	q0 := b.handler.Now()
	cvc := b.cueFromConsensusValue(cv)
	liblog.Debugf("hashing cv for block %d took %v", nomination.Block_height, b.handler.Now().Sub(q0))

	evNominating := batonkit.EventNominating{
		Onto:                 ontoDb,
		NumTx:                numTxs,
		Height:               nomination.Block_height,
		BlockHash:            nomination.Block_hash,
		DbgConsensusValue:    cv,
		DbgConsensusValueCue: cvc,
	}
	liblog.Infof("%v", evNominating)
	b.handler.Emit(ctx, evNominating)

	// in a single-node setup, the externalization resulting from calling scp.nominate (handler.Nominate below)
	// may come before the dissemination of the cv (DisseminateCV below), in which case the local node will
	// not be able to externalize the value (and look for catchup when no other nodes exist).
	// to fix this, we populate the cv in the local pool optimistically.
	b.handler.ConsensusValuePoolAdd(ctx, *cv, cvc.Cv_hash)

	// kick-off an optimistic pre-persist of the catchup block;
	// this will be committed/persisted by commit if the block is committed.
	b.handler.CatchupBlockPoolAdd(ctx, *cb, cvc.Cv_hash)

	// similarly, save the validated nomination (needed for commit) in the event that
	// an externalization occurs before the call to scp nominate returns.
	b.lock()
	b.setValidated(cv, cvc.Cv_hash, nominatedBlock.Block_id)
	b.unlock()

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{Block_hash: nomination.Prev_block_hash}
	n0 := time.Now()
	accepted := b.handler.Nominate(ctx, scpkit.Slot(nomination.Block_height), cvc, &prevSummary)
	b.reportSCPNominationDelay(nomination.Block_height, time.Since(n0))

	if accepted {

		// if nomination accepted, broadcast consensus value
		liblog.Debugf("disseminating consensus value for slot %d with hash %s",
			nomination.Block_height,
			libxdr.HashString(cvc.Cv_hash))
		b.handler.DisseminateCV(ctx, *cv, cvc.Cv_hash)

		evNominated := batonkit.EventNominated{
			Onto:                 ontoDb,
			NumTx:                numTxs,
			Height:               nomination.Block_height,
			BlockHash:            nomination.Block_hash,
			DbgConsensusValue:    cv,
			DbgConsensusValueCue: cvc,
		}
		liblog.Infof("%v", evNominated)
		b.handler.Emit(ctx, evNominated)

		liblog.Debugf("coordinator_nomination_delay=%v | result=accepted, height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), nomination.Block_height, numTxs)

		return cv, cvc, nominatedBlock.Block_id, nil
	} else {
		liblog.Infof("nomination rejected by scp, onto tip %v", protoutil.DbTipString(ontoDb))

		liblog.Debugf("coordinator_nomination_delay=%v | result=rejected height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), nomination.Block_height, numTxs)

		return nil, nil, 0, ErrNominationRejected
	}
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {

	if b.nominating == nil || b.nominating.OntoDb.Height+1 > height {
		return
	}

	b.nominating.Cancel()
	b.nominating = nil
}
