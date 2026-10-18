package scpd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/libruntime"
	"github.com/thruputchain/lib-go/libtext"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
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
	networkHash *proto.NetworkHash,
) (*SCPHandler, *SCPController, func()) {

	liblog.Infof(ctx, "starting scpd, control_address=%s, handle_address=%s ...", controlAddr, handleAddr)
	liblog.Infof(ctx, "scp config: %v", libxdr.MustXdrToJsonString(&cfg))

	// prepare config

	// cfgText := libxdr.MustXdrToJsonString(&cfg)
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
	cmd.Env = scpdEnv(os.Environ(), liblog.GetLogLevel().String(), networkHash)
	cmd.Stdout = libtext.NewPrefixWriter(liblog.Writer(ctx), "[scpd:out] ")
	cmd.Stderr = libtext.NewPrefixWriter(liblog.Writer(ctx), "[scpd:err] ")

	err = cmd.Start()
	must.NoError(ctx, err)
	kill := func(sig os.Signal) {
		fmt.Fprintf(os.Stderr, "killing scpd with pid %d\n", cmd.Process.Pid)
		syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
	}
	libruntime.InstallInterruptHook(kill)

	// connect controller

	var controller *SCPController
	deadline := time.Now().Add(10 * time.Second)
	for {
		controller, err = DialSCPController(ctx, libnet.TCPNet{}, controlAddr)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			must.NoError(ctx, err)
		}
		select {
		case <-ctx.Done():
			must.NoError(ctx, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}

	return handler, controller, func() { kill(os.Kill) }
}

// scpdEnv returns the Rust logging settings, parent OTEL_* variables, and network hash.
// Parent service identity settings are withheld because scpd sets its own service identity.
// scpd uses the hash to derive per-slot block trace IDs (see util/blocktrace).
func scpdEnv(parent []string, logLevel string, networkHash *proto.NetworkHash) []string {
	env := []string{"RUST_BACKTRACE=full", "RUST_LOG=" + logLevel}
	for _, entry := range parent {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(key, "OTEL_") || key == "OTEL_SERVICE_NAME" {
			continue
		}
		if key == "OTEL_RESOURCE_ATTRIBUTES" {
			var kept []string
			for _, pair := range strings.Split(value, ",") {
				attrKey, _, _ := strings.Cut(pair, "=")
				attrKey = strings.TrimSpace(attrKey)
				if strings.TrimSpace(pair) == "" || attrKey == "service.name" || attrKey == "service.instance.id" {
					continue
				}
				kept = append(kept, pair)
			}
			if len(kept) == 0 {
				continue
			}
			entry = key + "=" + strings.Join(kept, ",")
		}
		env = append(env, entry)
	}
	if networkHash != nil {
		env = append(env, "GROUNDHOG_NETWORK_HASH="+hex.EncodeToString(networkHash[:]))
	}
	return env
}
