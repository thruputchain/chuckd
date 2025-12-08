package hogsched

import (
	"fmt"
	"maps"
	"path/filepath"
)

type RewrittenSpec struct {
	Name string         `toml:"name"`
	Jobs []RewrittenJob `toml:"jobs"`
}

type RewrittenJob struct {
	Name      string            `toml:"name"`
	HostKey   string            `toml:"host_key"`
	HostValue string            `toml:"host_value"`
	Script    string            `toml:"script"`
	After     string            `toml:"after"`
	Env       map[string]string `toml:"env"`
	FileIn    []string          `toml:"file_in"`
	FileOut   []string          `toml:"file_out"`
}

func (x RewrittenSpec) HostValues() []string {
	hvs := make([]string, len(x.Jobs))
	for i, job := range x.Jobs {
		hvs[i] = job.HostValue
	}
	return hvs
}

func (x RewrittenSpec) ScriptsWithJobDir(jobDir string, user string, dumpDir string) []string {
	jobDir = filepath.Clean(jobDir)
	prefix := fmt.Sprintf(`
echo "creating dump directory" %q
mkdir -p %q
export HOG_DUMP_DIR=%q
echo "creating job directory" %q
chmod -R +w %q || true
rm -rf %q
mkdir -p %q
cd %q
export USER_HOME=$HOME
export HOME=%q
echo "forwarding github urls through ssh-agent"
hog_fwd
export GIT_SSH_COMMAND="ssh -F $HOME/.ssh/config"
`, dumpDir, dumpDir, dumpDir, jobDir, jobDir, jobDir, jobDir, jobDir, jobDir)
	suffix := `

`
	r := []string{}
	for _, job := range x.Jobs {
		r = append(r, prefix+job.Script+suffix)
	}
	return r
}

func (x RewrittenSpec) AftersWithJobDir(jobDir string, user string, dumpDir string) []string {
	jobDir = filepath.Clean(jobDir)
	prefix := fmt.Sprintf(`
export HOG_DUMP_DIR=%q
echo "going to job directory" %q
cd %q
export HOME=%q
echo "re-forwarding github urls through ssh-agent"
hog_fwd
export GIT_SSH_COMMAND="ssh -F $HOME/.ssh/config"
`, dumpDir, jobDir, jobDir, jobDir)
	suffix := `
`
	r := []string{}
	for _, job := range x.Jobs {
		if job.After != "" {
			r = append(r, prefix+job.After+suffix)
		} else {
			r = append(r, "")
		}
	}
	return r
}

func (x RewrittenSpec) Envs(inject map[string]string) []map[string]string {
	envs := []map[string]string{}
	for _, job := range x.Jobs {
		env := maps.Clone(job.Env)
		maps.Copy(env, inject)
		env["HOG_HOST_KEY"] = job.HostKey
		env["HOG_HOST"] = job.HostValue
		envs = append(envs, env)
	}
	return envs
}
