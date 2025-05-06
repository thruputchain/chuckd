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

func NewMultiplex(s *spectra.SpectraTransport) *Multiplex {
	return &Multiplex{
		transport: s,
	}
}

// XXX: should batch messages be signed?
func (x *Multiplex) SendBatch(ctx context.Context, batch proto.BatchMessage) {
	panic("XXX")
}

func (x *Multiplex) ReceiveBatch(ctx context.Context) proto.BatchMessage {
	panic("XXX")
}
