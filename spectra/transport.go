// Package spectra provides a broadcast transport with forward erasure correction and path redundancy
package spectra

import (
	"context"

	"github.com/groundhogchain/chuckd/broadcast"
	proto "github.com/groundhogchain/protocol-go"
)

type SilkTransport struct {
	cfg proto.SpectraConfig
	bb  broadcast.BulkBroadcast
}

func NewSilkTransport(ctx context.Context, bb broadcast.BulkBroadcast) *SilkTransport {
	return &SilkTransport{bb: bb}
}

func (x *SilkTransport) BulkSend(context.Context, []byte) (broadcast.Channel, error) {
	panic("XXX")
}

func (x *SilkTransport) BulkReceive(context.Context) (broadcast.Channel, []byte, error) {
	panic("XXX")
}
