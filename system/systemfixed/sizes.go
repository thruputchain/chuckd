package systemfixed

const (
	MaxBytesTxsMessage = 2_000_000_000

	MaxBytesConsensusValueMessage = 70_000_000

	// SECURITY: MaxBytesSCPEnvelopeMessage must be smaller than the max message of the SCPD rpc server.
	// Otherwise, an oversize SCP message forwarded from dogwood to SCPD will cause a crash.
	MaxBytesSCPEnvelopeMessage = 5_000_000
)
