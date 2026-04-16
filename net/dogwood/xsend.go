package dogwood

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/lib-go/libflow"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libmem"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TransmissionSender struct {
	nav    *Navigator
	sender *StreamSender
	//
	sendWorkers libflow.SubmitWorkerPool
	encPool     *MagnoliaEncoderPool
	//
	rlk  sync.Mutex
	rand *rand.Rand
}

func NewTransmissionSender(
	ctx context.Context,
	cfg proto.DogwoodConfig,
	nav *Navigator,
	sender *StreamSender,
	rand *rand.Rand,
) *TransmissionSender {

	return &TransmissionSender{
		nav:         nav,
		sender:      sender,
		rand:        rand,
		sendWorkers: libflow.NewSubmitWorkerPool(ctx, int(cfg.Transmission_sender_worker_count)),
		encPool:     NewMagnoliaEncoderPool(ctx),
	}
}

func (x *TransmissionSender) Start(context.Context) {}

func (x *TransmissionSender) Send(
	ctx context.Context,
	typ proto.DogwoodTransmissionType,
	pay libmem.BorrowedBytes,
	genRoute bool,
) {
	x.sendWorkers.Submit(
		func() {
			x.send(ctx, typ, pay, genRoute)
		},
	)
}

func (x *TransmissionSender) genID() proto.DogwoodTransmissionID {
	x.rlk.Lock()
	defer x.rlk.Unlock()
	return x.rand.Uint64()
}

func (x *TransmissionSender) send(
	ctx context.Context,
	typ proto.DogwoodTransmissionType,
	pay libmem.BorrowedBytes,
	genRoute bool,
) {

	must.Assert(ctx, pay.Len() > 0)

	regime := CalcRegime(pay.Len())

	enc := x.encPool.Get(regime)
	defer x.encPool.Recycle(enc)

	// encode
	cs := enc.Encode(pay.Bytes())

	// header
	transmission := proto.DogwoodTransmission{
		Transmission_id:     x.genID(),
		Type:                typ,
		Num_payload_bytes:   proto.DogwoodInt64(pay.Len()),
		Num_payload_sheaves: proto.DogwoodInt32(regime.NumPaySheaves),
		Num_parity_sheaves:  proto.DogwoodInt32(regime.NumParitySheaves),
		Max_bytes_per_chunk: proto.DogwoodInt32(regime.MaxBytesPerChunk),
		Shuffle_seed:        0, // XXX: DogwoodInt32, shuffle not implemented
	}

	// open streams
	var err error
	stream := make([]*SendStream, 0, enc.NumSheaves())

	for s := 0; s < enc.NumSheaves(); s++ {
		var dialed *SendStream
		if genRoute {
			dialed, err = x.sender.DialRoute(ctx, x.nav.GenRoute(ctx))
		} else {
			dialed, err = x.sender.Dial(ctx)
		}
		if err == nil {
			stream = append(stream, dialed)
		}
	}

	// close all streams
	defer func() {
		for _, s := range stream {
			s.Close()
		}
	}()

	// check that enough streams opened for recovery
	if len(stream) < enc.NumPaySheaves() {
		liblog.Errorf(ctx, "dropping transmission %d, not enough streams, have %d, need %d",
			transmission.Transmission_id, len(stream), enc.NumPaySheaves())
		return
	}

	// send round robin
	var wg sync.WaitGroup
	wg.Add(len(cs) * len(stream))

	for c := range cs {
		for s := range stream {
			pkg := proto.DogwoodPackage{
				Transmission: transmission,
				Chunk: proto.DogwoodChunk{
					Sheaf_index: proto.DogwoodInt32(s),
					Chunk_index: proto.DogwoodInt32(c),
					Chunk:       cs[c][s],
				},
			}
			stream[s].Send(ctx, libmem.BorrowOnWaitGroup(&wg, &pkg))
		}
	}

	go func() {
		wg.Wait()
		pay.Recycle()
	}()
}

// stats

type TransmissionSenderStats struct {
	libflow.WorkerPoolStats
}

func (x *TransmissionSender) Stats() TransmissionSenderStats {
	return TransmissionSenderStats{
		WorkerPoolStats: x.sendWorkers.Stats(),
	}
}
