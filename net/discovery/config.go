package discovery

import "time"

type DiscoveryConfig struct {
	AdvertiseInterval time.Duration
	AdvertiseMaxPar   int
	MinSnapshotTTL    time.Duration
}

var DefaultDiscoveryConfig = DiscoveryConfig{
	AdvertiseInterval: time.Minute * 2,
	AdvertiseMaxPar:   10,
	MinSnapshotTTL:    time.Second * 10,
}
