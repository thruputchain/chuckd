package protoutil

import (
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX(PERF): if HashValue is too inefficient, memoize HashValue hash computation.
// XXX: to make memoizing *ConsensusValue to hash effective, all ConsensusValues must
// be passed as pointers.
func ValueDisplay(value *proto.ConsensusValue) string {
	return libxdr.HashString(HashValue(value).Bytes())
}
