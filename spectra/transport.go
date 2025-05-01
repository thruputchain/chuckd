// Package spectra provides a broadcast transport with forward erasure correction and path redundancy
package spectra

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/klauspost/reedsolomon"
)

type SpectraTransport struct {
	cfg proto.SpectraConfig
	rr  *rand.Rand
	bb  broadcast.BulkBroadcast
	// TODO: use a pool of encoders in parallel
	elk sync.Mutex
	enc reedsolomon.Encoder
	// TODO: use a pool of decoders in parallel
	dlk sync.Mutex
	dec reedsolomon.Encoder
	// partial receptions
	rlk     sync.Mutex
	partial map[transmissionKey]*partialTransmission
	//
	arrival chan transmissionArrival
}

type transmissionKey struct {
	validatorID    proto.ValidatorID
	transmissionID proto.SpectraTransmissionID
}

type partialTransmission struct {
	arrived     time.Time
	lk          sync.Mutex
	firstHeader proto.SpectraHeader
	shards      [][]byte
	decoded     bool
}

func (p *partialTransmission) CanDecode() bool {
	nreceived := 0
	for _, shard := range p.shards {
		if shard != nil {
			nreceived += 1
		}
	}
	return nreceived >= int(p.firstHeader.Num_payload_channels)
}

type transmissionArrival struct {
	senderID       proto.ValidatorID
	transmissionID proto.SpectraTransmissionID
	bulk           []byte
}

func NewSpectraTransport(
	ctx context.Context,
	cfg proto.SpectraConfig,
	rr *rand.Rand,
	bb broadcast.BulkBroadcast,
) *SpectraTransport {

	// redundancy beyond 100% can cause a transmission to be received twice.
	// this is fixable, but not necessary in practice.
	must.Assertf(ctx, cfg.Num_parity_channels < cfg.Num_payload_channels, "spectra too much redundancy")

	// make encoder
	enc, err := reedsolomon.New(
		int(cfg.Num_payload_channels),
		int(cfg.Num_parity_channels),
	)
	must.NoError(ctx, err)

	// make decoder
	dec, err := reedsolomon.New(
		int(cfg.Num_payload_channels),
		int(cfg.Num_parity_channels),
	)
	must.NoError(ctx, err)

	// start receiver and collector
	x := &SpectraTransport{
		cfg:     cfg,
		rr:      rr,
		bb:      bb,
		enc:     enc,
		dec:     dec,
		partial: map[transmissionKey]*partialTransmission{},
		arrival: make(chan transmissionArrival, 1),
	}
	go x.expire()
	go x.receive(ctx)
	return x
}

func (x *SpectraTransport) expire() {
	ttl := time.Duration(x.cfg.Partial_ttl_msec) * time.Millisecond
	for {
		time.Sleep(ttl)
		now := time.Now()
		x.rlk.Lock()
		for k, p := range x.partial {
			if now.Sub(p.arrived) > ttl {
				delete(x.partial, k)
			}
		}
		x.rlk.Unlock()
	}
}

var headerSize int = len(xdrutil.MarshalXdr(&proto.SpectraHeader{}))

func (x *SpectraTransport) BulkSend(ctx context.Context, bulk []byte) (proto.SpectraTransmissionID, error) {

	must.Assertf(ctx, len(bulk) > 0, "spectra empty payload")

	// compute shard size
	n := len(bulk)
	p := int(x.cfg.Num_payload_channels)
	s := (n + p - 1) / p // shard size
	q := int(x.cfg.Num_parity_channels)

	must.Assertf(ctx, s > 0, "spectra empty shard") // would be a bug

	// prepare header template
	h := proto.SpectraHeader{
		Transmission_id:      x.rr.Int63(),
		Num_payload_channels: x.cfg.Num_payload_channels,
		Num_parity_channels:  x.cfg.Num_parity_channels,
		Payload_size:         uint64(n),
		Shard_size:           uint64(s),
		// leave Shard_index yet unassigned
	}

	// prepare payload shards and headers
	shardsWithHeader := make([][]byte, p+q)
	for i := 0; i < p; i++ {
		shardsWithHeader[i] = make([]byte, headerSize+s)
		h.Shard_index = uint32(i)
		copy(shardsWithHeader[i][:headerSize], xdrutil.MarshalXdr(&h))            // prepend header
		copy(shardsWithHeader[i][headerSize:], bulk[min(i*s, n):min((i+1)*s, n)]) // append shard payload
	}

	// prepare parity shard headers
	for i := 0; i < q; i++ {
		shardsWithHeader[p+i] = make([]byte, headerSize+s)
		h.Shard_index = uint32(p + i)
		copy(shardsWithHeader[p+i][:headerSize], xdrutil.MarshalXdr(&h)) // prepend header
	}

	shardsWithoutHeader := make([][]byte, p+q)
	for i := range shardsWithHeader {
		shardsWithoutHeader[i] = shardsWithHeader[i][headerSize:]
	}

	// compute parity shards
	err := x.encode(shardsWithoutHeader)
	if err != nil {
		return h.Transmission_id, err
	}

	// send shards
	for i := range shardsWithHeader {
		_, err := x.bb.BulkSend(ctx, shardsWithHeader[i])
		if err != nil {
			return h.Transmission_id, err
		}
	}

	return h.Transmission_id, nil
}

