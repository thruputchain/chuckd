package broadcast

import "time"

type BroadcastConfig struct {
	DeliveryPar            int // XXX: not used
	UnusedMulticastTreeTTL time.Duration
}

var DefaultBroadcastConfig = BroadcastConfig{
	DeliveryPar:            100,
	UnusedMulticastTreeTTL: time.Minute,
}
