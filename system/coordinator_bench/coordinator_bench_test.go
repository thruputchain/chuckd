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
	"syscall"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/libtime"
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

const virtualBenchEnvVar = "RUN_CHUCKD_VIRTUAL_TEST"

func Test3Healthy(t *testing.T) {
	testHealthyValidators(t, virtualBenchEnvVar, 3)
}

func Test3Staggered(t *testing.T) {
	test3Staggered(t, virtualBenchEnvVar, 5*time.Second)
}

func Test2Healthy1Down(t *testing.T) {
	testHealthyValidators(t, virtualBenchEnvVar, 2)
}

func Test2Healthy1Restart(t *testing.T) {
	setup := setupValidatorBenchTest(t, virtualBenchEnvVar, 180*time.Second)
	ctx := setup.ctx
	defer setup.cancel()
	systems := setup.systems
	opcfgs := setup.opcfgs
	cfgs := setup.cfgs
	outsideNets := setup.outsideNets
	dbDirs := setup.dbDirs

	v1Ready := make(chan struct{})
	v2Ready := make(chan struct{})
	v0RestartedReady := make(chan struct{})

	var restartOnce sync.Once
	systems[0].startAndWaitHeight(ctx, opcfgs[0], 30, make(chan struct{}),
		func(cb *batonkit.CommittableBlock) {
			if cb.BlockHeight() < proto.BlockHeight(10) {
				return
			}
			restartOnce.Do(func() {
				go func() {
					time.Sleep(time.Second)
					systems[0].Stop()
					time.Sleep(10 * time.Second)

					liblog.Infof(ctx, "restarting v0")
					restarted := NewCoordinatorBenchSystem(
						liblog.WithLabeledLogger(ctx, "restarted-v0"),
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
					restartOpCfg := opcfgs[0]
					restartOpCfg.SCPHandleAddr = fmt.Sprintf("127.0.0.1:%d", 8880)
					restartOpCfg.SCPControlAddr = fmt.Sprintf("127.0.0.1:%d", 9990)
					restarted.startAndWaitHeight(ctx, restartOpCfg, 30, v0RestartedReady, nil)
				}()
			})
		})

	systems[1].startAndWaitHeight(ctx, opcfgs[1], 30, v1Ready, nil)
	systems[2].startAndWaitHeight(ctx, opcfgs[2], 30, v2Ready, nil)

	for _, ready := range []chan struct{}{v1Ready, v2Ready, v0RestartedReady} {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for restart benchmark to complete: %v", ctx.Err())
		}
	}

	liblog.Infof(ctx, "SUCCESS")

	syscall.Exit(0)
}

type validatorBenchSetup struct {
	ctx         context.Context
	cancel      context.CancelFunc
	systems     []*CoordinatorBenchSystem
	opcfgs      []coordinator.OperationalConfig
	readies     []chan struct{}
	cfgs        []proto.Config
	outsideNets []hostScopedNetwork
	dbDirs      []string
}

func setupValidatorBenchTest(
	t *testing.T,
	envVar string,
	timeout time.Duration,
) validatorBenchSetup {
	liblog.SetLogLevelDebug()

	if os.Getenv(envVar) != "1" {
		t.Skipf("set %s=1 to run this integration test", envVar)
	}

	scpdPath, err := exec.LookPath("scpd")
	if err != nil {
		t.Skip("skipping coordinator bench integration test: scpd not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	validatorAddrs := []string{"v0:23000", "v1:23000", "v2:23000"}

	baseCfg := common.MakeBaseConfig(400*time.Millisecond, int64(len(validatorAddrs)), 5)
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}

	cfgs, _ := common.GenQuorumConfigs(ctx, rand.New(rand.NewSource(1)), baseCfg, validatorAddrs)

	outsideBase := testnet.NewTestNetwork()
	systems := make([]*CoordinatorBenchSystem, 0, len(cfgs))
	opcfgs := make([]coordinator.OperationalConfig, 0, len(cfgs))
	readies := make([]chan struct{}, 0, len(cfgs))
	outsideNets := make([]hostScopedNetwork, 0, len(cfgs))
	dbDirs := make([]string, 0, len(cfgs))

	for i := range cfgs {
		outside := hostScopedNetwork{
			host: strings.Split(validatorAddrs[i], ":")[0],
			base: outsideBase,
		}
		outsideNets = append(outsideNets, outside)

		dbDir := filepath.Join(t.TempDir(), "db")
		dbDirs = append(dbDirs, dbDir)
		storageDir := filepath.Join(t.TempDir(), "storage")

		sys := NewCoordinatorBenchSystem(
			liblog.WithLabeledLogger(ctx, fmt.Sprintf("v%d", i)),
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

	return validatorBenchSetup{
		ctx:         ctx,
		cancel:      cancel,
		systems:     systems,
		opcfgs:      opcfgs,
		readies:     readies,
		cfgs:        cfgs,
		outsideNets: outsideNets,
		dbDirs:      dbDirs,
	}
}

func testHealthyValidators(t *testing.T, envVar string, numToStart int) {
	setup := setupValidatorBenchTest(t, envVar, 90*time.Second)
	ctx := setup.ctx
	defer setup.cancel()
	systems := setup.systems
	opcfgs := setup.opcfgs
	readies := setup.readies

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

	liblog.Infof(ctx, "SUCCESS")

	// coordinator cannot exit gracefully. Just end the test without cleanup.
	syscall.Exit(0)
}

func test3Staggered(t *testing.T, envVar string, staggerDelay time.Duration) {
	// libruntime.InstallStackTraceOnInterrupt()
	librpc.SetLogStats(true)
	setup := setupValidatorBenchTest(t, envVar, 120*time.Second)
	ctx := setup.ctx
	defer setup.cancel()
	systems := setup.systems
	opcfgs := setup.opcfgs
	readies := setup.readies

	allReached := make(chan struct{})
	var wg sync.WaitGroup
	for i := range systems {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			startDelay := time.Duration(i) * staggerDelay
			if startDelay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(startDelay):
				}
			}

			systems[i].startAndWaitHeightAndKeepRunning(
				ctx,
				opcfgs[i],
				proto.BlockHeight(30),
				readies[i],
				allReached,
			)
		}(i)
	}

	for _, ready := range readies {
		select {
		case <-ready:
			liblog.Debugf(ctx, "READY")
		case <-ctx.Done():
			t.Fatalf("timed out waiting for staggered coordinator bench systems to complete: %v", ctx.Err())
		}
	}

	liblog.Infof(ctx, "SUCCESS")

	syscall.Exit(0)

	// Coordinator cannot exit gracefully. Just end the test without cleanup.
	// close(allReached)
	// wg.Wait()
}
