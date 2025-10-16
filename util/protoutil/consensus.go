package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func HashCV(m xdrutil.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueHash {
	bb := m.MarshalXdr(cv)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}
