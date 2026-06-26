package protoutil

import (
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX(PERF): if HashCVC is too inefficient, memoize HashCVC hash computation.
// XXX: to make memoizing *CVC to hash effective, all CVCs must be passed as pointers.
func ValueDisplay(value *proto.ConsensusValue) string {
	return libxdr.HashString(HashValue(value).Bytes())
}
