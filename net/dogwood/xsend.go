package dogwood

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

// XXX: GetRoutes must return the requested number of routes, even when fewer available!

type TransmissionSender struct {
	cfg  proto.TransmissionConfig
	m    xdrutil.XdrMarshaler
	wp   *ants.Pool
	enc  *EncoderPool
	bulk BulkSender
	rlk  sync.Mutex
	rand *rand.Rand
}

func NewTransmissionSender(
	ctx context.Context,
	cfg proto.TransmissionConfig,
	rand *rand.Rand,
	bulk BulkSender,
	numWorkers int,
) *TransmissionSender {
	wp, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)
	return &TransmissionSender{
		cfg:  cfg,
		rand: rand,
		m:    xdrutil.NewXdrMarshaler(),
		wp:   wp,
		enc:  NewEncoderPool(ctx, int(cfg.Num_payload_sheaves), int(cfg.Num_parity_sheaves)),
		bulk: bulk,
	}
}

func (x *TransmissionSender) Send(ctx context.Context, bulk []byte) {
	x.wp.Submit(
		func() {
			x.send(ctx, bulk)
		},
	)
}

func (x *TransmissionSender) genID() proto.DogwoodTransmissionID {
	x.rlk.Lock()
	defer x.rlk.Unlock()
	return x.rand.Int63()
}

type chunk = []byte

func (x *TransmissionSender) send(ctx context.Context, bulk []byte) (proto.SpectraTransmissionID, error) {

	must.Assertf(ctx, len(bulk) > 0, "spectra empty payload")

	panic("XXX")
}
