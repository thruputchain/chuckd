package catchup

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type CatchupService struct {
	signer  protoutil.Signatory
	time    libtime.Time
	net     libnet.Network
	dialDb  DialDb
	ttl     time.Duration // ttl of cached catchup blocks
	age     proto.BlockHeight
	shallow *pebble.DB // shallow cache
	//
	lk      sync.Mutex
	highest proto.BlockHeight
	block   map[proto.BlockHeight]stampedCatchupBlock
}

type DialDb interface {
	DialDb(context.Context) (sdk.DbCoordConn, error)
}

type stampedCatchupBlock struct {
	time.Time
	*proto.CatchupBlock
}

func NewCatchupService(
	ctx context.Context,
	signer protoutil.Signatory,
	time libtime.Time,
	net libnet.Network,
	dialDb DialDb,
	ttl time.Duration,
	age proto.BlockHeight,
) *CatchupService {

	// pebble.Open(cacheDir, &pebble.Options{})
	svc := &CatchupService{
		signer: signer,
		time:   time,
		net:    net,
		dialDb: dialDb,
		ttl:    ttl,
		age:    age,
		block:  map[proto.BlockHeight]stampedCatchupBlock{},
	}
	go svc.expire(ctx)
	return svc
}

func (svc *CatchupService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.Catchup1_Server{Srv: svc})
}

func (svc *CatchupService) Add(ctx context.Context, cb *proto.CatchupBlock) {
	svc.lk.Lock()
	defer svc.lk.Unlock()
	svc.highest = max(svc.highest, cb.Block_height)
	svc.block[cb.Block_height] = stampedCatchupBlock{Time: svc.time.Now(), CatchupBlock: cb}
}

func (svc *CatchupService) Lookup(ctx context.Context, height proto.BlockHeight) (*proto.CatchupBlock, bool) {
	svc.lk.Lock()
	defer svc.lk.Unlock()
	cb, ok := svc.block[height]
	if !ok {
		return nil, false
	}
	return cb.CatchupBlock, true
}

func (svc *CatchupService) expire(_ context.Context) {
	for {
		svc.time.Sleep(svc.ttl)
		now := svc.time.Now()
		svc.lk.Lock()
		for height, scb := range svc.block {
			if now.Sub(scb.Time) > svc.ttl || svc.highest > svc.age+scb.Block_height {
				delete(svc.block, height)
			}
		}
		svc.lk.Unlock()
	}
}

func (svc *CatchupService) Catchup_null(context.Context) error {
	return nil
}

func (svc *CatchupService) Catchup_solicit(ctx context.Context, sreq proto.SignedCatchupBlocksRequest) (proto.SignedCatchupBlocksResult, error) {

	// check signatures
	if !protoutil.VerifyXdr(sreq.Signer.Public_key, &sreq.Request, sreq.Signature) {
		return proto.SignedCatchupBlocksResult{}, fmt.Errorf("signature verification")
	}

	// compress spans to remove overlaps
	spans := make(protoutil.Spans[proto.BlockHeight], len(sreq.Request.Spans))
	for i, span := range sreq.Request.Spans {
		spans[i] = protoutil.Span[proto.BlockHeight]{From: span.From_block_height, To: span.To_block_height}
	}
	spans = protoutil.SumSpans(spans...)

	blocks := []proto.CatchupBlock{}
	for _, span := range spans {
		for h := span.From; h < span.To; h++ {
			if cb, ok := svc.Lookup(ctx, h); ok {
				blocks = append(blocks, *cb)
			}
		}
	}
	// log
	var heights bytes.Buffer
	for _, b := range blocks {
		fmt.Fprintf(&heights, "%d, ", b.Block_height)
	}
	liblog.Infof("serving catchup heights %s", heights.String())

	result := proto.CatchupBlocksResult{
		Timestamp: svc.time.Now().UnixNano(),
		Blocks:    blocks,
	}

	return proto.SignedCatchupBlocksResult{
		Signer:    svc.signer.Signer(),
		Signature: svc.signer.SignXdr(&result),
		Result:    result,
	}, nil
}
