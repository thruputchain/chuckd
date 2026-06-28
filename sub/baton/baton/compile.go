package baton

import (
	"context"
	"fmt"
	"math"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type Span = protoutil.Span[proto.BlockHeight]
type Spans = protoutil.Spans[proto.BlockHeight]

var (
	ErrConsensusBelowDb      = fmt.Errorf("scp is below db")     // unrecoverable
	ErrDbFarBehind           = fmt.Errorf("db far behind scp")   // must resort to cold start
	ErrFaultyCatchup         = fmt.Errorf("faulty catchup")      // must drop all catchup blocks and blacklist sources
	ErrDbConsensusDisagree   = fmt.Errorf("db and scp disagree") // unrecoverable
	ErrConsensusInconsistent = fmt.Errorf("scp blocks disagree") // unrecoverable
)

func (b *Baton) buildCleanView(ctx context.Context) (v *View) {
	for v == nil {
		var err error
		v, err = b.buildView(ctx)
		switch err {
		case nil:
			must.Assertf(ctx, v != nil, "bug")
		case ErrConsensusBelowDb: // unrecoverable
			liblog.Fatalf(ctx, "consensus height is below db; unrecoverable")
		case ErrDbFarBehind: // must resort to cold start
			liblog.Fatalf(ctx, "db is far behind consensus; needs a cold start")
		case ErrDbConsensusDisagree: // unrecoverable
			liblog.Fatalf(ctx, "consensus and db disagree; unrecoverable")
		case ErrConsensusInconsistent: // unrecoverable
			liblog.Fatalf(ctx, "consensus is inconsistent; unrecoverable")
		case ErrFaultyCatchup: // must drop all catchup blocks and blacklist sources
			liblog.Errorf(ctx, "purging faulty catchup")
			b.purgeCatchup(ctx) // go back and retry building the view
			v = nil
		default:
			panic("unexpected")
		}
	}
	return v
}

// getMaxCatchupHeight returns the maximum gap between the db and new externalizations
// that can be handled by the replay catchup block mechanism.
func (b *Baton) getMaxCatchupHeight() int {
	return b.maxCatchupHeight
}

func (b *Baton) buildView(ctx context.Context) (*View, error) {

	v := &View{Stamp: b.handler.Now()}

	dbBlock := ViewDbBlock{
		TipDb:  b.tipDb,
		TipExt: b.tipExt,
	}

	// if there are no scp blocks, we are done
	if len(b.externalized) == 0 {
		v.Blocks = []ViewBlock{dbBlock}
		v.Reqs = [][]batonkit.ID64{nil}
		return v, nil
	}

	scpTopExt := b.externalized[len(b.externalized)-1]
	scpTopHeight := scpTopExt.Height()
	if scpTopHeight < dbBlock.TipDb.Height {
		return nil, ErrConsensusBelowDb
	}
	if scpTopHeight == dbBlock.TipDb.Height {
		if scpTopExt.BlockHash() == dbBlock.BlockHash() &&
			scpTopExt.ConsensusBasisHash() == dbBlock.BasisHash() {
			v.Blocks = []ViewBlock{dbBlock}
			v.Reqs = [][]batonkit.ID64{nil}
			return v, nil
		} else {
			return nil, ErrDbConsensusDisagree
		}
	}

	if scpTopHeight == math.MaxInt64 {
		liblog.Fatalf(ctx, "top externalized height is too high: %d", scpTopHeight)
	}

	n := int(scpTopHeight) + 1 - int(dbBlock.TipDb.Height)
	if n > b.getMaxCatchupHeight() {
		return nil, ErrDbFarBehind
	}
	v.Blocks = make([]ViewBlock, n)
	v.Blocks[0] = dbBlock

	scpPrevHeight := proto.BlockHeight(0)
	for _, ext := range b.externalized {
		// there can be gaps between scp blocks
		must.Assertf(ctx, ext.Height() > scpPrevHeight, "externalization heights not increasing")
		scpPrevHeight = ext.Height()
		//
		if extConsensus, ok := ext.(externalizedBasis); ok {
			v.Blocks[int(ext.Height())-int(dbBlock.TipDb.Height)] = ViewExtBasisBlock{
				Ext:       extConsensus,
				Validated: b.getValidatedBlockID(ext.ConsensusValue()),
			}
		} else {
			extValue := ext.(externalizedValue)
			v.Blocks[int(ext.Height())-int(dbBlock.TipDb.Height)] = ViewExtValueBlock{
				Ext: extValue,
			}
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
	if err := b.compileCatchupRequests(ctx, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (b *Baton) fillBlocks(_ context.Context, v *View) error {
	for i := 1; i < len(v.Blocks); i++ {
		height := v.indexHeight(i)
		switch u := v.Blocks[i].(type) {
		case nil:
			if ca, ok := b.findCatchupBlockAtHeight(height); ok {
				v.Blocks[i] = ViewCatchupBlock{
					Candidate: ca,
					Linked:    false, // linked will be computed later
					Validated: b.getValidatedBlockID(ca.ConsensusValue()),
				}
			} else {
				v.Blocks[i] = ViewMissing{Height: height}
			}
		case ViewExtBasisBlock:
			// if the block has not been validated already, catchup can help
			if u.Validated == nil {
				if ca, ok := b.findCatchupBlockAtHeight(height); ok {
					// a catchup block in the same slot as an scp block can be linked immediately using the scp block
					if protoutil.ValueEquals(ca.ConsensusValue(), BlockValue(u)) {
						u.Catchup = &ca
						v.Blocks[i] = u
					} else {
						return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
					}
				}
			}
		case ViewExtValueBlock:
			if ca, ok := b.findCatchupBlockAtHeight(height); ok {
				// a catchup block in the same slot as an scp block can be linked immediately using the scp block
				if protoutil.ValueEquals(ca.ConsensusValue(), BlockValue(u)) {
					u.Catchup = &ca
					v.Blocks[i] = u
				} else {
					return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
				}
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

// chainBlocks verifies that each block does not disagree with the previous one, if available.
func (b *Baton) chainBlocks(_ context.Context, v *View) error {
	prev := v.Blocks[0]
	for i := 1; i < len(v.Blocks); i++ {
		switch u := v.Blocks[i].(type) {
		case ViewMissing:
		case ViewCatchupBlock:
			if !prev.IsMissing() &&
				(prev.BlockHash() != u.PrevBlockHash() ||
					prev.BasisHash() != u.PrevBasisHash()) {
				return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
			}
		case ViewExtBasisBlock, ViewExtValueBlock:
			switch {
			case prev.IsMissing():
			case prev.IsCatchup():
				if u.PrevBlockHash() != prev.BlockHash() || u.PrevBasisHash() != prev.BasisHash() {
					return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
				}
			case prev.IsDb():
				if u.PrevBlockHash() != prev.BlockHash() || u.PrevBasisHash() != prev.BasisHash() {
					return ErrDbConsensusDisagree // unrecoverable
				}
			case prev.IsExtBasis() || prev.IsExtValue():
				if u.PrevBlockHash() != prev.BlockHash() || u.PrevBasisHash() != prev.BasisHash() {
					return ErrConsensusInconsistent // unrecoverable
				}
			default:
				panic("unreachable")
			}
		default:
			panic("unreachable")
		}
		prev = v.Blocks[i]
	}
	return nil
}

func (b *Baton) linkBlocks(ctx context.Context, v *View) error {
	var start *int // first catchup block in a contiguous sequence of such
	for i := 1; i < len(v.Blocks); i++ {
		switch v.Blocks[i].(type) {
		case ViewMissing:
			if start != nil { // the run of catchup blocks has a missing block on top, hence it is not linked
				start = nil
			}
		case ViewCatchupBlock:
			if start == nil { // this is the first catchup block in a contiguous sequence
				j := i // don't take a pointer to i, it changes
				start = &j
			}
		case ViewExtBasisBlock, ViewExtValueBlock:
			if start != nil { // this scp block tops a contiguous sequence of catchup blocks; mark them linked
				for k := *start; k < i; k++ {
					cb := v.Blocks[k].(ViewCatchupBlock)
					cb.Linked = true
					v.Blocks[k] = cb
				}
				start = nil
			}
		default:
			panic("unreachable")
		}
	}
	must.Assertf(ctx, start == nil, "view cannot be topped by a catchup block")
	return nil
}

func (b *Baton) compileCatchupRequests(_ context.Context, v *View) error {
	v.Reqs = make([][]batonkit.ID64, len(v.Blocks))
	for i := 1; i < len(v.Blocks); i++ {
		height := v.Blocks[0].BlockHeight() + proto.BlockHeight(i)
		v.Reqs[i] = b.findCatchupRequestsAtHeight(height)
	}
	return nil
}
