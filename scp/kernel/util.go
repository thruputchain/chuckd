package kernel

import (
	proto "github.com/groundhogchain/protocol-go"
)

func EnvelopeEqual(p, q *proto.SCPEnvelope) bool {
	panic("XXX")
}

func BallotHash(b proto.SCPBallot) proto.SCPHash {
	// hash field and value, don't serialize
	panic("XXX")
}
