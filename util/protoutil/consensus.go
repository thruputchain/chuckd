package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func HashProposalTxsSlow(txs *proto.ProposalTxs) proto.ProposalTxsHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, txs)
	return proto.ProposalTxsHash(h.Sum(nil))
}

func HashProposalTxsFast(txs *proto.ProposalTxs) proto.ProposalTxsHash {
	txsBase := *txs
	txsBase.Hash = proto.ParallelBatch{}
	txsBase.Nondeterministic_data = proto.ParallelBatch{}

	var hashBase proto.SHA256Hash
	var hashTxsHash proto.SHA256Hash
	var hashTxsND proto.SHA256Hash

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		hashBase = HashProposalTxsSlow(&txsBase)
	}()
	go func() {
		defer wg.Done()
		hashTxsHash = xdrutil.HashParallelBatch(txs.Hash)
	}()
	go func() {
		defer wg.Done()
		hashTxsND = xdrutil.HashParallelBatch(txs.Nondeterministic_data)
	}()
	wg.Wait()
	for i := range hashBase {
		hashBase[i] ^= hashTxsHash[i] ^ hashTxsND[i]
	}
	return hashBase
}

func IsSaneProposalTxs(txs *proto.ProposalTxs) bool {
	if err := xdrutil.ValidateBatch(txs.Hash); err != nil {
		return false
	}
	if err := xdrutil.ValidateBatch(txs.Nondeterministic_data); err != nil {
		return false
	}
	if xdrutil.TokenSizeOfBatch(txs.Hash) !=
		xdrutil.TokenSizeOfBatch(txs.Nondeterministic_data) {
		return false
	}
	return true
}

// IsValidBasis reports whether a ConsensusBasis is valid:
// its proposal_txs are structurally sane and fast-hash to the
// proposal_txs_hash in the ConsensusValue it carries.
func IsValidBasis(basis *proto.ConsensusBasis) bool {
	if !IsSaneProposalTxs(&basis.Proposal_txs) {
		return false
	}
	return HashProposalTxsFast(&basis.Proposal_txs) == basis.Value.Proposal_txs_hash
}
