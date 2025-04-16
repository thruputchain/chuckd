package broadcast

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"sync"
	"testing"

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

func TestDelivery(t *testing.T) {

	r0 := runtime.NewRandFromRoot()
	r1 := runtime.NewRandFromRoot()
	r2 := runtime.NewRandFromRoot()

	testnet := test.NewTestNetwork()
	b0 := StartServerTest(testCtx, r0, testnet, testVs[0].Public_info.Address, *testVs[0], testRouter, AcceptEverything{})
	b1 := StartServerTest(testCtx, r1, testnet, testVs[1].Public_info.Address, *testVs[1], testRouter, AcceptEverything{})
	b2 := StartServerTest(testCtx, r2, testnet, testVs[2].Public_info.Address, *testVs[2], testRouter, AcceptEverything{})

	payload := []byte{1, 2, 3}

	go func() {
		sch := b0.API().MakeChannel()
		sch.Send(payload)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rch := b1.API().AcceptChannel()
		_, _, p := rch.Receive()
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		rch := b2.API().AcceptChannel()
		_, _, p := rch.Receive()
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}
