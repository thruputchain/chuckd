package script

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/librand"
)

func TestBroadcastLocalScript(t *testing.T) {

	ctx := context.Background()

	def := BenchBroadcastExecDef{
		QUIC:             false,
		FanOut:           5,
		ChunkPerChannel:  100,
		BytesPerSec:      128e3, // 2 chunks/sec
		NumIterations:    1,
		ChunkSize:        64e3,
		MulticastTreeTTL: time.Second * 3,
		MetricsPort:      2222,
		BenchPort:        4000,
	}

	fmt.Println(def.LocalScript(ctx, librand.New(), "chuckd", 3))
}
