package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func HashProposalTxsSlow(txs *proto.ProposalTxs) proto.ProposalTxsHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, txs)
	return proto.ProposalTxsHash(h.Sum(nil))
}

// proposalTxsDomainSeparator is cxs(DomainSeparator("ProposalTxs")): the zero
// ExtensionPoint, the length of the name, and the name.
var proposalTxsDomainSeparator = []byte("\x00\x0bProposalTxs")

// HashProposalTxsFast computes the ProposalTxsHash a ConsensusValue commits to:
//
//	SHA-256(cxs(DomainSeparator("ProposalTxs")) || SHA-256(base) ||
//	        HashParallelBatch(signed_hash) ||
//	        HashParallelBatch(nondeterministic_data))
//
// where base is the XDR of txs with both batches emptied. The three hashes are
// computed in parallel and combined in order, so the batches cannot trade
// places (hogd#4508).
func HashProposalTxsFast(txs *proto.ProposalTxs) proto.ProposalTxsHash {
	txsBase := *txs
	txsBase.Signed_hash = proto.ParallelBatch{}
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
		hashTxsHash = xdrutil.HashParallelBatch(txs.Signed_hash)
	}()
	go func() {
		defer wg.Done()
		hashTxsND = xdrutil.HashParallelBatch(txs.Nondeterministic_data)
	}()
	wg.Wait()
	h := sha256.New()
	h.Write(proposalTxsDomainSeparator)
	h.Write(hashBase[:])
	h.Write(hashTxsHash[:])
	h.Write(hashTxsND[:])
	return proto.ProposalTxsHash(h.Sum(nil))
}

func IsSaneProposalTxs(txs *proto.ProposalTxs) bool {
	if err := xdrutil.ValidateBatch(txs.Signed_hash); err != nil {
		return false
	}
	if err := xdrutil.ValidateBatch(txs.Nondeterministic_data); err != nil {
		return false
	}
	if xdrutil.TokenSizeOfBatch(txs.Signed_hash) !=
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
