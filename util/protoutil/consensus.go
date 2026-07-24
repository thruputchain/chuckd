package protoutil

import (
	"crypto/sha256"

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
	// ProposalTxsHash is defined by the protocol as SHA-256 of the canonical
	// XDR encoding. SHA-256 cannot be assembled by XORing hashes of the
	// parallel-batch segments; doing so produces a different digest from hogd.
	// Keep this compatibility entry point while using the canonical hash.
	return HashProposalTxsSlow(txs)
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
