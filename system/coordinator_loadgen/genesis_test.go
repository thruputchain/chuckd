package coordinator_loadgen

import (
	"context"
	"strings"
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type fakeGenesisCeremonyControl struct {
	tips       []proto.SchedulerGetTipResult
	loadResult proto.DbSchedResult
	loads      []proto.GenesisConfigurationV2
	commit     proto.GenesisCeremonyResult
	request    *proto.GenesisCeremonyRequest
}

func (f *fakeGenesisCeremonyControl) GetTip(context.Context) (proto.SchedulerGetTipResult, error) {
	if len(f.tips) == 0 {
		return proto.SchedulerGetTipResult{}, nil
	}
	tip := f.tips[0]
	f.tips = f.tips[1:]
	return tip, nil
}

func (f *fakeGenesisCeremonyControl) LoadGenesisConfigurationV2(_ context.Context, genesis proto.GenesisConfigurationV2) (proto.DbSchedResult, error) {
	f.loads = append(f.loads, genesis)
	return f.loadResult, nil
}

func (f *fakeGenesisCeremonyControl) CommitGenesisCeremony(_ context.Context, request proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	f.request = &request
	return f.commit, nil
}

func loadgenTip(height proto.BlockHeight) proto.SchedulerGetTipResult {
	result := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*result.Db_tip() = proto.DbTip{}
	result.Db_tip().Value.Proposal_header.Block_height = height
	return result
}

func loadgenCeremonyResult(height proto.BlockHeight) proto.GenesisCeremonyResult {
	result := proto.GenesisCeremonyResult{Outcome: proto.GenesisCeremonyCommitted}
	result.Commit().Db_tip.Value.Proposal_header.Block_height = height
	return result
}

func TestEnsureGenesisCeremonyLoadsAndCommitsBeforeInitialization(t *testing.T) {
	parent := loadgenTip(0)
	ctl := &fakeGenesisCeremonyControl{
		tips: []proto.SchedulerGetTipResult{
			{Outcome: proto.SchedulerGetTipNoGenesisBlock},
			parent,
		},
		loadResult: proto.DbSchedResultOK,
		commit:     loadgenCeremonyResult(1),
	}
	tx := proto.SignedTransaction{}
	initialize, err := ensureGenesisCeremony(
		context.Background(), ctl, proto.GenesisConfigurationV2{}, []proto.SignedTransaction{tx})
	if err != nil {
		t.Fatal(err)
	}
	if !initialize {
		t.Fatal("new chain should submit post-genesis initialization transactions")
	}
	if len(ctl.loads) != 1 || ctl.request == nil {
		t.Fatalf("loads=%d request=%v", len(ctl.loads), ctl.request)
	}
	if ctl.request.Expected_parent_value_hash != protoutil.HashValueToProto(&parent.Db_tip().Value) {
		t.Fatal("ceremony request used the wrong parent hash")
	}
	txs, err := xdrutil.SerializeSignedTransactionBatch(ctl.request.Transactions)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || protoutil.HashSignedTx(&txs[0]) != protoutil.HashSignedTx(&tx) {
		t.Fatalf("ceremony request contained %d unexpected transactions", len(txs))
	}
}

func TestEnsureGenesisCeremonyRestoredChainSkipsInitialization(t *testing.T) {
	ctl := &fakeGenesisCeremonyControl{tips: []proto.SchedulerGetTipResult{loadgenTip(7)}}
	initialize, err := ensureGenesisCeremony(
		context.Background(), ctl, proto.GenesisConfigurationV2{}, []proto.SignedTransaction{{}})
	if err != nil {
		t.Fatal(err)
	}
	if initialize {
		t.Fatal("restored chain should drain generated initialization transactions")
	}
	if len(ctl.loads) != 0 || ctl.request != nil {
		t.Fatalf("restored chain loaded or recommitted genesis: loads=%d request=%v", len(ctl.loads), ctl.request)
	}
}

func TestEnsureGenesisCeremonyRejectsFailedCommit(t *testing.T) {
	ctl := &fakeGenesisCeremonyControl{
		tips:   []proto.SchedulerGetTipResult{loadgenTip(0)},
		commit: proto.GenesisCeremonyResult{Outcome: proto.GenesisCeremonyExecutionFailed},
	}
	_, err := ensureGenesisCeremony(
		context.Background(), ctl, proto.GenesisConfigurationV2{}, []proto.SignedTransaction{{}})
	if err == nil || !strings.Contains(err.Error(), "GenesisCeremonyExecutionFailed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
