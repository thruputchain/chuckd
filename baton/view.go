package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/xid"
)

var (
	ErrFaultyCatchup   = fmt.Errorf("faulty catchup")
	ErrDbSCPDisagree   = fmt.Errorf("db and scp disagree") // unrecoverable
	ErrSCPInconsistent = fmt.Errorf("scp blocks disagree") // unrecoverable
)

func (b *Baton) buildView(ctx context.Context) (*view, error) {

	b.lock()
	defer b.unlock()

	v := &view{
		stamp: b.handler.Now(),
	}

	dbBlock := viewDbBlock{
		hash:   b.db.tipHash,
		height: b.db.tipHeight,
	}

	// if there are no scp blocks, we are done
	if b.scp.prop.IsEmpty() {
		v.blocks = []viewBlock{dbBlock}
		v.reqs = [][]xid.ID{nil}
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

	// try to fill every slot independently, one at a time
	if err := b.fillBlocks(ctx, v); err != nil {
		return nil, err
	}

	// for every catchup block, make sure hashes stack below and above (if available) correctly
	if err := b.chainBlocks(ctx, v); err != nil {
		return nil, err
	}

	// compute which catchup blocks form a "linked" chain above and below
	if err := b.linkBlocks(ctx, v); err != nil {
		return nil, err
	}

	// compute pending requests per block
	if err := b.compileRequests(ctx, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (b *Baton) fillBlocks(ctx context.Context, v *view) error {
	for i := 1; i < len(v.blocks); i++ {
		height := v.blocks[0].BlockHeight() + proto.BlockHeight(i)
		switch u := v.blocks[i].(type) {
		case nil:
			if cb, ok := b.findCatchupBlocksAtHeight(height); ok {
				v.blocks[i] = viewCatchupBlock{candidate: cb, linked: false} // linked will be computed later
			} else {
				v.blocks[i] = viewMissing{height: height}
			}
		case viewSCPBlock:
			// if the block is missing batches, catchup can still help
			if u.IsMissingTransactions() {
				if cb, ok := b.findCatchupBlocksAtHeight(height); ok {
					// a catchup block in the same slot as an scp block can be linked immediately using the scp block
					if cb.BlockHash() == u.BlockHash() && cb.PrevBlockHash() == u.PrevBlockHash() {
						u.catchup = &cb
					} else {
						return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
					}
				}
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

func (b *Baton) chainBlocks(ctx context.Context, v *view) error {
	prev := v.blocks[0]
	for i := 1; i < len(v.blocks); i++ {
		switch u := v.blocks[i].(type) {
		case viewMissing:
		case viewCatchupBlock:
			if !prev.IsMissing() && prev.BlockHash() != u.PrevBlockHash() {
				return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
			}
		case viewSCPBlock:
			switch {
			case prev.IsMissing():
			case prev.IsCatchup():
				if u.PrevBlockHash() != prev.BlockHash() {
					return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
				}
			case prev.IsDb():
				if u.PrevBlockHash() != prev.BlockHash() {
					return ErrDbSCPDisagree // unrecoverable
				}
			case prev.IsSCP():
				if u.PrevBlockHash() != prev.BlockHash() {
					return ErrSCPInconsistent // unrecoverable
				}
			default:
				panic("unreachable")
			}
		default:
			panic("unreachable")
		}
		prev = v.blocks[i]
	}
	return nil
}

func (b *Baton) linkBlocks(ctx context.Context, v *view) error {
	var start *int // first catchup block in a contiguous sequence of such
	for i := 1; i < len(v.blocks); i++ {
		switch v.blocks[i].(type) {
		case viewMissing:
			if start != nil { // the run of catchup blocks has a missing block on top, hence it is not linked
				start = nil
			}
		case viewCatchupBlock:
			if start == nil { // this is the first catchup block in a contiguous sequence
				j := i // don't take a pointer to i, it changes
				start = &j
			}
		case viewSCPBlock:
			if start != nil { // this scp block tops a contiguous sequence of catchup blocks; mark them linked
				for k := *start; k < i; k++ {
					cb := v.blocks[i].(viewCatchupBlock)
					cb.linked = true
					v.blocks[i] = cb
				}
				start = nil
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

func (b *Baton) compileRequests(ctx context.Context, v *view) error {
	v.reqs = make([][]xid.ID, len(v.blocks))
	for i := 1; i < len(v.blocks); i++ {
		height := v.blocks[0].BlockHeight() + proto.BlockHeight(i)
		v.reqs[i] = b.findCatchupRequestsAtHeight(height)
	}
	return nil
}

type view struct {
	stamp  time.Time
	blocks []viewBlock
	reqs   [][]xid.ID // catchup requests already fired for this block
}

type viewBlock interface {
	BlockHash() proto.BlockHash
	BlockHeight() proto.BlockHeight
	PrevBlockHash() proto.BlockHash
	IsReadyToValidate() bool // enough data is available to run block validation
	IsMissing() bool
	IsSCP() bool
	IsCatchup() bool
	IsDb() bool
}

type viewDbBlock struct {
	hash   proto.BlockHash
	height proto.BlockHeight
}

func (x viewDbBlock) BlockHash() proto.BlockHash     { return x.hash }
func (x viewDbBlock) BlockHeight() proto.BlockHeight { return x.height }
func (x viewDbBlock) PrevBlockHash() proto.BlockHash { panic("should not be called") }
func (x viewDbBlock) IsReadyToValidate() bool        { return true }
func (x viewDbBlock) IsMissing() bool                { return false }
func (x viewDbBlock) IsSCP() bool                    { return false }
func (x viewDbBlock) IsCatchup() bool                { return false }
func (x viewDbBlock) IsDb() bool                     { return true }

type viewCatchupBlock struct {
	linked    bool // all catchup blocks between db and first scp chain hashes correctly
	candidate catchupBlock
}

func (x viewCatchupBlock) BlockHash() proto.BlockHash     { return x.candidate.BlockHash() }
func (x viewCatchupBlock) BlockHeight() proto.BlockHeight { return x.candidate.BlockHeight() }
func (x viewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.candidate.PrevBlockHash() }
func (x viewCatchupBlock) IsReadyToValidate() bool        { return x.linked }
func (x viewCatchupBlock) IsMissing() bool                { return false }
func (x viewCatchupBlock) IsSCP() bool                    { return false }
func (x viewCatchupBlock) IsCatchup() bool                { return true }
func (x viewCatchupBlock) IsDb() bool                     { return false }

type viewSCPBlock struct {
	proto   *proto.Proposal
	batches map[proto.BatchHash][]proto.ProposedTransaction
	catchup *catchupBlock // a catchup block can be used if tx batches are missing
}

func (x viewSCPBlock) BlockHash() proto.BlockHash     { return x.proto.Block_hash }
func (x viewSCPBlock) BlockHeight() proto.BlockHeight { return x.proto.Block_height }
func (x viewSCPBlock) PrevBlockHash() proto.BlockHash { return x.proto.Prev_block_hash }
func (x viewSCPBlock) IsReadyToValidate() bool {
	// if a catchup block is placed inside an scp block, then it must link against it by design
	return (!x.IsMissingTransactions()) || x.catchup != nil
}
func (x viewSCPBlock) IsMissingTransactions() bool {
	return len(x.batches) != len(x.proto.Proposed_transaction_batches)
}
func (x viewSCPBlock) IsMissing() bool { return false }
func (x viewSCPBlock) IsSCP() bool     { return true }
func (x viewSCPBlock) IsCatchup() bool { return false }
func (x viewSCPBlock) IsDb() bool      { return false }

type viewMissing struct {
	height proto.BlockHeight
}

func (x viewMissing) BlockHash() proto.BlockHash     { panic("must not call") }
func (x viewMissing) BlockHeight() proto.BlockHeight { return x.height }
func (x viewMissing) PrevBlockHash() proto.BlockHash { panic("must not call") }
func (x viewMissing) IsReadyToValidate() bool        { return false }
func (x viewMissing) IsMissing() bool                { return true }
func (x viewMissing) IsSCP() bool                    { return false }
func (x viewMissing) IsCatchup() bool                { return false }
func (x viewMissing) IsDb() bool                     { return false }
