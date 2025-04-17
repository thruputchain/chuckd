package broadcast

import "time"

type BroadcastConfig struct {
	MaxDeliveryPar         int
	UnusedMulticastTreeTTL time.Duration
}

var DefaultBroadcastConfig = BroadcastConfig{
	MaxDeliveryPar:         100,
	UnusedMulticastTreeTTL: time.Minute,
}
