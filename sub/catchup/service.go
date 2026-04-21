package catchup

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type CatchupService struct {
	signer protoutil.Signatory
	time   libtime.Time
	net    libnet.Network
	pool   CBPool
	//
	reqM    libxdr.XdrMarshaler
	resultM libxdr.XdrMarshaler
	//
	l net.Listener
}

func NewCatchupService(
	ctx context.Context,
	signer protoutil.Signatory,
	time libtime.Time,
	net libnet.Network,
	pool CBPool,
) *CatchupService {

	return &CatchupService{
		signer:  signer,
		time:    time,
		net:     net,
		pool:    pool,
		reqM:    libxdr.NewXdrMarshaler(),
		resultM: libxdr.NewXdrMarshaler(),
	}
}

func (svc *CatchupService) Start(ctx context.Context, address string) {
	svc.l = librpc.StartSerial(
		ctx,
		svc.net,
		netutil.MaxMessageSize,
		rpcutil.BaseDriverRegime,
		address,
		proto.Catchup1_Server{Srv: svc},
	)
}

func (svc *CatchupService) Stop() {
	if svc.l == nil {
		return
	}
	_ = svc.l.Close()
	svc.l = nil
}

func (svc *CatchupService) Lookup(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	return svc.pool.CBPoolFindCommitted(ctx, height)
}

func (svc *CatchupService) Catchup_null(context.Context) error {
	return nil
}

func (svc *CatchupService) Catchup_solicit(
	ctx context.Context,
	sreq proto.SignedCatchupBlocksRequest,
) (proto.SignedCatchupBlocksResult, error) {

	// check signatures
	if !protoutil.VerifyXdr(svc.reqM, sreq.Signer.Public_key, &sreq.Request, sreq.Signature) {
		return proto.SignedCatchupBlocksResult{}, fmt.Errorf("signature verification")
	}

	// compress spans to remove overlaps
	spans := make(protoutil.Spans[proto.BlockHeight], len(sreq.Request.Spans))
	for i, span := range sreq.Request.Spans {
		spans[i] = protoutil.Span[proto.BlockHeight]{From: span.From_block_height, To: span.To_block_height}
	}
	liblog.Infof(ctx, "catchup requested for spans: %v", spans)
	spans = protoutil.SumSpans(spans...)

	blocks := []proto.CatchupBlock{}
	for _, span := range spans {
		for h := span.From; h < span.To; h++ {
			if cb := svc.Lookup(ctx, h); cb != nil {
				blocks = append(blocks, *cb)
			}
		}
	}
	// log
	var heights bytes.Buffer
	for _, b := range blocks {
		fmt.Fprintf(&heights, "%d ", b.Consensus_value.Nomination.Block_height)
	}
	liblog.Infof(ctx, "serving catchup heights: [%s]", heights.String())

	result := proto.CatchupBlocksResult{
		Timestamp: protoutil.TimepointFromTime(svc.time.Now()),
		Blocks:    blocks,
	}

	return proto.SignedCatchupBlocksResult{
		Signer:    svc.signer.Signer(),
		Signature: svc.signer.SignXdr(svc.resultM, &result),
		Result:    result,
	}, nil
}
