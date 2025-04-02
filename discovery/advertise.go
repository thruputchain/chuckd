package discovery

import "context"

// advertise tries to advetise the quorum slices we know of to every validator participating in them.
// we try to deliver quorum slices to each validator in parallel with throttle and backoff.
// the process repeats once every few minutes hours.
func (s *DiscoveryService) advertise(ctx context.Context) {
	panic("XXX")
}
