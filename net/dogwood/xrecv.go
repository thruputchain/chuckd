package dogwood

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/magnolia"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

// NOTE: transmission can be received twice if code redundancy exceeds 100%.
// For this case, the receiver caches received transmission IDs for the TTL of a partial transmission,
// reducing the likelihood of duplicate receptions.

type TransmissionReceiver struct {
	cfg      proto.TransmissionConfig
	time     libtime.Time
	nav      *Navigator
	receiver *StreamReceiver
	workPool *ants.Pool
	decPool  *MagnoliaDecoderPool
	receiveQ chan receivedTransmission
	rlk      sync.Mutex
	received map[transmissionKey]time.Time
	plk      sync.Mutex
	partial  map[transmissionKey]*partialTransmission
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
	arrived time.Time // time first chunk of transmission arrived
	header  proto.DogwoodTransmission
	done    sync.Once
	lk      sync.Mutex
	dec     *magnolia.Decoder
	ding    *magnolia.Decoding
}

func NewTransmissionReceiver(
	ctx context.Context,
	cfg proto.TransmissionConfig,
	time_ libtime.Time,
	nav *Navigator,
	receiver *StreamReceiver,
	numWorkers int,
) *TransmissionReceiver {

	wp, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)
	return &TransmissionReceiver{
		cfg:      cfg,
		time:     time_,
		nav:      nav,
		receiver: receiver,
		workPool: wp,
		decPool: NewMagnoliaDecoderPool(
			ctx,
			int(cfg.Num_payload_sheaves),
			int(cfg.Num_parity_sheaves),
			int(cfg.Max_chunk_bytes),
		),
		receiveQ: make(chan receivedTransmission),
		received: map[transmissionKey]time.Time{},
		partial:  map[transmissionKey]*partialTransmission{},
	}
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
	go x.do(ctx)
	go x.expirePartial(ctx)
	go x.expireReceived(ctx)
}

func (x *TransmissionReceiver) expireReceived(_ context.Context) {
	ttl := time.Duration(x.cfg.Partial_ttl_msec) * time.Millisecond
	for {
		x.time.Sleep(ttl)
		now := x.time.Now()

		x.rlk.Lock()
		for key, stamp := range x.received {
			if now.Sub(stamp) > ttl {
				delete(x.received, key)
			}
		}
		x.rlk.Unlock()
	}
}

// expirePartial transmissions that have exceeded ttl
func (x *TransmissionReceiver) expirePartial(_ context.Context) {
	ttl := time.Duration(x.cfg.Partial_ttl_msec) * time.Millisecond
	for {
		x.time.Sleep(ttl)
		now := x.time.Now()

		x.plk.Lock()
		scrub := []*partialTransmission{}
		for _, par := range x.partial {
			if now.Sub(par.arrived) > ttl {
				scrub = append(scrub, par)
			}
		}
		x.plk.Unlock()

		for _, par := range scrub {
			par.Close()
		}
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
			defer rs.Close()

			stream := rs.Stream()
			for {

				pkg, err := rs.Receive(ctx)
				if err != nil {
					liblog.Errorf("broken stream: %v", err)
					return
				}

				key := transmissionKey{
					ValidatorID:           stream.Source_id,
					DogwoodTransmissionID: pkg.Transmission.Transmission_id,
				}

				par, err := x.getMakePartial(key, pkg.Transmission)
				if err != nil {
					liblog.Errorf("retrieving partial transmission: %v", err)
					continue // only this package is corrupt
				}

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
					continue // only this pkg is corrupt
				}

			}
		}()
	}
}

func (x *TransmissionReceiver) getMakePartial(
	key transmissionKey,
	transmission proto.DogwoodTransmission,
) (*partialTransmission, error) {

	x.plk.Lock()
	defer x.plk.Unlock()

	p, ok := x.partial[key]
	if !ok {
		dec := x.decPool.Get()
		p = &partialTransmission{
			parent:  x,
			key:     key,
			arrived: x.time.Now(),
			header:  transmission,
			dec:     dec,
			ding:    dec.StartDecoding(int(transmission.Num_payload_bytes)),
		}
		x.partial[key] = p
	} else {
		if transmission != p.header {
			return nil, fmt.Errorf("mismatched transmission header")
		}
	}

	return p, nil
}

type placeResult struct {
	done bool
	err  error
}

func (x *partialTransmission) PlaceUsingWorker(sheafIndex int, chunkIndex int, chunk []byte) (done bool, err error) {

	resultQ := make(chan placeResult, 1)
	x.parent.workPool.Submit(
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

	x.done.Do(func() { x.parent.receiveQ <- receivedTransmission{key: x.key, typ: x.header.Type, pay: pay} })

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
