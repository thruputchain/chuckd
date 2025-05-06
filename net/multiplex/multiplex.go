package multiplex

import (
	"context"

	"github.com/groundhogchain/chuckd/net/spectra"
	proto "github.com/groundhogchain/protocol-go"
)

// scp msgs, proposed txsets,
type Multiplex struct {
	transport *spectra.SpectraTransport
}

func (x *Multiplex) SendBatch(ctx context.Context, batch proto.BatchMessage) {
	panic("XXX")
}

func (x *Multiplex) ReceiveBatch(ctx context.Context) proto.BatchMessage {
	panic("XXX")
}
