package kernel

import proto "github.com/groundhogchain/protocol-go"

type EnvelopeState int

const (
	EnvelopeStateInvalid EnvelopeState = iota
	EnvelopeStateValid
)

type SCP struct{}

func (x *SCP) getLocalNodeID() proto.SCPNodeID {
	panic("XXX")
}