func (x *SpectraTransport) encode(shards [][]byte) error {
	x.elk.Lock()
	defer x.elk.Unlock()
	return x.enc.Encode(shards)
}

func (x *SpectraTransport) decode(shards [][]byte) error {
	x.dlk.Lock()
	defer x.dlk.Unlock()
	return x.dec.Reconstruct(shards)
}

func (x *SpectraTransport) BulkReceive(ctx context.Context) (proto.ValidatorID, proto.SpectraTransmissionID, []byte, error) {
	a := <-x.arrival
	return a.senderID, a.transmissionID, a.bulk, nil
}

func (x *SpectraTransport) receive(ctx context.Context) {
	for {
		// receive shard
		ch, bulk, err := x.bb.BulkReceive(ctx)
		if err != nil {
			fmt.Println("XXX", err)
			log.Errorf("spectra bulk reception: %v", err)
			break
		}
		fmt.Println("BULK", len(bulk), "AT", protoutil.ValidatorIDString(x.bb.ValidatorID()))

		// decode header
		if len(bulk) < headerSize {
			log.Errorf("spectra reception too short %d", len(bulk))
			break
		}
		var h proto.SpectraHeader
		err = xdrutil.UnmarshalXdr(&h, bulk[:headerSize])
		if err != nil {
			log.Errorf("spectra reception invalid header: %v", err)
			break
		}

		x.process(ctx, ch, h, bulk[headerSize:])
	}
}

func (x *SpectraTransport) process(_ context.Context, ch broadcast.Channel, h proto.SpectraHeader, shard []byte) {

	// construct key
	key := transmissionKey{
		validatorID:    ch.Source_id,
		transmissionID: h.Transmission_id,
	}

	// two receptions could race to create a partialTransmission structure.
	// TODO: race may be ok to allow to enable parallelism; needs special care
	x.rlk.Lock()
	defer x.rlk.Unlock()

	p := x.lookupPartial(key)
	// if we are encountering this transmission for the first time
	if p == nil {
		// verify header sanity
		if h.Shard_index >= h.Num_payload_channels+h.Num_parity_channels {
			log.Errorf("spectra invalid header %s", form.SprintJSON(h))
			return
		}
		// verify shard size is correct for number of payload and parity channels
		if h.Shard_size == 0 ||
			h.Shard_size*uint64(h.Num_payload_channels) < h.Payload_size {
			log.Errorf("spectra invalid shard size in header %s", form.SprintJSON(h))
			return
		}
		p = &partialTransmission{
			arrived:     time.Now(),
			firstHeader: h,
			shards:      make([][]byte, h.Num_payload_channels+h.Num_parity_channels),
		}
		x.savePartial(key, p)
	}

	if p.decoded {
		return
	}

	p.lk.Lock()
	defer p.lk.Unlock()

	// verify header matches first header
	if h.Num_payload_channels != p.firstHeader.Num_payload_channels ||
		h.Num_parity_channels != p.firstHeader.Num_parity_channels ||
		h.Payload_size != p.firstHeader.Payload_size ||
		h.Shard_size != p.firstHeader.Shard_size {
		log.Errorf("spectra subsequent header does not match")
		return
	}

	// verify actual shard size matches header
	if len(shard) != int(h.Shard_size) {
		log.Errorf("spectra actual shard size does not match header")
		return
	}

	// update partial structure
	p.shards[h.Shard_index] = shard

	// can we decode the payload?
	if !p.CanDecode() {
		return
	}

	// decode payload
	err := x.decode(p.shards)
	if err != nil {
		log.Errorf("spectra cannot reconstruct: %v", err)
		return
	}

	// merge payload shards and trim to payload size from header
	bulk := make([]byte, 0,
		int(p.firstHeader.Num_payload_channels+p.firstHeader.Num_parity_channels)*int(p.firstHeader.Shard_size))
	for i := 0; i < int(p.firstHeader.Num_payload_channels); i++ {
		bulk = append(bulk, p.shards[i]...)
	}
	bulk = bulk[:p.firstHeader.Payload_size]

	p.decoded = true

	// notify app
	x.arrival <- transmissionArrival{
		senderID:       key.validatorID,
		transmissionID: key.transmissionID,
		bulk:           bulk,
	}
}

func (x *SpectraTransport) lookupPartial(key transmissionKey) *partialTransmission {
	return x.partial[key]
}

func (x *SpectraTransport) savePartial(key transmissionKey, partial *partialTransmission) {
	x.partial[key] = partial
}
