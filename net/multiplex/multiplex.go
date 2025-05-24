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
	transport *spectra.SpectraTransport
	batch     chan batchMsgFrom
}

type batchMsgFrom struct {
	From proto.ValidatorID
	*proto.BatchMessage
}

func NewMultiplex(ctx context.Context, s *spectra.SpectraTransport) *Multiplex {
	x := &Multiplex{
		transport: s,
		batch:     make(chan batchMsgFrom),
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
		case proto.MultiplexMessageBatch:
			x.batch <- batchMsgFrom{From: fromID, BatchMessage: xmsg.Batch()}
		default:
			liblog.Errorf("unsupported multiplex message type %v", xmsg.Type)
		}
	}
}

// XXX: should batch messages be signed?
func (x *Multiplex) SendBatch(ctx context.Context, batch *proto.BatchMessage) {
	var xmsg proto.MultiplexMessage
	xmsg.Type = proto.MultiplexMessageBatch
	*xmsg.Batch() = *batch
	_, err := x.transport.BulkSend(ctx, xdrutil.MarshalXdr(&xmsg))
	if err != nil {
		liblog.Errorf("multiplex bul.k send")
	}
}

func (x *Multiplex) ReceiveBatch(ctx context.Context) (proto.ValidatorID, *proto.BatchMessage) {
	batchMsgFrom := <-x.batch
	return batchMsgFrom.From, batchMsgFrom.BatchMessage
}
