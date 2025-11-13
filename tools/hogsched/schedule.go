package hogsched

import (
	"context"
	"fmt"
	"path"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/chuckd/tools/libtools"
)

type Scheduler struct {
	Net          libnet.Network
	LockdAddress string
	JobDir       string
}

func (x Scheduler) Schedule(ctx context.Context, xid string, spec Spec) error {

	liblog.Infof("executing schedule %s", spec.Name)

	c := hoglock.Client{Net: x.Net, LockdAddress: x.LockdAddress}

	// acquire locks on hog machines
	hosts := spec.HostObjects()
	liblog.Infof("acquiring locks on %v", hosts)
	lease := c.AcquireLocks(ctx, hosts)
	defer lease.Unlock()

	liblog.Infof("locks acquired, starting jobs")

	// copy files into host
	for _, job := range spec.Jobs {
		err := x.copyFilesToJob(ctx, job)
		if err != nil {
			return err
		}
	}

	// executing scripts in parallel
	scripts := spec.ScriptsWithJobDir(x.JobDir)
	inject := map[string]string{
		"HOG_XID": xid,
	}
	fmt.Println("SPEC", spec)
	envs := spec.Envs(inject)
	fmt.Println("ENV", envs)
	err := libtools.SshExecPar(ctx, hosts, scripts, envs)
	if err != nil {
		return err
	}

	// copy files out of host
	for _, job := range spec.Jobs {
		err := x.copyFilesFromJob(ctx, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (x Scheduler) copyFilesToJob(ctx context.Context, job Job) error {
	remote := make([]string, len(job.FileIn))
	for i, local := range job.FileIn {
		liblog.Infof("copying %s to job %s", local, job.Name)
		remote[i] = path.Join(x.JobDir, local)
	}
	return libtools.CopyTo(ctx, job.Host, job.FileIn, remote)
}

func (x Scheduler) copyFilesFromJob(ctx context.Context, job Job) error {
	remote := make([]string, len(job.FileOut))
	for i, local := range job.FileOut {
		liblog.Infof("copying %s from job %s", local, job.Name)
		remote[i] = path.Join(x.JobDir, local)
	}
	return libtools.CopyFrom(ctx, job.Host, remote, job.FileOut)
}
