package coord

import "github.com/groundhogchain/chuckd/util/xdrutil"

func (svc *CoordinatorService) waitForDbReady() xdrutil.ShardCards {
	svc.dbLk.Lock()
	defer svc.dbLk.Unlock()
	for !svc.dbIsReady() {
		svc.dbCond.Wait()
	}
	return svc.dbShards
}

func (svc *CoordinatorService) dbIsReady() bool {
	return svc.dbIsUp && svc.dbIsRestored
}

// findCompleteDb returns a list of db shards of highest block height that form a complete cluster.
// It returns nil if there is no complete db cluster.
// A set of shards is deemed complete if:
// - it is not empty
// - all shards report the same block height and total shard count
// - the shard count matches the reported count
func (svc *CoordinatorService) findCompleteDb() xdrutil.ShardCards {
	byHeight := map[uint64]xdrutil.ShardCards{} // block_height -> shards
	for _, dbc := range svc.db {
		byHeight[dbc.Block_height] = append(byHeight[dbc.Block_height], dbc)
	}
	var foundReady bool
	var foundHeight uint64
	var foundShards xdrutil.ShardCards
	for height, shards := range byHeight {
		shards.Sort()
		if isShardCountComplete(shards) {
			if !foundReady || foundHeight < height {
				foundReady = true
				foundHeight = height
				foundShards = shards
			}
		}
	}
	return foundShards
}

func isShardCountComplete(shards xdrutil.ShardCards) bool {
	if len(shards) == 0 {
		return false
	}
	shardCount := int64(-1)
	for _, dbc := range shards {
		if shardCount < 0 {
			shardCount = int64(dbc.Shard_count)
		}
		if shardCount != int64(dbc.Shard_count) {
			return false
		}
	}
	if shardCount != int64(len(shards)) {
		return false
	}
	return true
}
