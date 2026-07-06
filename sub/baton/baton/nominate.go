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
	}
	return protoutil.TipHeight(*tipToNominateOn) > protoutil.TipHeight(nominating.OntoDb)
}

var ErrNominationRejected = fmt.Errorf("nomination rejected by scp")

func (b *Baton) nominate(
	ctx context.Context,
	ontoDb proto.DbTip,
	ontoExt externalizedBasis,
) (*proto.ConsensusBasis, *proto.ConsensusValue, proto.BlockID, error) {

	height := protoutil.TipHeight(ontoDb) + 1
	b.stampNominateStart(ctx, height)
	defer b.stampNominateEnd(height)

	t0 := b.handler.Now()

	// wait until `now - prevStamp >= 2*clock skew`
	prevStamp := protoutil.TimeFromTimepoint(ontoExt.ConsensusValue().Proposal_header.Timestamp)
	stamp := b.handler.Now()
	for stamp.Sub(prevStamp) < 2*b.skew {
		liblog.Debugf(ctx, "delaying block %d by %v to overcome clock skew", height, b.skew)
		b.handler.Sleep(b.skew)
		stamp = b.handler.Now()
	}
	timestamp := protoutil.TimepointFromTime(stamp.Add(-b.skew))

	result, err := b.handler.Schedulercoord_propose_block(ctx, ontoDb.Id, timestamp)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, nil, 0, err
	}
	must.NoError(ctx, err)
	must.Assert(ctx, result.Outcome == proto.SchedulerProposalOK)

	proposedBlock := result.Nominated_block()
	proposal := proposedBlock.Proposal
	numTxs := xdrutil.TokenSizeOfBatch(proposal.Transactions.Hash)

	tMempoolNominateEnd := b.handler.Now()
	b.stampMempoolNominateEnd(height, tMempoolNominateEnd)
	liblog.Debugf(ctx, "mempool_nomination_delay=%v | height=%d, num_txs=%d",
		tMempoolNominateEnd.Sub(t0), height, numTxs)

	t1 := b.handler.Now()

	value := &proto.ConsensusValue{
		Proposer:        protoutil.PublicVID(b.handler.Signer()),
		Upgrades:        nil, // XXX: placeholder for upgrades
		Proposal_header: proposal.Header,
		Proposal_hash:   protoutil.HashProposalFast(&proposal),
		Prev_value_hash: protoutil.HashValueToProto(ontoExt.ConsensusValue()),
	}
	basis := &proto.ConsensusBasis{
		Value:        *value,
		Proposal_txs: proposal.Transactions,
	}
	cb := &proto.CatchupBlock{
		Consensus_basis:     *basis,
		Signed_transactions: proposedBlock.Txs,
	}

	evNominating := batonkit.EventNominating{
		Onto:           ontoDb,
		NumTx:          numTxs,
		Height:         height,
		ValueHash:      protoutil.HashValueToProto(value),
		ConsensusBasis: basis,
		ConsensusValue: value,
	}
	liblog.Infof(ctx, "%v", evNominating)
	b.handler.Emit(ctx, evNominating)

	b.handler.CVPoolAdd(ctx, *basis, *value)
	b.handler.CBPoolAdd(ctx, *cb, *value)
	b.handler.CBPoolWaitPrePersist(ctx, *value)

	b.lk.Lock()
	b.setValidated(basis, value, proposedBlock.Block_id)
	b.lk.Unlock()

	b.handler.AnnounceNomination(value)

	prevSummary := proto.PrevProposalSummary{Value_hash: value.Prev_value_hash}
	n0 := time.Now()
	accepted := b.handler.SubmitNomination(ctx, scpkit.Slot(height), value, &prevSummary)
	b.reportSCPNominationDelay(ctx, height, time.Since(n0))

	if accepted {
		liblog.Debugf(ctx, "disseminating consensus basis for slot %d with value %s",
			height, protoutil.ValueDisplay(value))
		b.handler.DisseminateBasis(ctx, *basis, *value)

		evNominated := batonkit.EventNominated{
			Onto:           ontoDb,
			NumTx:          numTxs,
			Height:         height,
			ValueHash:      protoutil.HashValueToProto(value),
			ConsensusBasis: basis,
			ConsensusValue: value,
		}
		liblog.Infof(ctx, "%v", evNominated)
		b.handler.Emit(ctx, evNominated)

		liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=accepted, height=%d, num_txs=%d",
			b.handler.Now().Sub(t1), height, numTxs)

		return basis, value, proposedBlock.Block_id, nil
	}

	liblog.Errorf(ctx, "nomination rejected by scp, onto tip %v", protoutil.DbTipString(ontoDb))
	liblog.Debugf(ctx, "coordinator_nomination_delay=%v | result=rejected height=%d, num_txs=%d",
		b.handler.Now().Sub(t1), height, numTxs)
	return nil, nil, 0, ErrNominationRejected
}

func (b *Baton) cancelNominatingNotAbove(_ context.Context, height proto.BlockHeight) {
	if b.nominating == nil || protoutil.TipHeight(b.nominating.OntoDb)+1 > height {
		return
	}
	b.nominating.Cancel()
	b.nominating = nil
}
