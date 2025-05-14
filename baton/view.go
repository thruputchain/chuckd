package baton

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

var (
	ErrConsensusBelowDb      = fmt.Errorf("scp is below db")     // unrecoverable
	ErrDbFarBehind           = fmt.Errorf("db far behind scp")   // must resort to cold start
	ErrFaultyCatchup         = fmt.Errorf("faulty catchup")      // must drop all catchup blocks and blacklist sources
	ErrDbConsensusDisagree   = fmt.Errorf("db and scp disagree") // unrecoverable
	ErrConsensusInconsistent = fmt.Errorf("scp blocks disagree") // unrecoverable
)

func (b *Baton) buildView(ctx context.Context) (*view, error) {

	b.lock()
	defer b.unlock()

	v := &view{
		stamp: b.handler.Now(),
	}

	dbBlock := viewDbBlock{
		tip: b.db.tip,
	}

	// if there are no scp blocks, we are done
	if len(b.consensus.ext) == 0 {
		v.blocks = []viewBlock{dbBlock}
		v.reqs = [][]ID64{nil}
		return v, nil
	}

	scpTopBlock := b.consensus.ext[len(b.consensus.ext)-1]
	scpTopHeight := scpTopBlock.Block_height
	if scpTopHeight < dbBlock.tip.Height {
		return nil, ErrConsensusBelowDb
	}
	if scpTopHeight == dbBlock.tip.Height {
		if scpTopBlock.Block_hash == dbBlock.BlockHash() {
			v.blocks = []viewBlock{dbBlock}
			v.reqs = [][]ID64{nil}
			return v, nil
		} else {
			return nil, ErrDbConsensusDisagree
		}
	}

	n := int(scpTopHeight) + 1 - int(dbBlock.tip.Height)
	if n > 10000 {
		return nil, ErrDbFarBehind
	}
	v.blocks = make([]viewBlock, n)
	v.blocks[0] = dbBlock

	scpPrevHeight := proto.BlockHeight(0)
	for _, proposal := range b.consensus.ext {
		// there can be gaps between scp blocks
		must.Assertf(ctx, proposal.Block_height > scpPrevHeight, "scp proposal heights not increasing")
		scpPrevHeight = proposal.Block_height
		//
		v.blocks[int(proposal.Block_height)-int(dbBlock.tip.Height)] = viewExternalizedBlock{
			proposal: proposal,
			batches:  b.fetchBatches(ctx, proposal.Proposed_transaction_batches),
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

	v.nextToCommit = v.nextBlockToCommit(b)

	return v, nil
}

func (b *Baton) fillBlocks(_ context.Context, v *view) error {
	for i := 1; i < len(v.blocks); i++ {
		height := v.indexHeight(i)
		switch u := v.blocks[i].(type) {
		case nil:
			if cb, ok := b.findCatchupBlockAtHeight(height); ok {
				v.blocks[i] = viewCatchupBlock{candidate: cb, linked: false} // linked will be computed later
			} else {
				v.blocks[i] = viewMissing{height: height}
			}
		case viewExternalizedBlock:
			// if the block is missing batches, catchup can still help
			if u.IsMissingTransactions() {
				if cb, ok := b.findCatchupBlockAtHeight(height); ok {
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

func (b *Baton) chainBlocks(_ context.Context, v *view) error {
	prev := v.blocks[0]
	for i := 1; i < len(v.blocks); i++ {
		switch u := v.blocks[i].(type) {
		case viewMissing:
		case viewCatchupBlock:
			if !prev.IsMissing() && prev.BlockHash() != u.PrevBlockHash() {
				return ErrFaultyCatchup // the caller must purge all catchup blocks and rerun view
			}
		case viewExternalizedBlock:
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
		case viewExternalizedBlock:
			if start != nil { // this scp block tops a contiguous sequence of catchup blocks; mark them linked
				for k := *start; k < i; k++ {
					cb := v.blocks[k].(viewCatchupBlock)
					cb.linked = true
					v.blocks[k] = cb
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

func (b *Baton) compileRequests(_ context.Context, v *view) error {
	v.reqs = make([][]ID64, len(v.blocks))
	for i := 1; i < len(v.blocks); i++ {
		height := v.blocks[0].BlockHeight() + proto.BlockHeight(i)
		v.reqs[i] = b.findCatchupRequestsAtHeight(height)
	}
	return nil
}

type view struct {
	stamp        time.Time
	blocks       []viewBlock
	reqs         [][]ID64 // catchup requests already fired for this block // XXX: never used
	nextToCommit *ValidatableBlock
}

func (v *view) ReadyToNominate() *nominate {
	if len(v.blocks) != 1 {
		return nil
	}
	db := v.blocks[0].(viewDbBlock)
	return &nominate{
		tip: db.tip,
	}
}

func (v *view) NextBlockToCommit() *ValidatableBlock {
	return v.nextToCommit
}

func (v *view) nextBlockToCommit(b *Baton) *ValidatableBlock {

	if len(v.blocks) < 2 || !v.blocks[1].IsReadyToValidate() {
		return nil
	}
	vb := v.blocks[1].ValidatableBlock()

	// db state
	db := v.blocks[0].(viewDbBlock)
	vb.DbTip = db.tip

	// are we committing a block that we have already validated?
	if vtd, ok := b.consensus.validated[vb.BlockHash]; ok {
		vb.AlreadyValidatedProposal = vtd.Proposal
		vb.AlreadyValidatedBlockID = &vtd.BlockID
	}

	return vb
}

func (v *view) indexHeight(i int) proto.BlockHeight {
	return v.blocks[0].BlockHeight() + proto.BlockHeight(i)
}

func (v *view) NeededUnrequestedSpans() Spans {
	spans := Spans{}
	var start *int
	for i := 1; i < len(v.blocks); i++ {
		if len(v.reqs[i]) > 0 {
			if start != nil {
				spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
				start = nil
			}
		} else {
			switch u := v.blocks[i].(type) {
			case viewMissing:
				if start == nil {
					j := i
					start = &j
				}
			case viewExternalizedBlock:
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
			case viewCatchupBlock:
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
		spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(len(v.blocks))})
		start = nil
	}
	return spans
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
	ValidatableBlock() *ValidatableBlock
}

type viewDbBlock struct {
	tip Tip
}

func (x viewDbBlock) BlockHash() proto.BlockHash          { return x.tip.Hash }
func (x viewDbBlock) BlockHeight() proto.BlockHeight      { return x.tip.Height }
func (x viewDbBlock) PrevBlockHash() proto.BlockHash      { panic("should not be called") }
func (x viewDbBlock) IsReadyToValidate() bool             { panic("should not be called") }
func (x viewDbBlock) IsMissing() bool                     { return false }
func (x viewDbBlock) IsSCP() bool                         { return false }
func (x viewDbBlock) IsCatchup() bool                     { return false }
func (x viewDbBlock) IsDb() bool                          { return true }
func (x viewDbBlock) ValidatableBlock() *ValidatableBlock { panic("should not be called") }

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
func (x viewCatchupBlock) ValidatableBlock() *ValidatableBlock {
	return &ValidatableBlock{
		FromCatchup:          true,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: x.candidate.txs,
	}
}

type viewExternalizedBlock struct {
	proposal *proto.Proposal
	batches  map[proto.BatchHash][]*proto.ProposedTransaction
	catchup  *catchupBlock // a catchup block can be used if tx batches are missing
}

func (x viewExternalizedBlock) BlockHash() proto.BlockHash     { return x.proposal.Block_hash }
func (x viewExternalizedBlock) BlockHeight() proto.BlockHeight { return x.proposal.Block_height }
func (x viewExternalizedBlock) PrevBlockHash() proto.BlockHash { return x.proposal.Prev_block_hash }
func (x viewExternalizedBlock) IsReadyToValidate() bool {
	// if a catchup block is placed inside an scp block, then it must link against it by design
	return (!x.IsMissingTransactions()) || x.catchup != nil
}
func (x viewExternalizedBlock) IsMissingTransactions() bool {
	return len(x.batches) != len(x.proposal.Proposed_transaction_batches)
}
func (x viewExternalizedBlock) IsMissing() bool { return false }
func (x viewExternalizedBlock) IsSCP() bool     { return true }
func (x viewExternalizedBlock) IsCatchup() bool { return false }
func (x viewExternalizedBlock) IsDb() bool      { return false }
func (x viewExternalizedBlock) ValidatableBlock() *ValidatableBlock {
	var txs []*proto.ProposedTransaction
	var fromCatchup bool
	var fromConsensus bool
	if x.IsMissingTransactions() {
		fromCatchup = true
		txs = protoutil.RepackageProposedTransactions(x.catchup.block.Proposed_transactions)
	} else {
		fromConsensus = true
		for _, batchTxs := range x.batches {
			txs = append(txs, batchTxs...)
		}
	}
	return &ValidatableBlock{
		FromCatchup:          fromCatchup,
		FromConsensus:        fromConsensus,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: txs,
	}
}

type viewMissing struct {
	height proto.BlockHeight
}

func (x viewMissing) BlockHash() proto.BlockHash          { panic("must not call") }
func (x viewMissing) BlockHeight() proto.BlockHeight      { return x.height }
func (x viewMissing) PrevBlockHash() proto.BlockHash      { panic("must not call") }
func (x viewMissing) IsReadyToValidate() bool             { return false }
func (x viewMissing) IsMissing() bool                     { return true }
func (x viewMissing) IsSCP() bool                         { return false }
func (x viewMissing) IsCatchup() bool                     { return false }
func (x viewMissing) IsDb() bool                          { return false }
func (x viewMissing) ValidatableBlock() *ValidatableBlock { panic("must not call") }
