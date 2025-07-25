package dogwood

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/magnolia"
	proto "github.com/groundhogchain/protocol-go"
)

// NOTE: transmission can be received twice if code redundancy exceeds 100%.
// For this case, the receiver caches received transmission IDs for the TTL of a partial transmission,
// reducing the likelihood of duplicate receptions.

type TransmissionReceiver struct {
	partialMaxTTL  time.Duration
	partialIdleTTL time.Duration
	time           libtime.Time
	nav            *Navigator
	receiver       *StreamReceiver
	//
	decWorkers libflow.SubmitWorkerPool
	decPool    *MagnoliaDecoderPool
	//
	receiveQ chan receivedTransmission
	//
	plk     sync.Mutex
	partial map[transmissionKey]*partialTransmission
	//
	latency libstat.LatencyMeter
}

type receivedTransmission struct {
	key transmissionKey
	typ proto.DogwoodTransmissionType
	pay []byte
}

type transmissionKey struct {
	proto.ValidatorID
	proto.DogwoodTransmissionID
}

type partialTransmission struct {
	parent  *TransmissionReceiver
	key     transmissionKey
	arrived time.Time // first chunk arrival
	header  proto.DogwoodTransmission
	//
	lk       sync.Mutex
	latest   time.Time // latest chunk arrival
	decoder  *magnolia.Decoder
	decoding *magnolia.Decoding
}

func NewTransmissionReceiver(
	ctx context.Context,
	cfg proto.DogwoodConfig,
	time_ libtime.Time,
	nav *Navigator,
	receiver *StreamReceiver,
) *TransmissionReceiver {

	x := &TransmissionReceiver{
		partialMaxTTL:  time.Duration(cfg.Transmission_partial_max_ttl_msec) * time.Millisecond,
		partialIdleTTL: time.Duration(cfg.Transmission_partial_idle_ttl_msec) * time.Millisecond,
		time:           time_,
		nav:            nav,
		receiver:       receiver,
		decWorkers:     libflow.NewSubmitWorkerPool(ctx, int(cfg.Transmission_receiver_decode_worker_count)),
		decPool:        NewMagnoliaDecoderPool(ctx),
		receiveQ:       make(chan receivedTransmission),
		partial:        map[transmissionKey]*partialTransmission{},
	}
	x.receiver.SetProcessor(x)
	return x
}

func (x *TransmissionReceiver) NumPartial() int {
	x.plk.Lock()
	defer x.plk.Unlock()
	return len(x.partial)
}

func (x *TransmissionReceiver) Receive(ctx context.Context) (proto.DogwoodTransmissionType, []byte, proto.ValidatorID, error) {
	for {
		select {
		case r := <-x.receiveQ:
			return r.typ, r.pay, r.key.ValidatorID, nil
		case <-ctx.Done():
			return 0, nil, proto.ValidatorID{}, ctx.Err()
		}
	}
}

func (x *TransmissionReceiver) Start(ctx context.Context) {
	go x.expirePartial(ctx)
}

// expirePartial transmissions that have exceeded max or idle ttl
func (x *TransmissionReceiver) expirePartial(_ context.Context) {
	tick := min(x.partialMaxTTL, x.partialIdleTTL)
	for {
		x.time.Sleep(tick)
		now := x.time.Now()

		x.plk.Lock()
		scrub := []*partialTransmission{}
		for _, par := range x.partial {
			if now.Sub(par.arrived) > x.partialMaxTTL || now.Sub(par.getLatest()) > x.partialIdleTTL {
				scrub = append(scrub, par)
			}
		}
		x.plk.Unlock()

		for _, par := range scrub {
			if !par.IsComplete() {
				liblog.Errorf("dogwood dropping partial incomplete transmission")
			}
			par.Close()
		}
	}
}

