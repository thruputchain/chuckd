package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// advertise tries to advetise the quorum slices we know of to every validator participating in them.
// we try to deliver quorum slices to each validator in parallel with throttle and backoff.
// the process repeats once every few minutes hours.
func (s *DiscoveryService) advertise(ctx context.Context) {
	epoch := 0
	for {
		epoch += 1
		// log.Infof("advertising epoch %d begins", epoch)
		s.advertiseToAll(ctx, epoch)
		// log.Infof("advertising epoch %d ends", epoch)
		time.Sleep(time.Duration(s.cfg.Advertise_interval_msec) * time.Millisecond)
	}
}

func (s *DiscoveryService) advertiseToAll(ctx context.Context, epoch int) {
	limiter := make(chan struct{}, s.cfg.Advertise_par)
	var wg sync.WaitGroup

	snapshot := s.CachedSnapshots()
	for _, v := range s.ourValidators {
		wg.Add(1)
		limiter <- struct{}{} // acquire an execution slot
		go func(v proto.ValidatorPublicInfo) {
			defer wg.Done()
			defer func() { <-limiter }() // release the execution slot
			s.advertiseToValidator(ctx, epoch, v, snapshot)
		}(v)
	}
	wg.Wait()
}

func (s *DiscoveryService) advertiseToValidator(
	ctx context.Context,
	epoch int,
	v proto.ValidatorPublicInfo,
	snapshot proto.KnownQuorumSets,
) {
	// log.Infof("advertising to %s at epoch %d", protoutil.ValidatorPublicShortString(v), epoch)
	c, err := DialDiscoveryClient(ctx, s.net, v.Address)
	if err != nil {
		log.Errorf("discovery dialing %s: %v", protoutil.VPublicShortString(v), err)
		return
	}
	defer c.Close()

	r, err := c.Discovery_update_quorums(ctx, snapshot)
	if err != nil {
		log.Errorf("advertise to %s at epoch %d: %s", protoutil.VPublicShortString(v), epoch, err)
		return
	}
	s.updateKnown(r)
}
