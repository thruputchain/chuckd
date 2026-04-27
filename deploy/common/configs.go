package common

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libmath"
	proto "github.com/groundhogchain/protocol-go"
)

func MakeBaseConfig(
	round time.Duration, // round is the target round duration (nominate + commit).
	numVs int64, // numVs is an idealized number of equally important validators.
	fanOut uint32,
) proto.Config {
	return proto.Config{
		Validator_config: nil,
		Scp_config: &proto.SCPConfig{
			Max_timeout_ms:                  1000,
			Initial_ballot_timeout_ms:       750,
			Increment_ballot_timeout_ms:     750,
			Initial_nomination_timeout_ms:   750,
			Increment_nomination_timeout_ms: 750,
			Quorum_set:                      proto.SCPQuorumSet{},
		},
		Discovery_config: &proto.DiscoveryConfig{
			Advertise_interval_msec: (5 * round).Milliseconds(),
			Advertise_par:           5,
			Snapshot_ttl_msec:       (7 * round).Milliseconds(),
		},
		Router_config: &proto.RouterConfig{
			Tree_fan_out: 5,
		},
		Dogwood_config: &proto.DogwoodConfig{
			Route_count:             50,
			Route_time_to_live_msec: 60e3, // 1 minute
			Route_max_latency_msec:  1e3,  // 1 sec
			//
			Endpoint_max_pending: 16 * 50,
			Endpoint_num_clients: 16,
			//
			Endpoint_idle_ttl_msec:     60e3, // 1 minute
			Endpoint_dial_timeout_msec: 2e3,  // 2 sec
			Endpoint_redial_delay_msec: 100,  // 100 msec
			Endpoint_call_timeout_msec: 2e3,  // 2 sec
			//
			Stream_receiver_worker_count: 512,
			Stream_sender_worker_count:   32,
			//
			Transmission_receiver_decode_worker_count: 512,
			Transmission_sender_worker_count:          32,
			//
			Transmission_partial_max_ttl_msec:  10e3, // 10 sec
			Transmission_partial_idle_ttl_msec: 1e3,  // 1 sec
		},
		Mempool_config: &proto.MempoolConfig{
			Transaction_ttl_msec:     2 * round.Milliseconds(),
			Submission_lot_size:      1_000,
			Submission_lot_wait_msec: 50,
		},
		Catchup_config: &proto.CatchupConfig{
			Catchup_block_ttl_sec: proto.Seconds((5 * round).Seconds()),
			Catchup_block_age:     5,
		},
		Baton_config: &proto.BatonConfig{
			Skew_msec:          10,
			Heartbeat_msec:     round.Milliseconds() / 5,
			Patience_msec:      round.Milliseconds(),
			Max_catchup_height: 10000,
		},
		Repair_config: &proto.RepairConfig{
			Base_delay_msec:  200,
			Max_delay_msec:   5000,
			Multiplier_nom:   3,
			Multiplier_denom: 2,
		},
		Storage_config: &proto.StorageConfig{
			Reclaim_interval_msec: 5 * 60 * 1000, // 5 min
			Max_catchup_blocks:    60 * 60 * 24,  // 1 day worth, assuming 1 block per second
			Max_scp_slots:         60 * 60 * 24,  // 1 day worth, assuming 1 block per second
		},
	}
}

func GenQuorumConfigs(
	ctx context.Context,
	rand *rand.Rand,
	base proto.Config,
	addrs []string, // outside addresses of validators
) ([]proto.Config, []proto.ValidatorPublicInfo) {

	cfgs := make([]proto.Config, len(addrs))
	vs := make([]proto.ValidatorPrivateInfo, len(addrs))
	ks := make([]proto.ValidatorPublicInfo, len(addrs))
	ids := make([]proto.ValidatorID, len(addrs))
	for i := range cfgs {
		vs[i] = *protoutil.MustGenerateIdentity(ctx, rand, addrs[i])
		ks[i] = vs[i].Public_info
		ids[i] = protoutil.PublicVID(ks[i])
	}
	for i := range cfgs {
		cfgs[i] = base
		cfgs[i].Validator_config = &proto.ValidatorConfig{
			Private_info:     vs[i],
			Known_validators: ks,
		}
		cfgs[i].Scp_config.Quorum_set = proto.SCPQuorumSet{
			Threshold:  libmath.DivideRoundUp(uint32(2*len(addrs)), 3),
			Validators: protoutil.ValidatorToNodeIDs(ids),
			Inner_sets: nil,
		}
	}

	return cfgs, ks
}

func GenIdentity(ctx context.Context, rand *rand.Rand, addr string) proto.ValidatorPrivateInfo {
	return *protoutil.MustGenerateIdentity(ctx, rand, addr)
}
