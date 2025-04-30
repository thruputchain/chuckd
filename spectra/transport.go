// Package spectra provides a broadcast transport with forward erasure correction and path redundancy
package spectra

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/lib/must"
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
	// partial receptions
	rlk     sync.Mutex
	partial map[transmissionKey]*partialTransmission
	//
	arrival chan transmissionArrival
}

type transmissionKey struct {
	channel        broadcast.Channel
	transmissionID proto.SpectraTransmissionID
}

type partialTransmission struct {
	arrived              time.Time
	headers              []*proto.SpectraHeader
	shardsWithoutHeaders [][]byte
}

type transmissionArrival struct {
	channels []broadcast.Channel
	bulk     []byte
}

func NewSpectraTransport(
	ctx context.Context,
	cfg proto.SpectraConfig,
	rr *rand.Rand,
	bb broadcast.BulkBroadcast,
) *SpectraTransport {
	enc, err := reedsolomon.New(
		int(cfg.Num_payload_channels),
		int(cfg.Num_parity_channels),
	)
	must.NoError(ctx, err)
	x := &SpectraTransport{
		cfg:     cfg,
		rr:      rr,
		bb:      bb,
		enc:     enc,
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

func (x *SpectraTransport) BulkSend(ctx context.Context, bulk []byte) ([]broadcast.Channel, error) {

	// compute shard size
	n := len(bulk)
	p := int(x.cfg.Num_payload_channels)
	s := (n + p - 1) / p // shard size
	q := int(x.cfg.Num_parity_channels)

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
		return nil, err
	}

	// send shards
	chs := make([]broadcast.Channel, len(shardsWithHeader))
	for i := range shardsWithHeader {
		ch, err := x.bb.BulkSend(ctx, shardsWithHeader[i])
		if err != nil {
			return nil, err
		}
		chs[i] = ch
	}

	return chs, nil
}

func (x *SpectraTransport) encode(shards [][]byte) error {
	x.elk.Lock()
	defer x.elk.Unlock()
	return x.enc.Encode(shards)
}

func (x *SpectraTransport) BulkReceive(ctx context.Context) ([]broadcast.Channel, []byte, error) {
	a := <-x.arrival
	return a.channels, a.bulk, nil
}

func (x *SpectraTransport) receive(ctx context.Context) {
	panic("XXX")
}
