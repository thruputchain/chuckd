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
	decWorkers     *libflow.WorkerPool
	decPool        *MagnoliaDecoderPool
	receiveQ       chan receivedTransmission
	rlk            sync.Mutex
	received       map[transmissionKey]time.Time
	plk            sync.Mutex
	partial        map[transmissionKey]*partialTransmission
	//
	latency *libstat.LatencyMeter
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
	emit    sync.Once
	lk      sync.Mutex
	latest  time.Time // latest chunk arrival
	dec     *magnolia.Decoder
	ding    *magnolia.Decoding
}

func NewTransmissionReceiver(
	ctx context.Context,
	cfg proto.TransmissionConfig,
	time_ libtime.Time,
	nav *Navigator,
	receiver *StreamReceiver,
	numDecWorkers int,
	numStreamWorkers int, //XXX: not used
) *TransmissionReceiver {

	x := &TransmissionReceiver{
		partialMaxTTL:  time.Duration(cfg.Partial_max_ttl_msec) * time.Millisecond,
		partialIdleTTL: time.Duration(cfg.Partial_idle_ttl_msec) * time.Millisecond,
		time:           time_,
		nav:            nav,
		receiver:       receiver,
		decWorkers:     libflow.NewWorkerPool(ctx, "transmission-receive-decode-workers", numDecWorkers),
		decPool: NewMagnoliaDecoderPool(
			ctx,
			int(cfg.Num_payload_sheaves),
			int(cfg.Num_parity_sheaves),
			int(cfg.Max_chunk_bytes),
		),
		receiveQ: make(chan receivedTransmission),
		received: map[transmissionKey]time.Time{},
		partial:  map[transmissionKey]*partialTransmission{},
		latency:  libstat.NewLatency(time_),
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

			x.rlk.Lock()
			_, ok := x.received[r.key]
			if !ok {
				x.received[r.key] = x.time.Now()
			}
			x.rlk.Unlock()

			if !ok {
				return r.typ, r.pay, r.key.ValidatorID, nil
			}

		case <-ctx.Done():
			return 0, nil, proto.ValidatorID{}, ctx.Err()
		}
	}
}

func (x *TransmissionReceiver) Start(ctx context.Context) {
	go x.expirePartial(ctx)
	go x.expireReceived(ctx)
}

func (x *TransmissionReceiver) expireReceived(_ context.Context) {
	for {
		x.time.Sleep(x.partialMaxTTL)
		now := x.time.Now()

		x.rlk.Lock()
		for key, stamp := range x.received {
			if now.Sub(stamp) > x.partialMaxTTL {
				delete(x.received, key)
			}
		}
		x.rlk.Unlock()
	}
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
			if now.Sub(par.arrived) > x.partialMaxTTL ||
				now.Sub(par.getLatest()) > x.partialIdleTTL {
				scrub = append(scrub, par)
			}
		}
		x.plk.Unlock()

		for _, par := range scrub {
			liblog.Infof("dogwood dropping partial transmission")
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

	done, err := par.PlaceUsingWorker(
		int(pkg.Chunk.Sheaf_index),
		int(pkg.Chunk.Chunk_index),
		pkg.Chunk.Chunk,
	)
	if done {
		par.Close()
		return
	}
	if err != nil {
		liblog.Errorf("corrupt transmission package: %v", err)
		return
	}
}

func (x *TransmissionReceiver) getPartial(
	key transmissionKey,
	transmission proto.DogwoodTransmission,
) (*partialTransmission, error) {

	x.plk.Lock()
	defer x.plk.Unlock()

	p, ok := x.partial[key]
	if !ok {
		dec := x.decPool.Get()
		now := x.time.Now()
		p = &partialTransmission{
			parent:  x,
			key:     key,
			arrived: now,
			header:  transmission,
			latest:  now,
			dec:     dec,
			ding:    dec.StartDecoding(int(transmission.Num_payload_bytes)),
		}
		x.partial[key] = p
	} else {
		if transmission != p.header {
			return nil, fmt.Errorf("corrupt transmission")
		}
	}

	return p, nil
}

type placeResult struct {
	done bool
	err  error
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

func (x *partialTransmission) PlaceUsingWorker(sheafIndex int, chunkIndex int, chunk []byte) (done bool, err error) {

	resultQ := make(chan placeResult, 1)
	x.parent.decWorkers.Submit(
		func() {
			done, err := x.Place(sheafIndex, chunkIndex, chunk)
			resultQ <- placeResult{done, err}
		},
	)
	result := <-resultQ
	return result.done, result.err
}

func (x *partialTransmission) Place(sheafIndex int, chunkIndex int, chunk []byte) (done bool, err error) {

	x.lk.Lock()
	defer x.lk.Unlock()

	if x.ding == nil {
		return true, fmt.Errorf("partial transmission already closed")
	}

	err = x.ding.Place(sheafIndex, chunkIndex, chunk)
	if err != nil {
		return false, err
	}

	pay, err := x.ding.Decode()
	if err != nil {
		return false, nil // not ready to decode
	}

	latency := x.latest.Sub(x.arrived)

	x.emit.Do(
		func() {
			count := x.parent.latency.Add(latency)
			if count%10 == 0 {
				liblog.Infof("transmission reception latency: %v", x.parent.latency)
			}

			x.parent.receiveQ <- receivedTransmission{key: x.key, typ: x.header.Type, pay: pay}
		},
	)

	return true, nil
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
	if x.dec != nil {
		x.parent.decPool.Recycle(x.dec)
		x.dec, x.ding = nil, nil
	}
	x.lk.Unlock()
}
