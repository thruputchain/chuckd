package hogsched

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/chuckd/tools/libtools"
)

type Config struct {
	Net          libnet.Network
	LockdAddress string
	RunDir       string
}

func (cfg Config) Schedule(ctx context.Context, spec Spec) error {

	liblog.Infof("executing schedule %s", spec.Name)

	c := hoglock.Client{Net: cfg.Net, LockdAddress: cfg.LockdAddress}

	// acquire locks on hog machines
	hosts := spec.HostObjects()
	liblog.Infof("acquiring locks on %v", hosts)
	lease := c.AcquireLocks(ctx, hosts)
	defer lease.Unlock()

	liblog.Infof("locks acquired, starting jobs")

	// executing scripts in parallel
	scripts := spec.Scripts()
	return libtools.SshExecPar(ctx, hosts, scripts)
}
