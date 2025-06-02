package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/scp/solo/follower"
	"github.com/groundhogchain/chuckd/scp/solo/leader"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	liblog.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestSolo(t *testing.T) {

	ctx := context.Background()
	testnet := testnet.NewTestNetwork()
	rand := runtime.NewRandFromRoot()

	leaderCred := protoutil.MustGenerateIdentity(ctx, rand, "solo_leader")

	numFollowers := 3
	followerCred := make([]*proto.ValidatorPrivateInfo, numFollowers)
	quorum := make([]proto.ValidatorPublicInfo, numFollowers)
	for i := range followerCred {
		followerCred[i] = protoutil.MustGenerateIdentity(ctx, rand, fmt.Sprintf("solo_follower_%d", i))
		quorum[i] = followerCred[i].Public_info
	}

	leaderSvc := leader.NewSoloLeaderService(
		ctx,
		testnet,
		leader.NewRandomChoiceCombiner(ctx, runtime.NewRandFromRoot()),
		time.Millisecond*100, // slot delay
		*leaderCred,
		quorum,
	)
	leaderSvc.Start(ctx, leaderCred.Public_info.Address) // start leader

	followerHandler := make([]*testHandler, numFollowers)
	followerSvc := make([]*follower.SoloFollowerService, numFollowers)
	for i := range followerSvc {
		followerHandler[i] = newTestHandler()
		followerSvc[i] = follower.NewSoloFollowerService(
			ctx,
			testnet,
			leaderCred.Public_info,
			*followerCred[i],
			followerHandler[i],
		)
		followerSvc[i].Start(ctx, followerCred[i].Public_info.Address) // start followers
	}

	slot := int64(1)
	x := proto.SoloStatement{Statement_type: proto.SoloStatementAlpha}
	y := proto.SoloStatement{Statement_type: proto.SoloStatementBeta}

	followerSvc[0].Nominate(ctx, slot, xdrutil.MarshalXdr(&x), xdrutil.MarshalXdr(&y))

	// validate
	var wgValidate sync.WaitGroup
	wgValidate.Add(3)
	go func() {
		defer wgValidate.Done()
		vv := followerHandler[0].WaitValidateValue()
		vv.Return(scpapi.FullyValidValue)
	}()
	go func() {
		defer wgValidate.Done()
		vv := followerHandler[1].WaitValidateValue()
		vv.Return(scpapi.MaybeValidValue)
	}()
	go func() {
		defer wgValidate.Done()
		vv := followerHandler[2].WaitValidateValue()
		vv.Return(scpapi.FullyValidValue)
	}()
	wgValidate.Wait()

	// externalize
	var wgExternalize sync.WaitGroup
	wgExternalize.Add(3)
	go func() {
		defer wgExternalize.Done()
		followerHandler[0].WaitExternalizedValue()
	}()
	go func() {
		defer wgExternalize.Done()
		followerHandler[1].WaitExternalizedValue()
	}()
	go func() {
		defer wgExternalize.Done()
		followerHandler[2].WaitExternalizedValue()
	}()
	wgExternalize.Wait()
}

type testHandler struct {
	validateValue     chan validateValueCall
	externalizedValue chan externalizedValueCall
}

type validateValueCall struct {
	ctx        context.Context
	slot       scpapi.Slot
	value      scpapi.Value
	nomination bool
	result     chan<- scpapi.ValidationLevel
}

func (vv validateValueCall) Return(r scpapi.ValidationLevel) {
	vv.result <- r
}

type externalizedValueCall struct {
	ctx   context.Context
	slot  scpapi.Slot
	value scpapi.Value
}

func newTestHandler() *testHandler {
	return &testHandler{
		validateValue:     make(chan validateValueCall),
		externalizedValue: make(chan externalizedValueCall),
	}
}

func (h *testHandler) CombineCandidates(context.Context, scpapi.Slot, []scpapi.Value) scpapi.Value {
	panic("XXX")
}

func (h *testHandler) ValidateValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value, nomination bool) scpapi.ValidationLevel {
	result := make(chan scpapi.ValidationLevel)
	h.validateValue <- validateValueCall{ctx: ctx, slot: slot, value: value, nomination: nomination, result: result}
	return <-result
}

func (h *testHandler) WaitValidateValue() validateValueCall {
	return <-h.validateValue
}

func (h *testHandler) ExtractValidValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) *scpapi.Value {
	return nil
}

func (h *testHandler) ExternalizedValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) {
	h.externalizedValue <- externalizedValueCall{ctx: ctx, slot: slot, value: value}
}

func (h *testHandler) WaitExternalizedValue() externalizedValueCall {
	return <-h.externalizedValue
}
