package common

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func MakeBaseConfigTPS(
	tps int, // transactions per second
	txSize int, // transaction size
	numVs int64, // number of validators
) proto.Config {
	return MakeBaseConfig(
		time.Second,
		numVs,
		uint32(min(5, numVs)), // fan out
		1e6,                   // chunk size
		5,                     // number payload channels (aka trees)
		2,                     // number of parity channels
	)
}

func AddSoloLeaderConfig(base proto.Config, leader proto.ValidatorPublicInfo) proto.Config {
	if base.Solo_config == nil {
		base.Solo_config = &proto.SoloConfig{}
	}
	base.Solo_config.Leader = leader
	return base
}

func MakeBaseConfig(
	round time.Duration, // round is the target round duration (nominate + commit).
	numVs int64, // numVs is an idealized number of equally important validators.
	fanOut uint32,
	chunkSize uint64,
	numPayloadChannels uint32,
	numParityChannels uint32,
) proto.Config {
	return proto.Config{
		Validator_config: nil,
		Wallet_config:    nil,
		Solo_config:      nil,
		Discovery_config: &proto.DiscoveryConfig{
			Advertise_interval_msec: (5 * round).Milliseconds(),
			Advertise_par:           5,
			Snapshot_ttl_msec:       (7 * round).Milliseconds(),
		},
		Router_config: &proto.RouterConfig{
			Tree_fan_out: 5,
		},
		Broadcast_config: &proto.BroadcastConfig{
			Multicast_tree_ttl_msec: round.Milliseconds(), // how long to cache multicast tree topology
			Chunk_size:              chunkSize,            // size of individual broadcast chunks
		},
		Spectra_config: &proto.SpectraConfig{
			Num_payload_channels: numPayloadChannels,
			Num_parity_channels:  numParityChannels,
			Partial_ttl_msec:     round.Milliseconds(), // ttl for partial data from incomplete transmissions
		},
		Mempool_config: &proto.MempoolConfig{
			Transaction_ttl_msec:     numVs * round.Milliseconds(),
			Submission_lot_size:      1000,
			Submission_lot_wait_msec: 50,
		},
		Catchup_config: &proto.CatchupConfig{
			Catchup_block_ttl_sec: proto.Seconds((5 * round).Seconds()),
			Catchup_block_age:     5,
		},
		Baton_config: &proto.BatonConfig{
			Heartbeat_msec: round.Milliseconds() / 10,
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
		ids[i] = protoutil.VIDOfPublic(ks[i])
	}
	for i := range cfgs {
		cfgs[i] = base
		cfgs[i].Validator_config = &proto.ValidatorConfig{
			Private_info: vs[i],
			Quorum_set: proto.QuorumSet{
				Threshold:  uint32(2 * len(addrs) / 3),
				Validators: ids,
				Inner_sets: nil,
			},
			Known_validators: ks,
		}
	}

	return cfgs, ks
}

func GenIdentity(ctx context.Context, rand *rand.Rand, addr string) proto.ValidatorPrivateInfo {
	return *protoutil.MustGenerateIdentity(ctx, rand, addr)
}

func MakeSoloLeaderCfg(
	ctx context.Context,
	rand *rand.Rand,
	id proto.ValidatorPrivateInfo,
	quorum []proto.ValidatorPublicInfo,
) proto.Config {
	return proto.Config{
		Validator_config: &proto.ValidatorConfig{
			Private_info:     id,
			Known_validators: quorum,
		},
		Solo_config: &proto.SoloConfig{
			Quorum: quorum,
		},
	}
}
