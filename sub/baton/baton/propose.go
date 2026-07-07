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
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type proposing struct {
	OntoDb proto.DbTip
	Cancel context.CancelFunc
}

func (b *Baton) canPropose(tipToProposeOnto *proto.DbTip, proposing *proposing) bool {
	if tipToProposeOnto == nil {
		return false
	}
	if proposing == nil {
		return true
	} else {
		return protoutil.ValueHeight(&tipToProposeOnto.Value) > protoutil.ValueHeight(&proposing.OntoDb.Value)
	}
}

var ErrProposalRejected = fmt.Errorf("proposal rejected by scp")

func (b *Baton) propose(
	ctx context.Context,
	ontoDb proto.DbTip,
) (*proto.ConsensusBasis, *proto.ConsensusValue, proto.BlockID, error) {

	ontoHeight := protoutil.ValueHeight(&ontoDb.Value)

	b.stampNominateStart(ctx, ontoHeight+1)
	defer b.stampNominateEnd(ontoHeight + 1)

	// --- pick the block timepoint ---

	// wait until `now - prevStamp >= 2*clock skew`
	prevStamp := protoutil.TimeFromTimepoint(protoutil.ValueTimepoint(&ontoDb.Value))
	stamp := b.handler.Now()
	for stamp.Sub(prevStamp) < 2*b.skew {
		liblog.Debugf(ctx, "delaying block %d by %v to overcome clock skew", ontoHeight+1, b.skew)
		b.handler.Sleep(b.skew)
		stamp = b.handler.Now()
	}

	// --- prepare proposal ---

	propCx := proto.ProposalContext{
		Proposer:        protoutil.PublicVID(b.handler.Signer()),
		Block_timepoint: protoutil.TimepointFromTime(stamp.Add(-b.skew)),
	}

	t0 := b.handler.Now()
	result, err := b.handler.Schedulercoord_propose_block(ctx, ontoDb.Id, propCx)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, nil, 0, err
	}
	must.NoError(ctx, err)
	must.Assert(ctx, result.Outcome == proto.SchedulerProposalOK)

	proposedBlock := result.Proposed_block()
	proposal := proposedBlock.Proposal
	height := proposal.Header.Block_height
	numTxs := xdrutil.TokenSizeOfBatch(proposal.Transactions.Hash)

	tMempoolProposeEnd := b.handler.Now()
	b.stampMempoolNominateEnd(height, tMempoolProposeEnd)
	liblog.Debugf(ctx, "mempool_proposal_delay=%v | height=%d, num_txs=%d",
		tMempoolProposeEnd.Sub(t0),
		height,
		numTxs,
	)

	// --- fire proposal ---

	t1 := b.handler.Now()

	q0 := b.handler.Now()
	value := &proto.ConsensusValue{
		Prev_hash:   protoutil.HashValueToProto(&ontoDb.Value),
		Proposal_header:   proposal.Header,
		Proposal_txs_hash: protoutil.HashProposalTxsFast(&proposal.Transactions),
		Upgrades:          nil, // XXX: placeholder for upgrades
	}
	basis := &proto.ConsensusBasis{
		Value:        *value,
		Proposal_txs: proposal.Transactions,
	}
	cb := &proto.CatchupBlock{
		Consensus_basis:     *basis,
		Signed_transactions: proposedBlock.Txs,
	}
	liblog.Debugf(ctx, "hashing proposal txs for block %d took %v", height, b.handler.Now().Sub(q0))

	evNominating := batonkit.EventNominating{
		Onto:           ontoDb,
		NumTx:          numTxs,
		Height:         height,
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
	b.setValidated(basis, value, proposedBlock.Block_id)
	b.lk.Unlock()

	// notify the repair service that we are making the nomination, so that it
	// honors requests for missing transactions
	b.handler.AnnounceNomination(value)

	// nominate to consensus
	prevSummary := proto.PrevProposalSummary{Value_hash: value.Prev_hash}
	n0 := time.Now()
	accepted := b.handler.SubmitNomination(ctx, scpkit.Slot(height), value, &prevSummary)
	b.reportSCPNominationDelay(ctx, height, time.Since(n0))

	if accepted {

		// if nomination accepted, broadcast consensus basis
		liblog.Debugf(ctx, "disseminating consensus basis for slot %d with value hash %s",
			height,
			protoutil.ValueDisplay(value))
		b.handler.DisseminateBasis(ctx, *basis, *value)

		evNominated := batonkit.EventNominated{
			Onto:           ontoDb,
			NumTx:          numTxs,
			Height:         height,
			ConsensusBasis: basis,
			ConsensusValue: value,
		}
		liblog.Infof(ctx, "%v", evNominated)
		b.handler.Emit(ctx, evNominated)

		liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=accepted, height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), height, numTxs)

		return basis, value, proposedBlock.Block_id, nil
	} else {
		liblog.Errorf(ctx, "nomination rejected by scp, onto tip %v", protoutil.DbTipString(ontoDb))

		liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=rejected height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), height, numTxs)

		return nil, nil, 0, ErrProposalRejected
	}
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {

	if b.nominating == nil || protoutil.ValueHeight(&b.nominating.OntoDb.Value)+1 > height {
		return
	}

	b.nominating.Cancel()
	b.nominating = nil
}
