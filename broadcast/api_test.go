package broadcast

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

var testCtx context.Context
var testRR *rand.Rand
var testVs []*proto.ValidatorPrivateInfo
var testRouter router.Router

func TestMain(m *testing.M) {

	log.LogVerbosely()
	runtime.SetRootSeed(0)
	testCtx = context.Background()
	testRR = runtime.NewRandFromRoot()

	testVs = []*proto.ValidatorPrivateInfo{
		protoutil.MustGenerateIdentity(testCtx, testRR, "alpha"),
		protoutil.MustGenerateIdentity(testCtx, testRR, "beta"),
		protoutil.MustGenerateIdentity(testCtx, testRR, "gamma"),
	}
	for i := range testVs {
		log.Infof("test validator #%d: %s\n", i, protoutil.ValidatorPublicShortString(testVs[i].Public_info))
	}

	testRouter = &router.TestRouter{
		Trees: map[router.TestTreeKey]*router.TestTreeNode{
			// 0 -> 1, 0 -> 2
			{VID: protoutil.ValidatorIDFromPublic(testVs[0].Public_info)}: {
				V: testVs[0].Public_info,
				Children: []*router.TestTreeNode{
					{V: testVs[1].Public_info},
					{V: testVs[2].Public_info},
				},
			},
			// 1 -> 2 -> 0
			{VID: protoutil.ValidatorIDFromPublic(testVs[1].Public_info)}: {
				V: testVs[1].Public_info,
				Children: []*router.TestTreeNode{
					{
						V: testVs[2].Public_info,
						Children: []*router.TestTreeNode{
							{V: testVs[0].Public_info},
						},
					},
				},
			},
			// 2 -> 0 -> 1
			{VID: protoutil.ValidatorIDFromPublic(testVs[2].Public_info)}: {
				V: testVs[2].Public_info,
				Children: []*router.TestTreeNode{
					{
						V: testVs[0].Public_info,
						Children: []*router.TestTreeNode{
							{V: testVs[1].Public_info},
						},
					},
				},
			},
		},
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestStreaming(t *testing.T) {

	r0 := runtime.NewRandFromRoot()
	r1 := runtime.NewRandFromRoot()
	r2 := runtime.NewRandFromRoot()

	testnet := test.NewTestNetwork()
	b0 := NewBroadcastService(testCtx, DefaultBroadcastConfig, testnet, r0, *testVs[0], testRouter, AcceptEverything{})
	b1 := NewBroadcastService(testCtx, DefaultBroadcastConfig, testnet, r1, *testVs[1], testRouter, AcceptEverything{})
	b2 := NewBroadcastService(testCtx, DefaultBroadcastConfig, testnet, r2, *testVs[2], testRouter, AcceptEverything{})

	b0.Start(testCtx, testVs[0].Public_info.Address)
	b1.Start(testCtx, testVs[1].Public_info.Address)
	b2.Start(testCtx, testVs[2].Public_info.Address)

	payload := []byte{1, 2, 3}

	go func() {
		sch := b0.StreamingAPI().MakeChannel(testCtx)
		sch.Send(testCtx, payload)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rch := b1.StreamingAPI().AcceptChannel()
		_, _, p, err := rch.Receive(testCtx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		rch := b2.StreamingAPI().AcceptChannel()
		_, _, p, err := rch.Receive(testCtx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}

func TestBulk(t *testing.T) {

	r0 := runtime.NewRandFromRoot()
	r1 := runtime.NewRandFromRoot()
	r2 := runtime.NewRandFromRoot()

	payload := []byte{1, 2, 3}

	cfg := BroadcastConfig{
		UnusedMulticastTreeTTL: time.Second,
		ChunkSize:              1, // three messages for the payload
	}

	testnet := test.NewTestNetwork()
	b0 := NewBroadcastService(testCtx, cfg, testnet, r0, *testVs[0], testRouter, AcceptEverything{})
	b1 := NewBroadcastService(testCtx, cfg, testnet, r1, *testVs[1], testRouter, AcceptEverything{})
	b2 := NewBroadcastService(testCtx, cfg, testnet, r2, *testVs[2], testRouter, AcceptEverything{})

	b0.Start(testCtx, testVs[0].Public_info.Address)
	b1.Start(testCtx, testVs[1].Public_info.Address)
	b2.Start(testCtx, testVs[2].Public_info.Address)

	go func() {
		_, err := b0.BulkAPI().BulkSend(testCtx, payload)
		if err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, p, err := b1.BulkAPI().BulkReceive(testCtx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		_, p, err := b2.BulkAPI().BulkReceive(testCtx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}
