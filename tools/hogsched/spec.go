package hogsched

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/groundhogchain/chuckd/util/textutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Spec struct {
	Name string `toml:"name"`
	Jobs []Job  `toml:"jobs"`
}

type Job struct {
	Name    string            `toml:"name"`
	Host    []string          `toml:"host"`
	Script  string            `toml:"script"`
	After   string            `toml:"after"`
	Env     map[string]string `toml:"env"`
	FileIn  []string          `toml:"file_in"`
	FileOut []string          `toml:"file_out"`
}

type RewrittenSpec struct {
	Name string         `toml:"name"`
	Jobs []RewrittenJob `toml:"jobs"`
}

type RewrittenJob struct {
	Name    string            `toml:"name"`
	Host    string            `toml:"host"`
	Script  string            `toml:"script"`
	After   string            `toml:"after"`
	Env     map[string]string `toml:"env"`
	FileIn  []string          `toml:"file_in"`
	FileOut []string          `toml:"file_out"`
}

func (x Spec) Rewrite(ctx context.Context, kv map[string]any) RewrittenSpec {
	rw := RewrittenSpec{
		Name: textutil.Apply(ctx, x.Name, kv),
	}
	kv["hog_schedule"] = rw.Name
	for _, job := range x.Jobs {
		rw.Jobs = append(rw.Jobs, job.Rewrite(ctx, kv)...)
	}
	delete(kv, "hog_schedule")
	return rw
}

func (x Job) Rewrite(ctx context.Context, kv map[string]any) []RewrittenJob {
	rw := []RewrittenJob{}
	kv["hog_host_count"] = len(x.Host)
	for i, host := range x.Host {
		kv["hog_host"] = host
		kv["hog_host_index"] = i
		kv["hog_host_index_plus1"] = i + 1
		//
		rwEnv := map[string]string{}
		for envKey, envValue := range x.Env {
			rwEnv[textutil.Apply(ctx, envKey, kv)] = textutil.Apply(ctx, envValue, kv)
		}
		//
		rwJob := RewrittenJob{
			Name:    textutil.Apply(ctx, x.Name, kv),
			Host:    textutil.Apply(ctx, host, kv),
			Script:  textutil.Apply(ctx, x.Script, kv),
			After:   textutil.Apply(ctx, x.After, kv),
			Env:     rwEnv,
			FileIn:  make([]string, len(x.FileIn)),
			FileOut: make([]string, len(x.FileOut)),
		}
		for i := range x.FileIn {
			rwJob.FileIn[i] = textutil.Apply(ctx, x.FileIn[i], kv)
		}
		for i := range x.FileOut {
			rwJob.FileOut[i] = textutil.Apply(ctx, x.FileOut[i], kv)
		}
		rw = append(rw, rwJob)
	}
	delete(kv, "hog_host_count")
	delete(kv, "hog_host")
	delete(kv, "hog_host_index")
	delete(kv, "hog_host_index_plus1")
	return rw
}

func (x RewrittenSpec) Hosts() proto.HogHosts {
	r := proto.HogHosts{}
	for i := range x.Jobs {
		r = append(r, x.Jobs[i].Host)
	}
	return r
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
		env := cloneMap(job.Env)
		for n, v := range inject {
			env[n] = v
		}
		env["HOG_HOST"] = job.Host
		envs = append(envs, env)
	}
	return envs
}

func cloneMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func Parse(r io.Reader) (*Spec, error) {
	var spec Spec
	dec := toml.NewDecoder(r)
	if _, err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode hog schedule: %w", err)
	}
	if err := validate(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func validate(spec *Spec) error {
	if spec.Name == "" {
		return errors.New("missing workflow name")
	}

	if len(spec.Jobs) == 0 {
		return errors.New("no [[jobs]] entries found")
	}
	seenName := make(map[string]struct{}, len(spec.Jobs))
	seenHost := make(map[string]struct{}, len(spec.Jobs))
	for i, job := range spec.Jobs {

		if job.Name == "" {
			return fmt.Errorf("scripts[%d]: name is empty", i)
		}
		if _, ok := seenName[job.Name]; ok {
			return fmt.Errorf("duplicate job name %q", job.Name)
		}
		seenName[job.Name] = struct{}{}

		for _, host := range job.Host {
			if host == "" {
				return fmt.Errorf("scripts[%d]: host is empty", i)
			}
			if _, ok := seenHost[host]; ok {
				return fmt.Errorf("duplicate job host %q", host)
			}
			seenHost[host] = struct{}{}
		}

		if job.Script == "" {
			return fmt.Errorf("scripts[%d] (%q): script body is empty", i, job.Name)
		}
	}
	return nil
}
