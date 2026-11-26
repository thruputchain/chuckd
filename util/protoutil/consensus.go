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

// proposalTxsDomainSeparator is cxs(DomainSeparator("ProposalTxs")), which
// goes before the ProposalTxsCommit in the hash: the zero ExtensionPoint, the
// length of the name, and the name.
var proposalTxsDomainSeparator = []byte("\x00\x0bProposalTxs")

// HashProposalTxsFast computes the ProposalTxsHash a ConsensusValue commits to:
//
//	SHA-256(cxs(DomainSeparator("ProposalTxs")) || cxs(ProposalTxsCommit))
//
// ProposalTxsCommit is a struct of the protocol's proposal.x. Its fields are,
// in order, an ext, which follows the ext of the ProposalTxs and is 0, the
// HashParallelBatch of signed_hash and the HashParallelBatch of
// nondeterministic_data. The two batch hashes are computed in parallel. Each
// has its own field, so the batches cannot trade places (hogd#4508).
//
// As in HashParallelBatch, the cxs encoding is written out by hand: the bytes
// of the separator, the one byte of the ext, then the two 32-byte hashes.
func HashProposalTxsFast(txs *proto.ProposalTxs) proto.ProposalTxsHash {
	var hashTxsHash proto.SHA256Hash
	var hashTxsND proto.SHA256Hash

	var wg sync.WaitGroup
	wg.Add(2)
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
	// The ext of the ProposalTxsCommit follows txs.Ext, whose only value is 0,
	// one byte in cxs. A second value of txs.Ext must be encoded here with it.
	h.Write([]byte{0})
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
