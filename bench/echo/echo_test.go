package echo

import (
	"context"
	"sync"
	"testing"

	"github.com/groundhogchain/chuckd/test/testnet"
)

func TestEcho(t *testing.T) {

	ctx := context.Background()
	net := testnet.NewTestNetwork()

	numConns := 10
	maxPendingCalls := 10
	numIterations := 2000000
	numBytesPerPay := 0

	eSend := NewEchoBench(ctx, net, true, numConns, maxPendingCalls, numIterations, numBytesPerPay, true)
	eRecv := NewEchoBench(ctx, net, false, numConns, maxPendingCalls, numIterations, numBytesPerPay, true)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		eRecv.Run(ctx, "receiver", "sender")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		eSend.Run(ctx, "sender", "receiver")
	}()

	wg.Wait()
}
