package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ProposalTxsSerialized struct {
	Txs              *proto.ProposalTxs
	Transaction_hash []proto.SignedTransactionHash
	Transaction_data []proto.NondeterministicProposalData
}

func SerializeProposalTxs(txs *proto.ProposalTxs) (ProposalTxsSerialized, error) {
	txh, err := xdrutil.SerializeSignedTransactionHashBatch(txs.Hash)
	if err != nil {
		return ProposalTxsSerialized{}, err
	}
	txd, err := xdrutil.SerializeNondeterministicProposalDataBatch(txs.Nondeterministic_data)
	if err != nil {
		return ProposalTxsSerialized{}, err
	}
	return ProposalTxsSerialized{
		Txs:              txs,
		Transaction_hash: txh,
		Transaction_data: txd,
	}, nil
}
