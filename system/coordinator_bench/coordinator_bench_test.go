package coordinator_bench

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// hostScopedNetwork allows each validator to bind debug services on :port while
// still sharing a common underlying TestNetwork.
type hostScopedNetwork struct {
	host string
	base libnet.Network
}

func (n hostScopedNetwork) String() string {
	return n.base.String()
}

func (n hostScopedNetwork) DialAddress(ctx context.Context, address string) (net.Conn, error) {
	return n.base.DialAddress(ctx, address)
}

func (n hostScopedNetwork) ListenAddress(ctx context.Context, address string) (net.Listener, error) {
	if strings.HasPrefix(address, ":") {
		return n.base.ListenAddress(ctx, n.host+address)
	}
	return n.base.ListenAddress(ctx, address)
}

func Test3Healthy(t *testing.T) {
	testHealthyValidators(t, "RUN_COORDINATOR_BENCH_TEST3", 3)
}

func Test2Healthy1Down(t *testing.T) {
	testHealthyValidators(t, "RUN_COORDINATOR_BENCH_TEST3", 2)
}

func Test2Healthy1Restart(t *testing.T) {
	prevLevel := liblog.GetLogLevel()
	_ = liblog.SetLogLevelDebug()
	t.Cleanup(func() {
		_ = liblog.SetLogLevel(prevLevel)
	})

	if os.Getenv("RUN_COORDINATOR_BENCH_TEST3") != "1" {
		t.Skip("set RUN_COORDINATOR_BENCH_TEST3=1 to run this integration test")
	}

	scpdPath, err := exec.LookPath("scpd")
	if err != nil {
		t.Skip("skipping coordinator bench integration test: scpd not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	validatorAddrs := []string{"v0:23000", "v1:23000", "v2:23000"}

	baseCfg := common.MakeBaseConfig(400*time.Millisecond, int64(len(validatorAddrs)), 5)
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}

	cfgs, _ := common.GenQuorumConfigs(ctx, rand.New(rand.NewSource(1)), baseCfg, validatorAddrs)

	outsideBase := testnet.NewTestNetwork()
	outsideNets := make([]hostScopedNetwork, len(cfgs))
	dbDirs := make([]string, len(cfgs))
	storageDirs := make([]string, len(cfgs))
	opcfgs := make([]coordinator.OperationalConfig, len(cfgs))

	systems := make([]*CoordinatorBenchSystem, len(cfgs))
	for i := range cfgs {
		outsideNets[i] = hostScopedNetwork{
			host: strings.Split(validatorAddrs[i], ":")[0],
			base: outsideBase,
		}
		dbDirs[i] = filepath.Join(t.TempDir(), "db")
		storageDirs[i] = filepath.Join(t.TempDir(), "storage")

		systems[i] = NewCoordinatorBenchSystem(
			ctx,
			libtime.RealTime{},
			rand.New(rand.NewSource(int64(i+10))),
			outsideNets[i],
			cfgs[i],
			CoordinatorBenchConfig{
				TxsPerSec: 100,
				SchedPort: 31000,
			},
			dbDirs[i],
		)
		t.Cleanup(systems[i].Close)

		opcfgs[i] = coordinator.OperationalConfig{
			StorageDir:     storageDirs[i],
			SchedAddr:      systems[i].SchedAddress(),
			DebugPort:      40555,
			InAddr:         fmt.Sprintf("in-%d", i),
			OutAddr:        validatorAddrs[i],
			WalletAddr:     "127.0.0.1:0",
			SCPDaemonPath:  scpdPath,
			SCPHandleAddr:  fmt.Sprintf("127.0.0.1:%d", 3330+i),
			SCPControlAddr: fmt.Sprintf("127.0.0.1:%d", 4440+i),
			IsValidator:    true,
		}
	}

	v1Ready := make(chan struct{})
	v2Ready := make(chan struct{})
	v0RestartedReady := make(chan struct{})
	var restartedMu sync.Mutex
	var restartedSystem *CoordinatorBenchSystem
	t.Cleanup(func() {
		restartedMu.Lock()
		defer restartedMu.Unlock()
		if restartedSystem != nil {
			restartedSystem.Close()
		}
	})

	var restartOnce sync.Once
	startAndWaitHeight(ctx, systems[0], opcfgs[0], 30, make(chan struct{}), func(cb *batonkit.CommittableBlock) {
		if cb.BlockHeight() < proto.BlockHeight(10) {
			return
		}
		restartOnce.Do(func() {
			go func() {
				time.Sleep(time.Second)
				systems[0].Stop()
				time.Sleep(10 * time.Second)

				restarted := NewCoordinatorBenchSystem(
					ctx,
					libtime.RealTime{},
					rand.New(rand.NewSource(10)),
					outsideNets[0],
					cfgs[0],
					CoordinatorBenchConfig{
						TxsPerSec: 100,
						SchedPort: 31000,
					},
					dbDirs[0],
				)
				restartedMu.Lock()
				restartedSystem = restarted
				restartedMu.Unlock()
				startAndWaitHeight(ctx, restarted, opcfgs[0], 30, v0RestartedReady, nil)
			}()
		})
	})

	startAndWaitHeight(ctx, systems[1], opcfgs[1], 30, v1Ready, nil)
	startAndWaitHeight(ctx, systems[2], opcfgs[2], 30, v2Ready, nil)

	for _, ready := range []chan struct{}{v1Ready, v2Ready, v0RestartedReady} {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for restart benchmark to complete: %v", ctx.Err())
		}
	}
}

