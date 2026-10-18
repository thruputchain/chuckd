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

	"github.com/rs/zerolog"
	"github.com/thruputchain/chuckd/deploy/common"
	"github.com/thruputchain/chuckd/system/coordinator"
	"github.com/thruputchain/chuckd/test/testnet"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	"github.com/thruputchain/lib-go/libtime"
	proto "github.com/thruputchain/thruput-protocol-go"
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

func (n hostScopedNetwork) Dial(ctx context.Context, address string) (net.Conn, error) {
	return n.base.Dial(ctx, address)
}

func (n hostScopedNetwork) Listen(ctx context.Context, address string) (net.Listener, error) {
	if strings.HasPrefix(address, ":") {
		return n.base.Listen(ctx, n.host+address)
	}
	return n.base.Listen(ctx, address)
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

	maxHeight := proto.BlockHeight(30)
	restartAfterNumCommits := 10

	readyToRestart := make(chan struct{})
	systems[0].Start(ctx, opcfgs[0], &restartAfterNumCommits, &maxHeight, readyToRestart, nil)
	go func() {
		// when this is commented out, we hit a panic inside scp (wip)
		// <-readyToRestart
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
		restarted.Start(ctx, restartOpCfg, nil, &maxHeight, v0RestartedReady, nil)
	}()

	systems[1].Start(ctx, opcfgs[1], nil, &maxHeight, v1Ready, nil)
	systems[2].Start(ctx, opcfgs[2], nil, &maxHeight, v2Ready, nil)

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
	liblog.SetLogLevel(zerolog.TraceLevel, false)

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

	maxNumCommits := 30

	var wg sync.WaitGroup
	for i := range systems[:numToStart] {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			systems[i].Start(ctx, opcfgs[i], &maxNumCommits, nil, readies[i], nil)
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

	maxHeight := proto.BlockHeight(30)

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

			systems[i].Start(
				ctx,
				opcfgs[i],
				nil,
				&maxHeight,
				readies[i],
				nil,
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
