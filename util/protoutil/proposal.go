package protoutil

import (
	"github.com/thruputchain/chuckd/util/xdrutil"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type ProposalTxsSerialized struct {
	ProposalTxs      *proto.ProposalTxs
	Transaction_hash []proto.SignedTransactionHash
	Transaction_data []proto.NondeterministicProposalData
}

func SerializeProposalTxs(txs *proto.ProposalTxs) (ProposalTxsSerialized, error) {
	txh, err := xdrutil.SerializeSignedTransactionHashBatch(txs.Signed_hash)
	if err != nil {
		return ProposalTxsSerialized{}, err
	}
	txd, err := xdrutil.SerializeNondeterministicProposalDataBatch(txs.Nondeterministic_data)
	if err != nil {
		return ProposalTxsSerialized{}, err
	}
	return ProposalTxsSerialized{
		ProposalTxs:      txs,
		Transaction_hash: txh,
		Transaction_data: txd,
	}, nil
}

// ProposalFromBasis reconstructs the Proposal associated with a ConsensusBasis.
// The basis must be valid (see IsValidBasis) for the reconstruction to be meaningful.
func ProposalFromBasis(basis *proto.ConsensusBasis) *proto.Proposal {
	return &proto.Proposal{
		Header:       basis.Value.Proposal_header,
		Transactions: basis.Proposal_txs,
	}
}