func (x *TransmissionReceiver) OnReceive(from proto.ValidatorID, pkg *proto.DogwoodPackage) {

	key := transmissionKey{
		ValidatorID:           from,
		DogwoodTransmissionID: pkg.Transmission.Transmission_id,
	}

	par, err := x.getPartial(key, pkg.Transmission)
	if err != nil {
		liblog.Errorf("partial transmission: %v", err)
		return
	}

	par.setLatest(x.time.Now())

	if par.IsComplete() {
		return
	}

	par.PlaceUsingWorker(
		int(pkg.Chunk.Sheaf_index),
		int(pkg.Chunk.Chunk_index),
		pkg.Chunk.Chunk,
	)
}

func (x *TransmissionReceiver) getPartial(
	key transmissionKey,
	transmission proto.DogwoodTransmission,
) (*partialTransmission, error) {

	x.plk.Lock()
	defer x.plk.Unlock()

	p, ok := x.partial[key]
	if !ok {
		regime := magnolia.MkRegime(
			int(transmission.Num_payload_sheaves),
			int(transmission.Num_parity_sheaves),
			int(transmission.Max_bytes_per_chunk),
		)
		dec := x.decPool.Get(regime)
		now := x.time.Now()
		p = &partialTransmission{
			parent:   x,
			key:      key,
			arrived:  now,
			header:   transmission,
			latest:   now,
			decoder:  dec,
			decoding: dec.StartDecoding(int(transmission.Num_payload_bytes)),
		}
		x.partial[key] = p
	} else {
		if transmission != p.header {
			return nil, fmt.Errorf("corrupt transmission")
		}
	}

	return p, nil
}

func (x *partialTransmission) setLatest(v time.Time) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.latest = v
}

func (x *partialTransmission) getLatest() time.Time {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.latest
}

func (x *partialTransmission) PlaceUsingWorker(sheafIndex int, chunkIndex int, chunk []byte) {
	x.parent.decWorkers.Submit(
		func() {
			x.Place(sheafIndex, chunkIndex, chunk)
		},
	)
}

func (x *partialTransmission) IsComplete() bool {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.decoding == nil
}

func (x *partialTransmission) Place(sheafIndex int, chunkIndex int, chunk []byte) {

	pay, decoded := x.place(sheafIndex, chunkIndex, chunk)
	if !decoded {
		return
	}

	x.parent.receiveQ <- receivedTransmission{key: x.key, typ: x.header.Type, pay: pay}
}

func (x *partialTransmission) place(sheafIndex int, chunkIndex int, chunk []byte) (pay []byte, decoded bool) {

	x.lk.Lock()
	defer x.lk.Unlock()

	if x.decoding == nil {
		return nil, false // partial transmission already completed
	}

	err := x.decoding.Place(sheafIndex, chunkIndex, chunk)
	if err != nil {
		liblog.Debugf("dogwood placing chunk: %v", err)
		return nil, false
	}

	pay, err = x.decoding.Decode()
	if err != nil {
		return nil, false // not ready to decode
	}

	latency := x.latest.Sub(x.arrived)
	x.parent.latency.Add(latency)

	// indicate completion, but leave the partial transmission registered with the parent
	x.parent.decPool.Recycle(x.decoder)
	x.decoder, x.decoding = nil, nil // indicate completion

	return pay, true
}

func (x *partialTransmission) Close() {

	// deregister from receiver
	x.parent.plk.Lock()
	p, ok := x.parent.partial[x.key]
	if ok && p == x {
		delete(x.parent.partial, x.key)
	}
	x.parent.plk.Unlock()

	// recycle decoder
	x.lk.Lock()
	if x.decoder != nil {
		x.parent.decPool.Recycle(x.decoder)
		x.decoder, x.decoding = nil, nil
	}
	x.lk.Unlock()
}

// stats

type TransmissionReceiverStats struct {
	libflow.WorkerPoolStats
	Latency libstat.LatencyStats
}

func (x *TransmissionReceiver) Stats() TransmissionReceiverStats {
	return TransmissionReceiverStats{
		WorkerPoolStats: x.decWorkers.Stats(),
		Latency:         x.latency.Stats(),
	}
}
