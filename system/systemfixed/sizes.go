package systemfixed

import proto "github.com/groundhogchain/protocol-go"

const (
	MaxBytesTxsMessage = int(proto.KCoordinatorMaxBytesTxsMessage)

	MaxBytesConsensusValueMessage = int(proto.KCoordinatorMaxBytesCVMessage)

	// SECURITY: MaxBytesSCPEnvelopeMessage must be smaller than the max message of the SCPD rpc server.
	// Otherwise, an oversize SCP message forwarded from dogwood to SCPD will cause a crash.
	MaxBytesSCPEnvelopeMessage = int(proto.KCoordinatorMaxBytesSCPEnvelopeMessage)
)

const (
	MaxBytesCatchupReply = 1_000_000_000
)
