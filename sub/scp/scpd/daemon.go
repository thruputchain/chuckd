package scpd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libruntime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/textutil"
	proto "github.com/groundhogchain/protocol-go"
)

// Must call SCPHandler.SetHandler after this call.
func StartSCPDaemon(
	ctx context.Context,
	scpdPath string,
	controlAddr string, // localhost:3334
	handleAddr string, // localhost:3333
	isValidator bool,
	cfg proto.SCPConfig,
	secretKey proto.SecretKey,
) (*SCPHandler, *SCPController, func()) {

	liblog.Infof("starting scpd, control_address=%s, handle_address=%s ...", controlAddr, handleAddr)

	// prepare config

	// cfgText := xdrutil.MustXdrToJsonString(&cfg)
	// TODO: replace the shim when we have XDR-JSON for both Go and Rust
	cfgBytes, err := json.MarshalIndent(SCPConfigToShim(cfg), "", "   ")
	must.NoError(ctx, err)
	cfgText := string(cfgBytes)

	tmpFile, err := os.CreateTemp("", "chuckd-scpd-config-*.json")
	tmpPath := tmpFile.Name()
	must.NoError(ctx, err)
	_, err = io.WriteString(tmpFile, cfgText)
	must.NoError(ctx, err)
	err = tmpFile.Close()
	must.NoError(ctx, err)

	// encode secret key

	must.Assert(ctx, secretKey.Scheme == proto.Ed25519)
	secretKeyText := protoutil.EncodeBase64(secretKey.Ed25519()[:])

	// start handler

	handler := NewSCPHandler(ctx)
	handler.Start(ctx, handleAddr) // start returns after the tcp listener is installed

	// launch scpd

	cmd := exec.Command(
		scpdPath,
		"--control-addr", controlAddr,
		"--handle-addr", handleAddr,
		"--is-validator", fmt.Sprintf("%v", isValidator),
		"--config-path", tmpPath,
		"--secret-key", secretKeyText,
	)
	cmd.Env = []string{"RUST_BACKTRACE=1", "RUST_LOG=" + runtime.GetLogLevel()}
	cmd.Stdout = textutil.NewPrefixWriter(os.Stderr, "[scpd:out] ")
	cmd.Stderr = textutil.NewPrefixWriter(os.Stderr, "[scpd:err] ")

	err = cmd.Start()
	must.NoError(ctx, err)
	kill := func() {
		fmt.Fprintf(os.Stderr, "killing scpd with pid %d\n", -cmd.Process.Pid)
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	libruntime.RegisterInterruptHook(kill)

	time.Sleep(300 * time.Millisecond) // allow time for the scp control service to start

	// connect controller

	controller, err := DialSCPController(ctx, libnet.TCPNet{}, controlAddr)
	must.NoError(ctx, err)

	return handler, controller, kill
}
