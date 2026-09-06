package loadgen

import (
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestWorkBatchExpiryHeight(t *testing.T) {
	seen := make(map[proto.BlockHeight]bool)
	for batch := 0; batch < proto.KReplayCacheHeightsMaxLen; batch++ {
		height, err := workBatchExpiryHeight(batch)
		if err != nil {
			t.Fatal(err)
		}
		if height != proto.BlockHeight(1_000_000_000+batch) {
			t.Fatalf("batch %d: unexpected expiry %d", batch, height)
		}
		if seen[height] {
			t.Fatalf("batch %d reuses expiry %d", batch, height)
		}
		seen[height] = true
	}
}

func TestWorkBatchExpiryHeightRejectsOutOfRange(t *testing.T) {
	for _, batch := range []int{-1, proto.KReplayCacheHeightsMaxLen, proto.KReplayCacheHeightsMaxLen + 1} {
		if _, err := workBatchExpiryHeight(batch); err == nil {
			t.Fatalf("accepted invalid batch index %d", batch)
		}
	}
}
