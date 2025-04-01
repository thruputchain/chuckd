package discovery

import "context"

// advertise tries to advetise our quorum slices to every validator participating in them.
// we try to deliver quorum slices to each validator in parallel with throttle and backoff.
// the process repeats once every two hours.
func (s *DiscoveryService) advertise(ctx context.Context) {
	XXX
}
