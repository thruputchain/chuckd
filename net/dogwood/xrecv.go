package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/magnolia"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type TransmissionReceiver struct {
	cfg       proto.TransmissionConfig
	time      libtime.Time
	nav       *Navigator
	receiver  *StreamReceiver
	workPool  *ants.Pool //XXX: unused yet
	decPool   *MagnoliaDecoderPool
	receivedQ chan receivedTransmission
	lk        sync.Mutex
	partial   map[transmissionKey]*partialTransmission
}

type receivedTransmission struct {
	from proto.ValidatorID
	pay  []byte
}

type transmissionKey struct {
	proto.ValidatorID
	proto.DogwoodTransmissionID
}

type partialTransmission struct {
	arrived time.Time // time first chunk of transmission arrived
	header  proto.DogwoodTransmission
	lk      sync.Mutex
	dec     *magnolia.Decoder // XXX: must be recycled at end of transmission
}

func NewTransmissionReceiver(
	ctx context.Context,
	cfg proto.TransmissionConfig,
	time libtime.Time,
	nav *Navigator,
	receiver *StreamReceiver,
	numWorkers int,
) *TransmissionReceiver {

	wp, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)
	return &TransmissionReceiver{
		cfg:      cfg,
		time:     time,
		nav:      nav,
		receiver: receiver,
		workPool: wp,
		decPool: NewMagnoliaDecoderPool(
			ctx,
			int(cfg.Num_payload_sheaves),
			int(cfg.Num_parity_sheaves),
			int(cfg.Max_chunk_bytes),
		),
		receivedQ: make(chan receivedTransmission),
		partial:   map[transmissionKey]*partialTransmission{},
	}
}

func (x *TransmissionReceiver) Receive(ctx context.Context) ([]byte, proto.ValidatorID) {
	r := <-x.receivedQ
	return r.pay, r.from
}

func (x *TransmissionReceiver) Start(ctx context.Context) {
	go x.do(ctx)
	go x.expire(ctx)
}

// expire transmissions that have exceeded ttl
func (x *TransmissionReceiver) expire(ctx context.Context) {
	for {
		panic("XXX")
	}
}

func (x *TransmissionReceiver) do(ctx context.Context) {
	for {
		rs, err := x.receiver.Accept(ctx)
		if err != nil {
			liblog.Errorf("accepting stream: %v", err)
			continue
		}

		go func() {
			stream := rs.Stream()
			for {

				pkg, err := rs.Receive(ctx) //XXX: stream has to be closed by the partialTransmission manager
				if err != nil {
					panic("XXX")
				}

				key := transmissionKey{
					ValidatorID:           stream.Source_id,
					DogwoodTransmissionID: pkg.Transmission.Transmission_id,
				}

				par, err := x.getMakePartial(key, pkg.Transmission)
				_ = par
				if err != nil {
					liblog.Errorf("discarding transmission package: %v", err)
					continue
				}

				panic("XXX")
			}
		}()
	}
}

func (x *TransmissionReceiver) getMakePartial(
	key transmissionKey,
	transmission proto.DogwoodTransmission,
) (*partialTransmission, error) {

	x.lk.Lock()
	defer x.lk.Unlock()

	p, ok := x.partial[key]
	if !ok {
		p = &partialTransmission{
			arrived: x.time.Now(),
			header:  transmission,
			dec:     x.decPool.Get(),
		}
		x.partial[key] = p
	}

	return p, nil
}
