package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func HashProposalSlow(p *proto.Proposal) proto.ProposalHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, p)
	return proto.ProposalHash(h.Sum(nil))
}

// HashProposalFast computes the same combined hash as hashing the proposal
// with its transaction batches zeroed, XOR'd with the parallel hashes of the
// two batches. This allows the large batches to be hashed in parallel.
func HashProposalFast(p *proto.Proposal) proto.ProposalHash {
	pBase := *p
	pBase.Transactions.Hash = proto.ParallelBatch{}
	pBase.Transactions.Nondeterministic_data = proto.ParallelBatch{}

	var hashBase proto.SHA256Hash
	var hashTxsHash proto.SHA256Hash
	var hashTxsND proto.SHA256Hash

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		hashBase = HashProposalSlow(&pBase)
	}()
	go func() {
		defer wg.Done()
		hashTxsHash = xdrutil.HashParallelBatch(p.Transactions.Hash)
	}()
	go func() {
		defer wg.Done()
		hashTxsND = xdrutil.HashParallelBatch(p.Transactions.Nondeterministic_data)
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

func IsSaneProposal(p *proto.Proposal) bool {
	return IsSaneProposalTxs(&p.Transactions)
}

func IsSaneBasis(basis *proto.ConsensusBasis) bool {
	return IsSaneProposalTxs(&basis.Proposal_txs)
}
