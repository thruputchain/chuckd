package systemfixed

const (
	MaxBytesTxsMessage = 2e9

	MaxBytesConsensusValueMessage = 70e6

	// MaxBytesSCPEnvelopeMessage must be smaller than the max message on the SCPD rpc bridge
	MaxBytesSCPEnvelopeMessage = 5e6
)
