package protoutil

import proto "github.com/groundhogchain/protocol-go"

func RepackageProposedTransactions(ts []proto.ProposedTransaction) []*proto.ProposedTransaction {
	ps := make([]*proto.ProposedTransaction, len(ts))
	for i, t := range ts {
		ps[i] = &t
	}
	return ps
}
