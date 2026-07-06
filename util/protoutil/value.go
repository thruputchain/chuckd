package protoutil

import (
	"bytes"
	"crypto/sha256"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type ValueHash struct {
	bytes proto.SHA256Hash
}

func NewValueHash(b []byte) ValueHash {
	return ValueHash{bytes: proto.SHA256Hash(b)}
}

func (x ValueHash) Bytes() proto.SHA256Hash {
	return x.bytes
}

func (x ValueHash) Slice() []byte {
	return x.bytes[:]
}

func (x ValueHash) Display() string {
	return libxdr.HashString(x.Bytes())
}

func HashValueToProto(value *proto.ConsensusValue) proto.ConsensusValueHash {
	return HashValue(value).Bytes()
}

func HashValue(value *proto.ConsensusValue) ValueHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, value)
	return NewValueHash(h.Sum(nil))
}

// BasisAgreesWithValue reports whether the basis carries the given value and
// whether the transaction batches in the basis are the ones committed to by
// value.Proposal_hash.
func BasisAgreesWithValue(basis *proto.ConsensusBasis, value *proto.ConsensusValue) bool {
	if !ValueEquals(&basis.Value, value) {
		return false
	}
	proposal := proto.Proposal{
		Header:       value.Proposal_header,
		Transactions: basis.Proposal_txs,
	}
	return HashProposalFast(&proposal) == value.Proposal_hash
}

// ProposalMatchesValue reports whether the proposal is the one committed to
// by the consensus value.
func ProposalMatchesValue(p *proto.Proposal, value *proto.ConsensusValue) bool {
	return HeaderEquals(&p.Header, &value.Proposal_header) &&
		HashProposalFast(p) == value.Proposal_hash
}

// ValueEquals reports whether two ConsensusValues are equal. It compares
// field by field because ConsensusValue contains an Upgrades slice and is
// therefore not comparable with the == operator.
func ValueEquals(v1, v2 *proto.ConsensusValue) bool {
	return v1.Ext.V == v2.Ext.V &&
		v1.Proposer == v2.Proposer &&
		upgradesEqual(v1.Upgrades, v2.Upgrades) &&
		HeaderEquals(&v1.Proposal_header, &v2.Proposal_header) &&
		v1.Proposal_hash == v2.Proposal_hash &&
		v1.Prev_value_hash == v2.Prev_value_hash
}

// HeaderEquals reports whether two ProposalHeaders are equal. It compares
// field by field because ProposalHeader embeds an ExtensionPoint, which is
// not safely comparable with the == operator.
func HeaderEquals(h1, h2 *proto.ProposalHeader) bool {
	return h1.Ext.V == h2.Ext.V &&
		h1.Network_id == h2.Network_id &&
		h1.Timestamp == h2.Timestamp &&
		h1.Block_height == h2.Block_height &&
		h1.Txs_hash == h2.Txs_hash &&
		h1.Storage_hash == h2.Storage_hash &&
		h1.Events_hash == h2.Events_hash
}

func upgradesEqual(a, b []proto.Upgrade) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func ValidatorInfoEquals(v1, v2 *proto.ValidatorPublicInfo) bool {
	return *v1 == *v2
}
