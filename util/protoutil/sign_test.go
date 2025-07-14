package protoutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/libstat"
)

func BenchmarkSign(b *testing.B) {
	ctx := context.Background()
	priv := MustGenerateIdentity(ctx, librand.New(), "")
	msg := make([]byte, 300e3)
	for b.Loop() {
		SignBytes(priv.Secret_key, msg)
	}
}

func TestSign(t *testing.T) {
	ctx := context.Background()
	priv := MustGenerateIdentity(ctx, librand.New(), "")
	msg := make([]byte, 300e3)
	thru := libstat.Counter{}
	for i := 0; i < 10000; i++ {
		SignBytes(priv.Secret_key, msg)
		thru.Add(float64(len(msg)))
	}
	fmt.Printf("signing %v/sec\n", humanize.Bytes(uint64(thru.SpeedPerSec())))
}
