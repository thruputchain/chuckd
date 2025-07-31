package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// scp msgs, proposed txsets,
type Multiplex struct {
	m         xdrutil.XdrMarshaler
	dogwood   *dogwood.DogwoodService
	submitted chan submittedTxsFrom
}

type submittedTxsFrom struct {
	From proto.ValidatorID
	*proto.SubmittedTransactions
}

func NewMultiplex(ctx context.Context, dogwood *dogwood.DogwoodService) *Multiplex {
	x := &Multiplex{
		m:         xdrutil.NewXdrMarshaler(),
		dogwood:   dogwood,
		submitted: make(chan submittedTxsFrom),
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
	err := x.dogwood.Send(ctx, x.m.MarshalXdr(&xmsg).Bytes())
	if err != nil {
		liblog.Errorf("multiplex send: %v", err)
	}
}

func (x *Multiplex) Disseminate(ctx context.Context, submitted *proto.SubmittedTransactions) {
	x.SendSubmittedTxs(ctx, submitted)
}

func (x *Multiplex) ReceiveSubmittedTxs(ctx context.Context) (proto.ValidatorID, *proto.SubmittedTransactions) {
	submittedTxsFrom := <-x.submitted
	return submittedTxsFrom.From, submittedTxsFrom.SubmittedTransactions
}
