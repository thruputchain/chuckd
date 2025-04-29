package broadcast

import "time"

// XXX: move to xdr
type BroadcastConfig struct {
	UnusedMulticastTreeTTL time.Duration
	ChunkSize              int // maximum chunk size for bulk/batch broadcasts
}

var DefaultBroadcastConfig = BroadcastConfig{
	UnusedMulticastTreeTTL: time.Minute,
	ChunkSize:              10e3,
}
