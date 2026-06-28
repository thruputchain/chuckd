package baton

import (
	"context"
	"errors"
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
	OntoExt externalizedBasis
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
	ontoExt externalizedBasis,
) (*proto.ConsensusBasis, *proto.ConsensusValue, proto.BlockID, error) {

	b.stampNominateStart(ctx, ontoDb.Height+1)
	defer b.stampNominateEnd(ontoDb.Height + 1)

	// --- prepare nomination ---

	t0 := b.handler.Now()
	result, err := b.handler.Schedulercoord_nominate(ctx, ontoDb)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, nil, 0, err
	}
	must.NoError(ctx, err)
	must.Assert(ctx, result.Outcome == proto.SchedulerNominationOK)

	nominatedBlock := result.Nominated_block()
	nomination := nominatedBlock.Nomination
	numTxs := xdrutil.TokenSizeOfBatch(nomination.Transactions_hash)

	tMempoolNominateEnd := b.handler.Now()
	b.stampMempoolNominateEnd(nominatedBlock.Nomination.Block_height, tMempoolNominateEnd)
	liblog.Debugf(ctx, "mempool_nomination_delay=%v | height=%d, num_txs=%d",
		tMempoolNominateEnd.Sub(t0),
		nominatedBlock.Nomination.Block_height,
		numTxs,
	)

	// --- fire nomination ---

	t1 := b.handler.Now()

	// wait until `now - prevStamp >= 2*clock skew`
	prevStamp := protoutil.TimeFromTimepoint(ontoExt.basis.Timestamp)
	stamp := b.handler.Now()
	for stamp.Sub(prevStamp) < 2*b.skew {
		liblog.Debugf(ctx, "delaying block %d by %v to overcome clock skew", nomination.Block_height, b.skew)
		b.handler.Sleep(b.skew)
		stamp = b.handler.Now()
	}

	basis := &proto.ConsensusBasis{
		Timestamp:       protoutil.TimepointFromTime(stamp.Add(-b.skew)),
		Proposer:        protoutil.PublicVID(b.handler.Signer()),
		Nomination:      nomination,
		Upgrades:        nil, // XXX: placeholder for upgrades
		Prev_value_hash: protoutil.HashValue(ontoExt.ConsensusValue()).Bytes(),
		Prev_basis_hash: ontoExt.ConsensusBasisHash(),
	}
	cb := &proto.CatchupBlock{
		Consensus_basis:     *basis,
		Signed_transactions: nominatedBlock.Txs,
	}

	q0 := b.handler.Now()
	value := protoutil.BasisValue(basis)
	liblog.Debugf(ctx, "hashing basis for block %d took %v", nomination.Block_height, b.handler.Now().Sub(q0))

	evNominating := batonkit.EventNominating{
		Onto:           ontoDb,
		NumTx:          numTxs,
		Height:         nomination.Block_height,
		BlockHash:      nomination.Block_hash,
		ConsensusBasis: basis,
		ConsensusValue: value,
	}
	liblog.Infof(ctx, "%v", evNominating)
	b.handler.Emit(ctx, evNominating)

	// in a single-node setup, the externalization resulting from calling scp.nominate (handler.Nominate below)
	// may come before the dissemination of the basis (DisseminateBasis below), in which case the local node will
	// not be able to externalize the value (and look for catchup when no other nodes exist).
	// to fix this, we populate the basis in the local pool optimistically.
	b.handler.CVPoolAdd(ctx, *basis, *value)

	// kick-off an optimistic pre-persist of the catchup block;
	// this will be committed/persisted by commit if the block is committed.
	b.handler.CBPoolAdd(ctx, *cb, *value)

	// catchup block must be pre-persisted before we nominate it to SCP, because
	// if the SCP message goes out before the pre-persist is complete, and the node dies
	// it won't be able to server missing transaction requests.
	b.handler.CBPoolWaitPrePersist(ctx, *value)

	// similarly, save the validated nomination (needed for commit) in the event that
	// an externalization occurs before the call to scp nominate returns.
	b.lk.Lock()
	b.setValidated(basis, value, nominatedBlock.Block_id)
	b.lk.Unlock()

	// notify the repair service that we are making the nomination, so that it
	// honors requests for missing transactions
	b.handler.AnnounceNomination(value)

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{Block_hash: nomination.Prev_block_hash}
	n0 := time.Now()
	accepted := b.handler.SubmitNomination(ctx, scpkit.Slot(nomination.Block_height), value, &prevSummary)
	b.reportSCPNominationDelay(ctx, nomination.Block_height, time.Since(n0))

	if accepted {

		// if nomination accepted, broadcast consensus basis
		liblog.Debugf(ctx, "disseminating consensus basis for slot %d with hash %s",
			nomination.Block_height,
			libxdr.HashString(value.Basis_hash))
		b.handler.DisseminateBasis(ctx, *basis, *value)

		evNominated := batonkit.EventNominated{
			Onto:           ontoDb,
			NumTx:          numTxs,
			Height:         nomination.Block_height,
			BlockHash:      nomination.Block_hash,
			ConsensusBasis: basis,
			ConsensusValue: value,
		}
		liblog.Infof(ctx, "%v", evNominated)
		b.handler.Emit(ctx, evNominated)

		liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=accepted, height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), nomination.Block_height, numTxs)

		return basis, value, nominatedBlock.Block_id, nil
	} else {
		liblog.Errorf(ctx, "nomination rejected by scp, onto tip %v", protoutil.DbTipString(ontoDb))

		liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=rejected height=%d, num_txs=%d",
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
