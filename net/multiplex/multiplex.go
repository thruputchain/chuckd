package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libflow"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// scp msgs, proposed txsets,
type Multiplex struct {
	self                   proto.ValidatorID
	m                      libxdr.XdrMarshaler
	dbgM                   libxdr.XdrMarshaler
	dogwood                *dogwood.DogwoodService
	storage                storage.SCPEnvelopePersistence
	receivedSubmittedTxs   chan submittedTxsFrom
	receivedConsensusValue chan consensusValueFrom
	receivedEnvelope       chan envelopeFrom
	sendVettedTxsWorker    libflow.SubmitWorkerPool
}

type submittedTxsFrom struct {
	From proto.ValidatorID
	*proto.SubmittedTransactions
}

type envelopeFrom struct {
	From proto.ValidatorID
	*proto.SCPEnvelope
}

type consensusValueFrom struct {
	From proto.ValidatorID
	*proto.ConsensusBasis
}

func NewMultiplex(
	ctx context.Context,
	self proto.ValidatorID,
	dogwood *dogwood.DogwoodService,
	storage storage.SCPEnvelopePersistence,
) *Multiplex {
	x := &Multiplex{
		self:                   self,
		m:                      libxdr.NewXdrMarshaler(),
		dbgM:                   libxdr.NewXdrMarshaler(),
		dogwood:                dogwood,
		storage:                storage,
		receivedSubmittedTxs:   make(chan submittedTxsFrom),
		receivedConsensusValue: make(chan consensusValueFrom),
		receivedEnvelope:       make(chan envelopeFrom),
		sendVettedTxsWorker:    libflow.NewSubmitWorkerPool(ctx, 100), //XXX: needs parameter control
	}
	go x.multiplex(ctx)
	return x
}

func (x *Multiplex) multiplex(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		payload, fromID, err := x.dogwood.Receive(ctx)
		if err != nil {
			liblog.Errorf(ctx, "multiplex receive: %v", err)
			continue
		}
		var xmsg proto.MultiplexMessage
		if err := libxdr.UnmarshalXdrEOF(&xmsg, payload); err != nil {
			liblog.Errorf(ctx, "cannot unmarshal multiplex message: %v", err)
			continue
		}

		switch xmsg.Type {

		case proto.MultiplexMessageSubmittedTxs:
			if len(payload) > systemfixed.MaxBytesTxsMessage {
				liblog.Errorf(ctx, "submitted txs message too large: %d bytes", len(payload))
				continue
			}

			x.receivedSubmittedTxs <- submittedTxsFrom{From: fromID, SubmittedTransactions: xmsg.Submitted()}

		case proto.MultiplexMessageConsensusBasis:
			if len(payload) > systemfixed.MaxBytesConsensusBasisMessage {
				liblog.Errorf(ctx, "consensus basis message too large: %d bytes", len(payload))
				continue
			}

			x.receivedConsensusValue <- consensusValueFrom{From: fromID, ConsensusBasis: xmsg.Basis()}

		case proto.MultiplexMessageSCP:
			if len(payload) > systemfixed.MaxBytesSCPEnvelopeMessage {
				liblog.Errorf(ctx, "scp envelope message too large: %d bytes", len(payload))
				continue
			}

			x.receivedEnvelope <- envelopeFrom{From: fromID, SCPEnvelope: xmsg.Envelope()}

		default:
			liblog.Errorf(ctx, "unsupported multiplex message type %v", xmsg.Type)
		}
	}
}

func (x *Multiplex) SendSubmittedTxs(ctx context.Context, submitted *proto.SubmittedTransactions) {
	ctx = context.WithoutCancel(ctx)
	x.sendVettedTxsWorker.Submit(
		func() {
			x.doSendSubmittedTxs(ctx, submitted)
		},
	)
}

func (x *Multiplex) doSendSubmittedTxs(ctx context.Context, submitted *proto.SubmittedTransactions) {
	liblog.Debugf(ctx, "%v multiplex sending %d txs",
		protoutil.VIDString(x.self), xdrutil.TokenSizeOfBatch(submitted.Transactions))
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSubmittedTxs
	*xmsg.Submitted() = *submitted
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf(ctx, "multiplex send submitted txs: %v", err)
	}
}

func (x *Multiplex) SendConsensusBasis(ctx context.Context, basis proto.ConsensusBasis) error {
	liblog.Debugf(ctx, "%v multiplex sending consensus basis, for slot %v",
		protoutil.VIDString(x.self), basis.Value.Proposal_header.Block_height)
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageConsensusBasis
	*xmsg.Basis() = basis
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf(ctx, "multiplex send consensus basis: %v", err)
	}
	return nil
}

func (x *Multiplex) SendSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	liblog.Debugf(ctx, "%v multiplex sending envelope: %v",
		protoutil.VIDString(x.self),
		protoutil.SCPEnvelopeDisplay(env))
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSCP
	*xmsg.Envelope() = env
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf(ctx, "multiplex send envelope: %v", err)
	}
	return nil
}

func (x *Multiplex) SCPEmitEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	x.storage.PersistSCPEnvelope(ctx, env)
	x.storage.PersistSCPEnvelopeFromLocal(ctx, env)
	return x.SendSCPEnvelope(ctx, env)
}

func (x *Multiplex) ReceiveSubmittedTxs(ctx context.Context) (proto.ValidatorID, *proto.SubmittedTransactions, bool) {
	select {
	case <-ctx.Done():
		return proto.ValidatorID{}, nil, false
	case submittedTxsFrom := <-x.receivedSubmittedTxs:
		liblog.Debugf(ctx, "%v multiplex received txs %d from %v",
			protoutil.VIDString(x.self),
			xdrutil.TokenSizeOfBatch(submittedTxsFrom.Transactions),
			protoutil.VIDString(submittedTxsFrom.From))
		return submittedTxsFrom.From, submittedTxsFrom.SubmittedTransactions, true
	}
}

func (x *Multiplex) ReceiveConsensusBasis(ctx context.Context) (proto.ValidatorID, *proto.ConsensusBasis, bool) {
	select {
	case <-ctx.Done():
		return proto.ValidatorID{}, nil, false
	case cvFrom := <-x.receivedConsensusValue:
		liblog.Debugf(ctx, "%v multiplex received consensus basis from %v, for slot %v",
			protoutil.VIDString(x.self),
			protoutil.VIDString(cvFrom.From),
			cvFrom.ConsensusBasis.Value.Proposal_header.Block_height)
		return cvFrom.From, cvFrom.ConsensusBasis, true
	}
}

func (x *Multiplex) ReceiveEnvelope(ctx context.Context) (proto.ValidatorID, *proto.SCPEnvelope, bool) {
	select {
	case <-ctx.Done():
		return proto.ValidatorID{}, nil, false
	case envelopeFrom := <-x.receivedEnvelope:
		liblog.Debugf(ctx, "%v multiplex received envelope from %v: %v",
			protoutil.VIDString(x.self),
			protoutil.VIDString(envelopeFrom.From),
			protoutil.SCPEnvelopeDisplay(*envelopeFrom.SCPEnvelope))
		return envelopeFrom.From, envelopeFrom.SCPEnvelope, true
	}
}
