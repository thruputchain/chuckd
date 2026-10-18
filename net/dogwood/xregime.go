package dogwood

import "github.com/thruputchain/chuckd/net/magnolia"

// nChunksPerSheaf = nPayBytes / maxBytesPerChunk / nPaySheaves

func CalcRegime(numPayloadBytes int) magnolia.Regime {
	switch {
	case numPayloadBytes < 100e3:
		return magnolia.Regime{ // max 10 chunks per sheaf
			NumPaySheaves:    1,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 34e3,
		}
	case numPayloadBytes < 500e3:
		return magnolia.Regime{ // max 10 chunks per sheaf
			NumPaySheaves:    5,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 100e3,
		}
	case numPayloadBytes < 1e6:
		return magnolia.Regime{ // max 10 chunks per sheaf
			NumPaySheaves:    5,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 100e3,
		}
	case numPayloadBytes < 10e6:
		return magnolia.Regime{ // max 10 chunks per sheaf
			NumPaySheaves:    5,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 250e3,
		}
	case numPayloadBytes < 40e6:
		return magnolia.Regime{ // max 16 chunks per sheaf
			NumPaySheaves:    5,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 500e3,
		}
	default:
		return magnolia.Regime{ // more than 16 chunks per sheaf
			NumPaySheaves:    5,
			NumParitySheaves: 2,
			MaxBytesPerChunk: 500e3,
		}
	}
}
