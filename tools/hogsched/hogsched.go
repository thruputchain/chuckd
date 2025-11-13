package hogsched

import (
	"context"
	"path"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/chuckd/tools/libtools"
)

type Config struct {
	Net          libnet.Network
	LockdAddress string
	JobDir       string
}

// XXX: rundir
func (cfg Config) Schedule(ctx context.Context, spec Spec) error {

	liblog.Infof("executing schedule %s", spec.Name)

	c := hoglock.Client{Net: cfg.Net, LockdAddress: cfg.LockdAddress}

	// acquire locks on hog machines
	hosts := spec.HostObjects()
	liblog.Infof("acquiring locks on %v", hosts)
	lease := c.AcquireLocks(ctx, hosts)
	defer lease.Unlock()

	liblog.Infof("locks acquired, starting jobs")

	// copy files into host
	for _, job := range spec.Jobs {
		err := cfg.copyFilesToJob(ctx, job)
		if err != nil {
			return err
		}
	}

	// executing scripts in parallel
	scripts := spec.Scripts()
	err := libtools.SshExecPar(ctx, hosts, scripts)
	if err != nil {
		return err
	}

	// copy files out of host
	for _, job := range spec.Jobs {
		err := cfg.copyFilesFromJob(ctx, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cfg Config) copyFilesToJob(ctx context.Context, job Job) error {
	remote := make([]string, len(job.FileIn))
	for i, local := range job.FileIn {
		remote[i] = path.Join(cfg.JobDir, local)
	}
	return libtools.CopyTo(ctx, job.Host, job.FileIn, remote)
}

func (cfg Config) copyFilesFromJob(ctx context.Context, job Job) error {
	remote := make([]string, len(job.FileOut))
	for i, local := range job.FileOut {
		remote[i] = path.Join(cfg.JobDir, local)
	}
	return libtools.CopyFrom(ctx, job.Host, remote, job.FileOut)
}
