package kernel

import proto "github.com/groundhogchain/protocol-go"

type LocalNode struct{}

func (x *LocalNode) getQuorumSetHash() proto.SCPHash {
	panic("XXX")
}
