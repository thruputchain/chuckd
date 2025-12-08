package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtext"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// scp msgs, proposed txsets,
type Multiplex struct {
	self                   proto.ValidatorID
	m                      libxdr.XdrMarshaler
	dbgM                   libxdr.XdrMarshaler
	dogwood                *dogwood.DogwoodService
	receivedSubmittedTxs   chan submittedTxsFrom
	receivedConsensusValue chan consensusValueFrom
	receivedEnvelope       chan envelopeFrom
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

func NewMultiplex(ctx context.Context, self proto.ValidatorID, dogwood *dogwood.DogwoodService) *Multiplex {
	x := &Multiplex{
		self:                   self,
		m:                      libxdr.NewXdrMarshaler(),
		dbgM:                   libxdr.NewXdrMarshaler(),
		dogwood:                dogwood,
		receivedSubmittedTxs:   make(chan submittedTxsFrom),
		receivedConsensusValue: make(chan consensusValueFrom),
		receivedEnvelope:       make(chan envelopeFrom),
	}
	go x.multiplex(ctx)
	return x
}

func (x *Multiplex) multiplex(ctx context.Context) {
	for {
		payload, fromID, err := x.dogwood.Receive(ctx)
		if err != nil {
			liblog.Errorf("multiplex receive: %v", err)
			continue
		}
		var xmsg proto.MultiplexMessage
		if err := libxdr.UnmarshalXdr(&xmsg, payload); err != nil {
			liblog.Errorf("cannot unmarshal multiplex message: %v", err)
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
			liblog.Errorf("unsupported multiplex message type %v", xmsg.Type)
		}
	}
}

func (x *Multiplex) SendSubmittedTxs(ctx context.Context, submitted *proto.SubmittedTransactions) {
	liblog.Debugf("%v multiplex sending txs: %v",
		protoutil.VIDString(x.self),
		libtext.Display(protoutil.HashSignedTxs(x.dbgM, submitted.Transactions)))
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSubmittedTxs
	*xmsg.Submitted() = *submitted
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg).Bytes())
	if err != nil {
		liblog.Errorf("multiplex send submitted txs: %v", err)
	}
}

func (x *Multiplex) SendConsensusValue(ctx context.Context, cv proto.ConsensusValue) error {
	liblog.Debugf("%v multiplex sending consensus value, for slot %v",
		protoutil.VIDString(x.self), cv.Nomination.Block_height)
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageConsensusValue
	*xmsg.Consensus_value() = cv
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg).Bytes())
	if err != nil {
		liblog.Errorf("multiplex send consensus value: %v", err)
	}
	return nil
}

func (x *Multiplex) SendSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	liblog.Debugf("%v multiplex sending envelope: %v",
		protoutil.VIDString(x.self),
		protoutil.SCPEnvelopeShortString(env))
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSCP
	*xmsg.Envelope() = env
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg).Bytes())
	if err != nil {
		liblog.Errorf("multiplex send envelope: %v", err)
	}
	return nil
}

func (x *Multiplex) EmitEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	return x.SendSCPEnvelope(ctx, env)
}

func (x *Multiplex) ReceiveSubmittedTxs(ctx context.Context) (proto.ValidatorID, *proto.SubmittedTransactions) {
	submittedTxsFrom := <-x.receivedSubmittedTxs
	liblog.Debugf("%v multiplex received txs %d from %v",
		protoutil.VIDString(x.self),
		len(submittedTxsFrom.Transactions),
		protoutil.VIDString(submittedTxsFrom.From))
	return submittedTxsFrom.From, submittedTxsFrom.SubmittedTransactions
}

func (x *Multiplex) ReceiveConsensusValue(ctx context.Context) (proto.ValidatorID, *proto.ConsensusValue) {
	cvFrom := <-x.receivedConsensusValue
	liblog.Debugf("%v multiplex received consensus value from %v, for slot %v",
		protoutil.VIDString(x.self),
		protoutil.VIDString(cvFrom.From),
		cvFrom.ConsensusValue.Nomination.Block_height)
	return cvFrom.From, cvFrom.ConsensusValue
}

func (x *Multiplex) ReceiveEnvelope(ctx context.Context) (proto.ValidatorID, *proto.SCPEnvelope) {
	envelopeFrom := <-x.receivedEnvelope
	liblog.Debugf("%v multiplex received envelope from %v: %v",
		protoutil.VIDString(x.self),
		protoutil.VIDString(envelopeFrom.From),
		protoutil.SCPEnvelopeShortString(*envelopeFrom.SCPEnvelope))
	return envelopeFrom.From, envelopeFrom.SCPEnvelope
}
