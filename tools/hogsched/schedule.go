package hogsched

import (
	"context"
	"os/user"
	"path"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/chuckd/tools/libtools"
)

type Scheduler struct {
	Net          libnet.Network
	LockdAddress string
	JobDir       string
}

func (x Scheduler) Schedule(ctx context.Context, xid string, spec Spec, kv map[string]any) error {

	liblog.Infof("executing schedule %s", spec.Name)
	rwSpec := spec.Rewrite(ctx, kv)

	c := hoglock.Client{Net: x.Net, LockdAddress: x.LockdAddress}

	// acquire locks on hog machines
	hosts := rwSpec.Hosts()
	liblog.Infof("acquiring locks on %v", hosts)
	lease := c.AcquireLocks(ctx, hosts)
	defer lease.Unlock()

	liblog.Infof("locks acquired, starting jobs")

	// copy files into host
	// TODO: parallelize (or combine with script execution)
	for _, job := range rwSpec.Jobs {
		err := x.copyFilesToJob(ctx, job)
		if err != nil {
			return err
		}
	}

	// executing scripts in parallel
	user, err := user.Current()
	must.NoError(ctx, err)

	scripts := rwSpec.ScriptsWithJobDir(x.JobDir, user.Username)
	inject := map[string]string{
		"HOG_USER": user.Username,
		"HOG_XID":  xid,
	}
	envs := rwSpec.Envs(inject)
	err = libtools.SshExecPar(ctx, hosts, scripts, envs)
	if err != nil {
		return err
	}

	// copy files out of host
	// TODO: parallelize (or combine with script execution)
	for _, job := range rwSpec.Jobs {
		err := x.copyFilesFromJob(ctx, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (x Scheduler) copyFilesToJob(ctx context.Context, job RewrittenJob) error {
	remote := make([]string, len(job.FileIn))
	for i, local := range job.FileIn {
		liblog.Infof("copying %s to job %s", local, job.Name)
		remote[i] = path.Join(x.JobDir, local)
	}
	return libtools.CopyTo(ctx, job.Host, job.FileIn, remote)
}

func (x Scheduler) copyFilesFromJob(ctx context.Context, job RewrittenJob) error {
	remote := make([]string, len(job.FileOut))
	for i, local := range job.FileOut {
		liblog.Infof("copying %s from job %s", local, job.Name)
		remote[i] = path.Join(x.JobDir, local)
	}
	return libtools.CopyFrom(ctx, job.Host, remote, job.FileOut)
}
