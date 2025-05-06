package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) buildView(ctx context.Context) (*view, error) {

	b.lock()
	defer b.unlock()

	v := &view{
		stamp: b.handler.Now(),
	}

	// place db blocks

	dbBlock := viewDbBlock{
		hash:   b.db.tipHash,
		height: b.db.tipHeight,
	}

	// place scp blocks

	if b.scp.prop.IsEmpty() {
		v.blocks = []viewBlock{dbBlock}
		return v, nil
	}

	scpTopHeight := b.scp.prop[len(b.scp.prop)-1].BlockHeight()
	if scpTopHeight <= dbBlock.height {
		log.Errorf("scp height is below db")
		v.blocks = []viewBlock{dbBlock}
		return v, nil
	}

	n := int(scpTopHeight) + 1 - int(dbBlock.height)
	if n > 10000 {
		return nil, fmt.Errorf("db is too far behind scp")
	}
	v.blocks = make([]viewBlock, n)
	v.blocks[0] = dbBlock

	scpPrevHeight := proto.BlockHeight(0)
	for _, prop := range b.scp.prop {
		// there can be gaps between scp blocks
		must.Assertf(ctx, prop.BlockHeight() > scpPrevHeight, "scp proposal heights not increasing")
		scpPrevHeight = prop.BlockHeight()
		//
		v.blocks[int(prop.BlockHeight())-int(dbBlock.height)] = viewSCPBlock{
			proto:   prop.Proto(),
			batches: b.fetchBatches(ctx, prop.Proto().Proposed_transaction_batches),
		}
	}

	// place catchup blocks

	// mark them computable only if they hash-link on both sides
	// if catchup blocks fill slot gaps and don't link up, must throw away all catchup blocks
	XXX

	// try to fill every slot independently, one at a time, at the end check connectivity
	for i := 1; i < n; i++ {
		height := dbBlock.height + proto.BlockHeight(i)
		switch u := v.blocks[i].(type) {
		case nil:
			if cb, ok := b.findCatchupAtHeight(height); ok {
				v.blocks[i] = viewCatchupBlock{candidate: cb, linked: false} // XXX: linked will be computed later
			} else {
				v.blocks[i] = viewMissing{height: height}
			}
		case viewSCPBlock:
			// if the block is missing batches, catchup can still help
			// a catchup block in the same slot as an scp block can be linked immediately using the scp block
			XXX
		default:
			panic("unreachable")
		}
	}
	// for every catchup block, make sure hashes link below and above (if available)
	XXX

	// compute v.reqs
	XXX
}

type view struct {
	stamp  time.Time
	blocks []viewBlock
	reqs   []int // number of catchup requests already fired for this block
}

type viewBlock interface {
	BlockHash() proto.BlockHash
	BlockHeight() proto.BlockHeight
	PrevBlockHash() proto.BlockHash
	IsComputable() bool // enough data is available to run block validation
}

type viewDbBlock struct {
	hash   proto.BlockHash
	height proto.BlockHeight
}

func (x viewDbBlock) BlockHash() proto.BlockHash     { return x.hash }
func (x viewDbBlock) BlockHeight() proto.BlockHeight { return x.height }
func (x viewDbBlock) PrevBlockHash() proto.BlockHash { panic("should not be called") }
func (x viewDbBlock) IsComputable() bool             { return true }

type viewCatchupBlock struct {
	linked    bool // all catchup blocks between db and first scp chain hashes correctly
	candidate catchupBlock
}

func (x viewCatchupBlock) BlockHash() proto.BlockHash     { return x.proto.Block_hash }
func (x viewCatchupBlock) BlockHeight() proto.BlockHeight { return x.proto.Block_height }
func (x viewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.proto.Prev_block_hash }
func (x viewCatchupBlock) IsComputable() bool             { return x.linked }

type viewSCPBlock struct {
	proto   *proto.Proposal
	batches map[proto.BatchHash][]proto.ProposedTransaction
	catchup *catchupBlock // a catchup block can be used if tx batches are missing
}

func (x viewSCPBlock) BlockHash() proto.BlockHash     { return x.proto.Block_hash }
func (x viewSCPBlock) BlockHeight() proto.BlockHeight { return x.proto.Block_height }
func (x viewSCPBlock) PrevBlockHash() proto.BlockHash { return x.proto.Prev_block_hash }
func (x viewSCPBlock) IsComputable() bool {
	return len(x.batches) == len(x.proto.Proposed_transaction_batches)
}

type viewMissing struct {
	height proto.BlockHeight
}

func (x viewMissing) BlockHash() proto.BlockHash     { panic("must not call") }
func (x viewMissing) BlockHeight() proto.BlockHeight { return x.height }
func (x viewMissing) PrevBlockHash() proto.BlockHash { panic("must not call") }
func (x viewMissing) IsComputable() bool             { return false }
