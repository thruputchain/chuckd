package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/baton/sdk"
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

func (b *Baton) buildView(ctx context.Context) (*View, error) {

	v := &View{
		Stamp: b.handler.Now(),
	}

	dbBlock := ViewDbBlock{
		Tip: b.tip,
	}

	// if there are no scp blocks, we are done
	if len(b.externalized) == 0 {
		v.Blocks = []ViewBlock{dbBlock}
		v.Reqs = [][]sdk.ID64{nil}
		return v, nil
	}

	scpTopBlock := b.externalized[len(b.externalized)-1]
	scpTopHeight := scpTopBlock.Block_height
	if scpTopHeight < dbBlock.Tip.Height {
		return nil, ErrConsensusBelowDb
	}
	if scpTopHeight == dbBlock.Tip.Height {
		if scpTopBlock.Block_hash == dbBlock.BlockHash() {
			v.Blocks = []ViewBlock{dbBlock}
			v.Reqs = [][]sdk.ID64{nil}
			return v, nil
		} else {
			return nil, ErrDbConsensusDisagree
		}
	}

	n := int(scpTopHeight) + 1 - int(dbBlock.Tip.Height)
	if n > 10000 {
		return nil, ErrDbFarBehind
	}
	v.Blocks = make([]ViewBlock, n)
	v.Blocks[0] = dbBlock

	scpPrevHeight := proto.BlockHeight(0)
	for _, proposal := range b.externalized {
		// there can be gaps between scp blocks
		must.Assertf(ctx, proposal.Block_height > scpPrevHeight, "scp proposal heights not increasing")
		scpPrevHeight = proposal.Block_height
		//
		txs, _ := b.handler.LookupTxs(ctx, proposal.Transaction_hashes)
		v.Blocks[int(proposal.Block_height)-int(dbBlock.Tip.Height)] = ViewExternalizedBlock{
			Nomination: proposal,
			Txs:        txs,
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

	v.NextToCommit = v.nextBlockToCommit(b)

	return v, nil
}

func (b *Baton) fillBlocks(_ context.Context, v *View) error {
	for i := 1; i < len(v.Blocks); i++ {
		height := v.indexHeight(i)
		switch u := v.Blocks[i].(type) {
		case nil:
			if cb, ok := b.findCatchupBlockAtHeight(height); ok {
				v.Blocks[i] = ViewCatchupBlock{Candidate: cb, Linked: false} // linked will be computed later
			} else {
				v.Blocks[i] = ViewMissing{Height: height}
			}
		case ViewExternalizedBlock:
			// if the block is missing batches, catchup can still help
			if u.IsMissingTransactions() {
				if cb, ok := b.findCatchupBlockAtHeight(height); ok {
					// a catchup block in the same slot as an scp block can be linked immediately using the scp block
					if cb.BlockHash() == u.BlockHash() && cb.PrevBlockHash() == u.PrevBlockHash() {
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

func (b *Baton) chainBlocks(_ context.Context, v *View) error {
	prev := v.Blocks[0]
	for i := 1; i < len(v.Blocks); i++ {
		switch u := v.Blocks[i].(type) {
		case ViewMissing:
		case ViewCatchupBlock:
			if !prev.IsMissing() && prev.BlockHash() != u.PrevBlockHash() {
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
	v.Reqs = make([][]sdk.ID64, len(v.Blocks))
	for i := 1; i < len(v.Blocks); i++ {
		height := v.Blocks[0].BlockHeight() + proto.BlockHeight(i)
		v.Reqs[i] = b.findCatchupRequestsAtHeight(height)
	}
	return nil
}

type View struct {
	Stamp        time.Time
	Blocks       []ViewBlock
	Reqs         [][]sdk.ID64 // catchup requests already fired for this block
	NextToCommit *sdk.ValidatableBlock
}

func (v *View) ReadyToNominate() *sdk.Tip {
	if len(v.Blocks) != 1 {
		return nil
	}
	db := v.Blocks[0].(ViewDbBlock)
	return &db.Tip
}

func (v *View) NextBlockToCommit() *sdk.ValidatableBlock {
	return v.NextToCommit
}

func (v *View) nextBlockToCommit(b *Baton) *sdk.ValidatableBlock {

	if len(v.Blocks) < 2 || !v.Blocks[1].IsReadyToValidate() {
		return nil
	}
	vb := v.Blocks[1].ValidatableBlock()

	// db state
	db := v.Blocks[0].(ViewDbBlock)
	vb.DbTip = db.Tip

	// are we committing a block that we have already validated?
	if vtd, ok := b.validated[vb.BlockHash]; ok {
		vb.AlreadyValidatedNomination = vtd.Nomination
		vb.AlreadyValidatedBlockID = &vtd.BlockID
	}

	return vb
}

func (v *View) indexHeight(i int) proto.BlockHeight {
	return v.Blocks[0].BlockHeight() + proto.BlockHeight(i)
}

func (v *View) NeededUnrequestedSpans() Spans {
	spans := Spans{}
	var start *int
	for i := 1; i < len(v.Blocks); i++ {
		if len(v.Reqs[i]) > 0 {
			if start != nil {
				spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
				start = nil
			}
		} else {
			switch u := v.Blocks[i].(type) {
			case ViewMissing:
				if start == nil {
					j := i
					start = &j
				}
			case ViewExternalizedBlock:
				if u.IsReadyToValidate() {
					if start != nil {
						spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
						start = nil
					}
				} else {
					if start == nil {
						j := i
						start = &j
					}
				}
			case ViewCatchupBlock:
				if start != nil {
					spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
					start = nil
				}
			default:
				panic("unreachable")
			}
		}
	}
	if start != nil {
		spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(len(v.Blocks))})
		start = nil
	}
	return spans
}

type ViewBlock interface {
	BlockHash() proto.BlockHash
	BlockHeight() proto.BlockHeight
	PrevBlockHash() proto.BlockHash
	IsReadyToValidate() bool // enough data is available to run block validation
	IsMissing() bool
	IsSCP() bool
	IsCatchup() bool
	IsDb() bool
	ValidatableBlock() *sdk.ValidatableBlock
}

type ViewDbBlock struct {
	Tip sdk.Tip
}

func (x ViewDbBlock) BlockHash() proto.BlockHash              { return x.Tip.Hash }
func (x ViewDbBlock) BlockHeight() proto.BlockHeight          { return x.Tip.Height }
func (x ViewDbBlock) PrevBlockHash() proto.BlockHash          { panic("should not be called") }
func (x ViewDbBlock) IsReadyToValidate() bool                 { panic("should not be called") }
func (x ViewDbBlock) IsMissing() bool                         { return false }
func (x ViewDbBlock) IsSCP() bool                             { return false }
func (x ViewDbBlock) IsCatchup() bool                         { return false }
func (x ViewDbBlock) IsDb() bool                              { return true }
func (x ViewDbBlock) ValidatableBlock() *sdk.ValidatableBlock { panic("should not be called") }

type ViewCatchupBlock struct {
	Linked    bool // all catchup blocks between db and first scp chain hashes correctly
	Candidate CatchupBlock
}

func (x ViewCatchupBlock) BlockHash() proto.BlockHash     { return x.Candidate.BlockHash() }
func (x ViewCatchupBlock) BlockHeight() proto.BlockHeight { return x.Candidate.BlockHeight() }
func (x ViewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.Candidate.PrevBlockHash() }
func (x ViewCatchupBlock) IsReadyToValidate() bool        { return x.Linked }
func (x ViewCatchupBlock) IsMissing() bool                { return false }
func (x ViewCatchupBlock) IsSCP() bool                    { return false }
func (x ViewCatchupBlock) IsCatchup() bool                { return true }
func (x ViewCatchupBlock) IsDb() bool                     { return false }
func (x ViewCatchupBlock) ValidatableBlock() *sdk.ValidatableBlock {
	return &sdk.ValidatableBlock{
		FromCatchup:          true,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: x.Candidate.Txs,
	}
}

type ViewExternalizedBlock struct {
	Nomination *proto.Nomination
	Txs        []*proto.SignedTransaction
	Catchup    *CatchupBlock // a catchup block can be used if tx batches are missing
}

func (x ViewExternalizedBlock) BlockHash() proto.BlockHash     { return x.Nomination.Block_hash }
func (x ViewExternalizedBlock) BlockHeight() proto.BlockHeight { return x.Nomination.Block_height }
func (x ViewExternalizedBlock) PrevBlockHash() proto.BlockHash { return x.Nomination.Prev_block_hash }
func (x ViewExternalizedBlock) IsReadyToValidate() bool {
	// if a catchup block is placed inside an scp block, then it must link against it by design
	return (!x.IsMissingTransactions()) || x.Catchup != nil
}
func (x ViewExternalizedBlock) IsMissingTransactions() bool {
	return len(x.Txs) != len(x.Nomination.Transaction_hashes)
}
func (x ViewExternalizedBlock) IsMissing() bool { return false }
func (x ViewExternalizedBlock) IsSCP() bool     { return true }
func (x ViewExternalizedBlock) IsCatchup() bool { return false }
func (x ViewExternalizedBlock) IsDb() bool      { return false }
func (x ViewExternalizedBlock) ValidatableBlock() *sdk.ValidatableBlock {
	var txs []*proto.ProposedTransaction
	var fromCatchup bool
	var fromConsensus bool
	if x.IsMissingTransactions() {
		fromCatchup = true
		txs = protoutil.UnpackProposedTransactions(x.Catchup.Block.Proposed_transactions)
	} else {
		fromConsensus = true
		txs = make([]*proto.ProposedTransaction, len(x.Nomination.Transaction_hashes))
		for i := range txs {
			txs[i] = &proto.ProposedTransaction{
				Signed_tx:                      *x.Txs[i],
				Nondeterministic_proposal_data: x.Nomination.Transaction_nominations[i].Nondeterministic_proposal_data,
			}
		}
	}
	return &sdk.ValidatableBlock{
		FromCatchup:          fromCatchup,
		FromConsensus:        fromConsensus,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: txs,
	}
}

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) BlockHash() proto.BlockHash              { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight          { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash          { panic("must not call") }
func (x ViewMissing) IsReadyToValidate() bool                 { return false }
func (x ViewMissing) IsMissing() bool                         { return true }
func (x ViewMissing) IsSCP() bool                             { return false }
func (x ViewMissing) IsCatchup() bool                         { return false }
func (x ViewMissing) IsDb() bool                              { return false }
func (x ViewMissing) ValidatableBlock() *sdk.ValidatableBlock { panic("must not call") }
