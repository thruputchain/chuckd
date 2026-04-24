package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/sub/storage"
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
	*proto.ConsensusValue
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
		if err := libxdr.UnmarshalXdr(&xmsg, payload); err != nil {
			liblog.Errorf(ctx, "cannot unmarshal multiplex message: %v", err)
			continue
		}

		switch xmsg.Type {
		case proto.MultiplexMessageSubmittedTxs:
			x.receivedSubmittedTxs <- submittedTxsFrom{From: fromID, SubmittedTransactions: xmsg.Submitted()}
		case proto.MultiplexMessageConsensusValue:
			x.receivedConsensusValue <- consensusValueFrom{From: fromID, ConsensusValue: xmsg.Consensus_value()}
		case proto.MultiplexMessageSCP:
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
		protoutil.VIDString(x.self), xdrutil.SizeOfBatch(submitted.Transactions))
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSubmittedTxs
	*xmsg.Submitted() = *submitted
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf(ctx, "multiplex send submitted txs: %v", err)
	}
}

func (x *Multiplex) SendConsensusValue(ctx context.Context, cv proto.ConsensusValue) error {
	liblog.Debugf(ctx, "%v multiplex sending consensus value, for slot %v",
		protoutil.VIDString(x.self), cv.Nomination.Block_height)
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageConsensusValue
	*xmsg.Consensus_value() = cv
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf(ctx, "multiplex send consensus value: %v", err)
	}
	return nil
}

func (x *Multiplex) SendSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	liblog.Debugf(ctx, "%v multiplex sending envelope: %v",
		protoutil.VIDString(x.self),
		protoutil.SCPEnvelopeShortString(env))
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
	// persist the envelopes originating from our scp
	x.storage.PersistSCPEnvelope(ctx, env)
	return x.SendSCPEnvelope(ctx, env)
}

func (x *Multiplex) ReceiveSubmittedTxs(ctx context.Context) (proto.ValidatorID, *proto.SubmittedTransactions, bool) {
	select {
	case <-ctx.Done():
		return proto.ValidatorID{}, nil, false
	case submittedTxsFrom := <-x.receivedSubmittedTxs:
		liblog.Debugf(ctx, "%v multiplex received txs %d from %v",
			protoutil.VIDString(x.self),
			xdrutil.SizeOfBatch(submittedTxsFrom.Transactions),
			protoutil.VIDString(submittedTxsFrom.From))
		return submittedTxsFrom.From, submittedTxsFrom.SubmittedTransactions, true
	}
}

func (x *Multiplex) ReceiveConsensusValue(ctx context.Context) (proto.ValidatorID, *proto.ConsensusValue, bool) {
	select {
	case <-ctx.Done():
		return proto.ValidatorID{}, nil, false
	case cvFrom := <-x.receivedConsensusValue:
		liblog.Debugf(ctx, "%v multiplex received consensus value from %v, for slot %v",
			protoutil.VIDString(x.self),
			protoutil.VIDString(cvFrom.From),
			cvFrom.ConsensusValue.Nomination.Block_height)
		return cvFrom.From, cvFrom.ConsensusValue, true
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
			protoutil.SCPEnvelopeShortString(*envelopeFrom.SCPEnvelope))
		return envelopeFrom.From, envelopeFrom.SCPEnvelope, true
	}
}
