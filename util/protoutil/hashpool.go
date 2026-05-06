package protoutil

import (
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: if HashCVC is too inefficient, memoize HashCVC hash computation
func CVCDisplay(cvc *proto.ConsensusValueCue) string {
	return libxdr.HashString(HashCVC(cvc))
}