func startAndWaitHeight(
	ctx context.Context,
	system *CoordinatorBenchSystem,
	opcfg coordinator.OperationalConfig,
	targetHeight proto.BlockHeight,
	ready chan<- struct{},
	onCommit func(*batonkit.CommittableBlock),
) {
	t0 := time.Now()

	system.CoordinatorBenchCluster.StartScheduler(ctx)

	ctl := system.System.Start(ctx, opcfg)
	ctl.SignalGenesisIsReady()

	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, system.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	system.sched.SetCoordScheduler(coordSchedConn)

	system.CoordinatorBenchCluster.StartPump(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-system.stop:
				return
			default:
			}

			cb := system.notify.ReceiveCommitted(ctx)
			if cb == nil {
				return
			}

			if onCommit != nil {
				onCommit(cb)
			}
			if cb.BlockHeight() < targetHeight {
				continue
			}

			benchDuration := time.Since(t0)
			fmt.Println(float64(benchDuration.Milliseconds()) / float64(targetHeight))

			close(ready)
			return
		}
	}()
}

func testHealthyValidators(t *testing.T, envVar string, numToStart int) {
	prevLevel := liblog.GetLogLevel()
	_ = liblog.SetLogLevelDebug()
	t.Cleanup(func() {
		_ = liblog.SetLogLevel(prevLevel)
	})

	if os.Getenv(envVar) != "1" {
		t.Skipf("set %s=1 to run this integration test", envVar)
	}

	scpdPath, err := exec.LookPath("scpd")
	if err != nil {
		t.Skip("skipping coordinator bench integration test: scpd not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	validatorAddrs := []string{"v0:23000", "v1:23000", "v2:23000"}

	baseCfg := common.MakeBaseConfig(400*time.Millisecond, int64(len(validatorAddrs)), 5)
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}

	cfgs, _ := common.GenQuorumConfigs(ctx, rand.New(rand.NewSource(1)), baseCfg, validatorAddrs)

	outsideBase := testnet.NewTestNetwork()
	systems := make([]*CoordinatorBenchSystem, 0, len(cfgs))
	opcfgs := make([]coordinator.OperationalConfig, 0, len(cfgs))
	readies := make([]chan struct{}, 0, len(cfgs))

	for i := range cfgs {
		outside := hostScopedNetwork{
			host: strings.Split(validatorAddrs[i], ":")[0],
			base: outsideBase,
		}

		dbDir := filepath.Join(t.TempDir(), "db")
		storageDir := filepath.Join(t.TempDir(), "storage")

		sys := NewCoordinatorBenchSystem(
			ctx,
			libtime.RealTime{},
			rand.New(rand.NewSource(int64(i+10))),
			outside,
			cfgs[i],
			CoordinatorBenchConfig{
				TxsPerSec: 100,
				SchedPort: 31000,
			},
			dbDir,
		)
		systems = append(systems, sys)
		t.Cleanup(sys.Close)

		opcfgs = append(opcfgs, coordinator.OperationalConfig{
			StorageDir:     storageDir,
			SchedAddr:      sys.SchedAddress(),
			DebugPort:      40555,
			InAddr:         fmt.Sprintf("in-%d", i),
			OutAddr:        validatorAddrs[i],
			WalletAddr:     "127.0.0.1:0",
			SCPDaemonPath:  scpdPath,
			SCPHandleAddr:  fmt.Sprintf("127.0.0.1:%d", 3330+i),
			SCPControlAddr: fmt.Sprintf("127.0.0.1:%d", 4440+i),
			IsValidator:    true,
		})

		readies = append(readies, make(chan struct{}))
	}

	var wg sync.WaitGroup
	for i := range systems[:numToStart] {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			systems[i].Start(ctx, opcfgs[i], 30, readies[i])
		}(i)
	}

	for _, ready := range readies[:numToStart] {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for coordinator bench systems to complete: %v", ctx.Err())
		}
	}

	wg.Wait()
}
