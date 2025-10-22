package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
			liblog.Fatalf("consensus height is below db; unrecoverable")
		case ErrDbFarBehind: // must resort to cold start
			liblog.Fatalf("db is far behind consensus; needs a cold start")
		case ErrDbConsensusDisagree: // unrecoverable
			liblog.Fatalf("consensus and db disagree; unrecoverable")
		case ErrConsensusInconsistent: // unrecoverable
			liblog.Fatalf("consensus is inconsistent; unrecoverable")
		case ErrFaultyCatchup: // must drop all catchup blocks and blacklist sources
			liblog.Errorf("purging faulty catchup")
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
			scpTopExt.ConsensusValueHash() == dbBlock.ConsensusValueHash() {
			v.Blocks = []ViewBlock{dbBlock}
			v.Reqs = [][]batonkit.ID64{nil}
			return v, nil
		} else {
			return nil, ErrDbConsensusDisagree
		}
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
		v.Blocks[int(ext.Height())-int(dbBlock.TipDb.Height)] = ViewExternalizedBlock{
			Ext:       ext,
			Validated: b.getValidated(ext.ConsensusValue()),
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

func (b *Baton) fillBlocks(_ context.Context, v *View) error {
	for i := 1; i < len(v.Blocks); i++ {
		height := v.indexHeight(i)
		switch u := v.Blocks[i].(type) {
		case nil:
			if cb, ok := b.findCatchupBlockAtHeight(height); ok {
				v.Blocks[i] = ViewCatchupBlock{
					Candidate: cb,
					Linked:    false, // linked will be computed later
					Validated: b.getValidated(&cb.Block.Consensus_value),
				}
			} else {
				v.Blocks[i] = ViewMissing{Height: height}
			}
		case ViewExternalizedBlock:
			// if the block has not been validated already, catchup can help
			if u.Validated == nil {
				if cb, ok := b.findCatchupBlockAtHeight(height); ok {
					// a catchup block in the same slot as an scp block can be linked immediately using the scp block
					if cb.ConsensusValueHash() == u.ConsensusValueHash() {
						u.Catchup = &cb
						v.Blocks[i] = u
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

// chainBlocks verifies that each block does not disagree with the previous one, if available.
// XXX: consensus value must include the hash of the previous consensus value to enable correct chaining here
func (b *Baton) chainBlocks(_ context.Context, v *View) error {
	prev := v.Blocks[0]
	for i := 1; i < len(v.Blocks); i++ {
		switch u := v.Blocks[i].(type) {
		case ViewMissing:
		case ViewCatchupBlock:
			/* XXX: check previous consensus value hashes */
			/* b.hashCV(XXX) != u.Candidate.Block.Consensus_value.Prev_cv_hash*/
			if !prev.IsMissing() && (prev.BlockHash() != u.PrevBlockHash()) {
				return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
			}
		case ViewExternalizedBlock:
			switch {
			case prev.IsMissing():
			case prev.IsCatchup():
				if u.PrevBlockHash() != prev.BlockHash() {
					return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
				}
			case prev.IsDb():
				if u.PrevBlockHash() != prev.BlockHash() {
					return ErrDbConsensusDisagree // unrecoverable
				}
			case prev.IsSCP():
				if u.PrevBlockHash() != prev.BlockHash() {
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
		case ViewExternalizedBlock:
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

func (b *Baton) compileRequests(_ context.Context, v *View) error {
	v.Reqs = make([][]batonkit.ID64, len(v.Blocks))
	for i := 1; i < len(v.Blocks); i++ {
		height := v.Blocks[0].BlockHeight() + proto.BlockHeight(i)
		v.Reqs[i] = b.findCatchupRequestsAtHeight(height)
	}
	return nil
}
