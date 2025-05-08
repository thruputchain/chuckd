package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type committed struct{}

type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []proto.ProposedTransaction
}

func (b *Baton) startValidateAndCommit(ctx context.Context) {
	for {
		if err := b.validateAndCommit(ctx, <-b.commit); err != nil {
			log.Errorf("validate and commit: %v", err)
		}
	}
}

func (b *Baton) validateAndCommit(ctx context.Context, vb *ValidatableBlock) error {

	b.db.lk.Lock()
	tipID := b.db.tipID
	tipHeight := b.db.tipHeight
	b.db.lk.Unlock()

	if tipHeight >= vb.BlockHeight {
		// a block may be submitted for validation twice under a very unlikely race condition
		return nil
	}

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	nextID := b.handler.GenerateBlockID(ctx)

	// db open snapshot
	rpcErr, err := db.Dbcoord_open_snapshot(ctx, tipID, nextID)
	if err != nil {
		return err
	}
	if rpcErr != proto.KRPCNone {
		return fmt.Errorf("db open snapshot rpc error: %v", rpcErr)
	}

	// connect to available executors
	execs, err := b.handler.DialExecs(ctx)
	if err != nil {
		return err
	}
	for _, exec := range execs {
		defer exec.Close()
	}

	// validate all transactions
	txCh := make(chan proto.ProposedTransaction, len(vb.ProposedTransactions))
	for _, tx := range vb.ProposedTransactions {
		txCh <- tx
	}
	close(txCh)

	validated := make(chan error, len(execs))
	for _, exec := range execs {
		go func(exec ExecCoordConn) {

			// executor open validation block
			rpcErr, err = exec.Executorcoord_open_validation_block(ctx, nextID)
			if err != nil {
				validated <- err
				return
			}
			if rpcErr != proto.KRPCNone {
				validated <- fmt.Errorf("executor open validation block rpc error: %v", rpcErr)
				return
			}

			for tx := range txCh {
				rpcErr, err := exec.Executorcoord_validate_transaction(ctx, nextID, tx)
				if err != nil {
					validated <- err
					return
				}
				// XXX: validation failures are treated differently than retriable error conditions;
				// confirm with Geoff this is the only error indicating an invalid block proposal
				if rpcErr == proto.KRPCValidationFailure {
					// update the state to prevent this vb being retried
					b.invalidProposalExternalized(ctx, vb)
					validated <- fmt.Errorf("encountered invalid proposal")
					return
				} else if rpcErr != proto.KRPCNone {
					validated <- fmt.Errorf("executor validate transaction rpc error: %v", rpcErr)
					return
				}
			}

			// executor finish validation block
			err = exec.Executorcoord_finish_block(ctx, nextID)
			if err != nil {
				validated <- fmt.Errorf("executor finish error: %v", err)
				return
			}

			validated <- nil

		}(exec)
	}
	var verr error
	for range execs {
		if err := <-validated; err != nil {
			verr = err
		}
	}
	if verr != nil {
		// db abandon snapshot
		db.Dbcoord_abandon_snapshot(ctx, nextID)
		return fmt.Errorf("validation error: %v", verr)
	}

	// XXX: db must have a way of returning the current hash of the block, so we can verify

	// db commit snapshot
	rpcErr, err = db.Dbcoord_commit_snapshot(ctx, nextID)
	if err != nil {
		return err
	}
	if rpcErr != proto.KRPCNone {
		return fmt.Errorf("db commit snapshot rpc error: %v", rpcErr)
	}

	// notify baton
	defer func() { b.committed <- committed{} }()

	// update state
	b.lock()
	defer b.unlock()

	// only this function writes to the state, and sequentially
	b.db.tipHash = vb.BlockHash
	b.db.tipHeight += 1
	b.db.tipID = nextID

	// purge catchup or consensus blocks no higher than the newly committed one
	b.purgeBelowTip(ctx)

	return nil
}

func (b *Baton) purgeBelowTip(_ context.Context) {

	// purge catchup blocks
	var catchup catchupBlocks
	for _, cb := range b.catchup.blocks {
		if cb.BlockHeight() > b.db.tipHeight {
			catchup = append(catchup, cb)
		}
	}
	b.catchup.blocks = catchup

	// purge consensus blocks
	var props proposals
	for _, p := range b.consensus.prop {
		if p.BlockHeight() > b.db.tipHeight {
			props = append(props, p)
		}
	}
	b.consensus.prop = props

	// purge nomination blocks
	panic("XXX")

}

func (b *Baton) invalidProposalExternalized(ctx context.Context, vb *ValidatableBlock) {
	if vb.FromCatchup {
		b.purgeCatchup(ctx)
	}
	if vb.FromConsensus {
		must.Panic(ctx, fmt.Errorf("invalid proposal externalized"))
	}
}
