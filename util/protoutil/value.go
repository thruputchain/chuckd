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

func ValueHashFromBasis(basis *proto.ConsensusBasis) ValueHash {
	return HashValue(BasisValue(basis))
}

func BasisValue(basis *proto.ConsensusBasis) *proto.ConsensusValue {
	return &basis.Value
}

// ValueHeight returns the block height carried in the value's proposal header.
func ValueHeight(value *proto.ConsensusValue) proto.BlockHeight {
	return value.Proposal_header.Block_height
}

// ValueProposer returns the proposer carried in the value's proposal header.
func ValueProposer(value *proto.ConsensusValue) proto.ValidatorID {
	return value.Proposal_header.Proposer
}

// ValueTimepoint returns the block timepoint carried in the value's proposal header.
func ValueTimepoint(value *proto.ConsensusValue) proto.TimePoint {
	return value.Proposal_header.Block_timepoint
}

func BasisAgreesWithValue(basis *proto.ConsensusBasis, value *proto.ConsensusValue) bool {
	return ValueEquals(&basis.Value, value)
}

// HeaderEquals compares ProposalHeaders field by field because ProposalHeader
// contains an ExtensionPoint (whose Go representation holds an interface) and
// is therefore not safely comparable with the == operator.
func HeaderEquals(h1, h2 *proto.ProposalHeader) bool {
	return h1.Ext.V == h2.Ext.V &&
		h1.Network_hash == h2.Network_hash &&
		h1.Proposer == h2.Proposer &&
		h1.Block_timepoint == h2.Block_timepoint &&
		h1.Block_height == h2.Block_height &&
		h1.Txs_hash == h2.Txs_hash &&
		h1.Storage_hash == h2.Storage_hash &&
		h1.Events_hash == h2.Events_hash
}

// ValueEquals reports whether two ConsensusValues are equal. It compares
// field by field because ConsensusValue contains an Upgrades slice and is
// therefore not comparable with the == operator.
func ValueEquals(v1, v2 *proto.ConsensusValue) bool {
	return v1.Protocol == v2.Protocol &&
		v1.Ext.V == v2.Ext.V &&
		v1.Prev_hash == v2.Prev_hash &&
		HeaderEquals(&v1.Proposal_header, &v2.Proposal_header) &&
		v1.Proposal_txs_hash == v2.Proposal_txs_hash &&
		upgradesEqual(v1.Upgrades, v2.Upgrades)
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
