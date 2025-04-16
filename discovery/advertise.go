package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// Advertise quorum slices we know on every REFRESH_INTERVAL.
const REFRESH_INTERVAL = time.Minute * 2

// advertise tries to advetise the quorum slices we know of to every validator participating in them.
// we try to deliver quorum slices to each validator in parallel with throttle and backoff.
// the process repeats once every few minutes hours.
func (s *DiscoveryService) advertise(ctx context.Context) {
	epoch := 0
	for {
		epoch += 1
		log.Infof("advertising epoch %d begins", epoch)
		s.advertiseToAll(ctx, epoch)
		log.Infof("advertising epoch %d ends", epoch)
		time.Sleep(REFRESH_INTERVAL)
	}
}

// advertise quorum slices to ADVERTISE_CONCURRENCY validators in parallel.
const ADVERTISE_CONCURRENCY = 10

func (s *DiscoveryService) advertiseToAll(ctx context.Context, epoch int) {
	limiter := make(chan struct{}, ADVERTISE_CONCURRENCY)
	var wg sync.WaitGroup

	snapshots := s.CachedSnapshots()
	for _, v := range protoutil.ValidatorsInQuorumSlices(s.ourSlices) {
		wg.Add(1)
		limiter <- struct{}{} // acquire an execution slot
		go func(v proto.ValidatorPublicInfo) {
			defer wg.Done()
			defer func() { <-limiter }() // release the execution slot
			s.advertiseToValidator(ctx, epoch, v, snapshots)
		}(v)
	}
	wg.Wait()
}

func (s *DiscoveryService) advertiseToValidator(
	ctx context.Context,
	epoch int,
	v proto.ValidatorPublicInfo,
	snapshots proto.ValidatorQuorumSlicesSnapshots,
) {
	log.Infof("advertising to %s at epoch %d", protoutil.ValidatorPublicShortString(v), epoch)
	c, err := DialDiscoveryClient(ctx, s.dialer, v.Address)
	if err != nil {
		log.Errorf("discovery dialing %s: %v", protoutil.ValidatorPublicShortString(v), err)
		return
	}
	defer c.Close()

	r, err := xdrutil.CallRpc(c.Discovery_update_quorums, snapshots)
	if err != nil {
		log.Errorf("advertise to %s at epoch %d: %s", protoutil.ValidatorPublicShortString(v), epoch, err)
		return
	}
	for _, v := range r.Quorum_slices_snapshots {
		s.addQuorumSlices(v)
	}
}
