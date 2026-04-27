package systemfixed

const (
	MaxBytesTxsMessage = 2e9

	MaxBytesConsensusValueMessage = 70e6

	// SECURITY: MaxBytesSCPEnvelopeMessage must be smaller than the max message of the SCPD rpc server.
	// Otherwise, an oversize SCP message forwarded from dogwood to SCPD will cause a crash.
	MaxBytesSCPEnvelopeMessage = 5e6
)
