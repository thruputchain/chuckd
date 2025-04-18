package broadcast

import "time"

type BroadcastConfig struct {
	DeliveryPar            int
	UnusedMulticastTreeTTL time.Duration
}

var DefaultBroadcastConfig = BroadcastConfig{
	DeliveryPar:            100,
	UnusedMulticastTreeTTL: time.Minute,
}
