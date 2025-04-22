package broadcast

import "time"

type BroadcastConfig struct {
	UnusedMulticastTreeTTL time.Duration
}

var DefaultBroadcastConfig = BroadcastConfig{
	UnusedMulticastTreeTTL: time.Minute,
}
