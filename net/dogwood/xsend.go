package dogwood

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TransmissionSender struct {
	cfg         proto.TransmissionConfig
	nav         *Navigator
	sender      *StreamSender
	sendWorkers *libflow.WorkerPool
	encPool     *MagnoliaEncoderPool
	rlk         sync.Mutex
	rand        *rand.Rand
	lk          sync.Mutex
	nsending    int
	nwaiting    int
}

func NewTransmissionSender(
	ctx context.Context,
	cfg proto.TransmissionConfig,
	nav *Navigator,
	sender *StreamSender,
	rand *rand.Rand,
	numWorkers int,
) *TransmissionSender {

	return &TransmissionSender{
		cfg:         cfg,
		nav:         nav,
		sender:      sender,
		rand:        rand,
		sendWorkers: libflow.NewWorkerPool(ctx, "transmission-send-workers", numWorkers),
		encPool: NewMagnoliaEncoderPool(
			ctx,
			int(cfg.Num_payload_sheaves),
			int(cfg.Num_parity_sheaves),
			int(cfg.Max_chunk_bytes),
		),
	}
}

func (x *TransmissionSender) NumSending() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.nsending
}

func (x *TransmissionSender) NumWaiting() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.nwaiting
}

func (x *TransmissionSender) Start(context.Context) {}

func (x *TransmissionSender) Send(
	ctx context.Context,
	typ proto.DogwoodTransmissionType,
	pay []byte,
	genRoute bool,
) {
	x.lk.Lock()
	x.nwaiting += 1
	x.lk.Unlock()

	x.sendWorkers.Submit(
		func() {
			x.lk.Lock()
			x.nsending += 1
			x.lk.Unlock()

			x.send(ctx, typ, pay, genRoute)

			x.lk.Lock()
			x.nsending -= 1
			x.lk.Unlock()
		},
	)

	x.lk.Lock()
	x.nwaiting -= 1
	x.lk.Unlock()
}

func (x *TransmissionSender) genID() proto.DogwoodTransmissionID {
	x.rlk.Lock()
	defer x.rlk.Unlock()
	return x.rand.Int63()
}

func (x *TransmissionSender) send(
	ctx context.Context,
	typ proto.DogwoodTransmissionType,
	pay []byte,
	genRoute bool,
) {

	must.Assert(ctx, len(pay) > 0)

	enc := x.encPool.Get()
	defer x.encPool.Recycle(enc)

	// encode
	cs := enc.Encode(pay)

	// header
	transmission := proto.DogwoodTransmission{
		Transmission_id:     x.genID(),
		Type:                typ,
		Num_payload_bytes:   proto.DogwoodInt64(len(pay)),
		Num_payload_sheaves: proto.DogwoodInt32(enc.NumPaySheaves()),
		Num_parity_sheaves:  proto.DogwoodInt32(enc.NumParitySheaves()),
		Max_bytes_per_chunk: proto.DogwoodInt32(enc.MaxBytesPerChunk()),
		Shuffle_seed:        0, // XXX: DogwoodInt32, shuffle not implemented
	}

	// open streams
	var err error
	stream := make([]*SendStream, enc.NumSheaves())

	ndialed := 0
	for s := 0; s < enc.NumSheaves(); s++ {
		if genRoute {
			stream[s], err = x.sender.DialRoute(ctx, x.nav.GenRoute(ctx))
		} else {
			stream[s], err = x.sender.Dial(ctx)
		}
		if err != nil {
			liblog.Errorf("transmission %d, sheaf %d, stream not created: %v",
				transmission.Transmission_id, s, err)
		}
		ndialed += 1
	}

	// check that enough streams opened for recovery
	if ndialed < enc.NumPaySheaves() {
		liblog.Errorf("dropping transmission %d, not enough streams, have %d, need %d",
			transmission.Transmission_id, ndialed, enc.NumPaySheaves())
		return
	}

	// send round robin
	for c := range cs {
		for s := range cs[c] {
			pkg := proto.DogwoodPackage{
				Transmission: transmission,
				Chunk: proto.DogwoodChunk{
					Sheaf_index: proto.DogwoodInt32(s),
					Chunk_index: proto.DogwoodInt32(c),
					Chunk:       cs[c][s],
				},
			}
			stream[s].Send(ctx, &pkg)
		}
	}
}
