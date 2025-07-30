package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/net/spectra"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// scp msgs, proposed txsets,
type Multiplex struct {
	m         xdrutil.XdrMarshaler
	transport *spectra.SpectraTransport
	submitted chan submittedTxsFrom
}

type submittedTxsFrom struct {
	From proto.ValidatorID
	*proto.SubmittedTransactions
}

func NewMultiplex(ctx context.Context, s *spectra.SpectraTransport) *Multiplex {
	x := &Multiplex{
		m:         xdrutil.NewXdrMarshaler(),
		transport: s,
		submitted: make(chan submittedTxsFrom),
	}
	go x.multiplex(ctx)
	return x
}

func (x *Multiplex) multiplex(ctx context.Context) {
	for {
		fromID, _, payload, err := x.transport.BulkReceive(ctx)
		if err != nil {
			liblog.Errorf("multiplex bulk receive: %v", err)
			continue
		}
		var xmsg proto.MultiplexMessage
		if err := xdrutil.UnmarshalXdr(&xmsg, payload); err != nil {
			liblog.Errorf("cannot unmarshal multiplex message: %v", err)
			continue
		}

		switch xmsg.Type {
		case proto.MultiplexMessageSubmittedTxs:
			x.submitted <- submittedTxsFrom{From: fromID, SubmittedTransactions: xmsg.Submitted()}
		default:
			liblog.Errorf("unsupported multiplex message type %v", xmsg.Type)
		}
	}
}

func (x *Multiplex) SendSubmittedTxs(ctx context.Context, submitted *proto.SubmittedTransactions) {
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageSubmittedTxs
	*xmsg.Submitted() = *submitted
	_, err := x.transport.BulkSend(ctx, x.m.MarshalXdr(&xmsg).Bytes())
	if err != nil {
		liblog.Errorf("multiplex bulk send: %v", err)
	}
}

func (x *Multiplex) Disseminate(ctx context.Context, submitted *proto.SubmittedTransactions) {
	x.SendSubmittedTxs(ctx, submitted)
}

func (x *Multiplex) ReceiveSubmittedTxs(ctx context.Context) (proto.ValidatorID, *proto.SubmittedTransactions) {
	submittedTxsFrom := <-x.submitted
	return submittedTxsFrom.From, submittedTxsFrom.SubmittedTransactions
}
