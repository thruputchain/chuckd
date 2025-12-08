package hogsched

import (
	"context"
	"fmt"
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
	DumpDir      string
}

func hostMapValues(hostMap map[string]string) []string {
	values := make([]string, 0, len(hostMap))
	for _, value := range hostMap {
		values = append(values, value)
	}
	return values
}

func SplitFixedAndVariable(hosts map[string]string) (fixedHostValues []string, variableHostKeys []string) {
	for hk, hv := range hosts {
		if IsWildcard(hv) {
			variableHostKeys = append(variableHostKeys, hk)
		} else {
			fixedHostValues = append(fixedHostValues, hv)
		}
	}
	return
}

func (x Scheduler) Schedule(ctx context.Context, xid string, spec Spec, kv map[string]any) error {

	liblog.Infof("executing schedule %s", spec.Name)

	c := hoglock.Client{Net: x.Net, LockdAddress: x.LockdAddress}

	// acquire locks on hog machines
	hosts := spec.Hosts()
	fixedHostValues, variableHostKeys := SplitFixedAndVariable(hosts) // variable contains host variable names, excluding the trailing "?"
	liblog.Infof("acquiring locks for %v on %v and %d more", xid, fixedHostValues, len(variableHostKeys))

	allocated, lease := c.AcquireLocks(ctx, xid, fixedHostValues, len(variableHostKeys))
	defer lease.Unlock()

	// prepare injected environment
	inject := map[string]string{
		"HOG_XID": xid,
	}

	// build host_variable->host map
	// host variables are identifiers followed by "?"
	assignedHostMap := map[string]string{}
	i := 0
	for _, vhk := range variableHostKeys {
		assignedHostMap[vhk] = allocated[i]
		i += 1
	}

	// replace all host variables in the job specs with their assignment
	spec.SubstituteHostVariables(ctx, assignedHostMap)

	// inject host map into environment
	hostMap := spec.Hosts()
	for hk, hv := range hostMap {
		inject[fmt.Sprintf("HOG_HOST_%s", hk)] = hv
	}
	kv["hog_host_map"] = hostMap

	// apply key/value substitutions in spec
	rwSpec := spec.Rewrite(ctx, kv)

	liblog.Infof("locks for %v acquired as %v, starting jobs", hostMap, xid)

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
	inject["HOG_USER"] = user.Username

	err = libtools.SshExecPar(
		ctx,
		rwSpec.HostValues(),
		rwSpec.ScriptsWithJobDir(x.JobDir, user.Username, x.DumpDir),
		rwSpec.AftersWithJobDir(x.JobDir, user.Username, x.DumpDir),
		rwSpec.Envs(inject),
	)
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
	return libtools.CopyTo(ctx, job.HostValue, job.FileIn, remote)
}

func (x Scheduler) copyFilesFromJob(ctx context.Context, job RewrittenJob) error {
	remote := make([]string, len(job.FileOut))
	for i, local := range job.FileOut {
		liblog.Infof("copying %s from job %s", local, job.Name)
		remote[i] = path.Join(x.JobDir, local)
	}
	return libtools.CopyFrom(ctx, job.HostValue, remote, job.FileOut)
}
