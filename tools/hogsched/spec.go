package hogsched

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/BurntSushi/toml"
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
	Env     map[string]string `toml:"env"`
	FileIn  []string          `toml:"file_in"`
	FileOut []string          `toml:"file_out"`
}

func (x Spec) Hosts() proto.LockObjects {
	r := proto.LockObjects{}
	for i := range x.Jobs {
		r = append(r, x.Jobs[i].Host...)
	}
	return r
}

func (x Spec) ScriptsWithJobDir(jobDir string, user string) []string {
	jobDir = filepath.Clean(jobDir)
	prefix := fmt.Sprintf(`
echo "creating job directory" %q
mkdir -p %q
cd %q
echo "clearing job directory"
chmod -R +w * || true
rm -rf *
echo "forwarding github urls through ssh-agent"
hog_fwd
(
`, jobDir, jobDir, jobDir)
	suffix := `
)
`
	r := []string{}
	for _, job := range x.Jobs {
		jobScript := prefix + job.Script + suffix
		for range job.Host {
			r = append(r, jobScript)
		}
	}
	return r
}

func (x Spec) Envs(inject map[string]string) []map[string]string {
	envs := []map[string]string{}
	for _, job := range x.Jobs {
		for _, host := range job.Host {
			env := cloneMap(job.Env)
			for n, v := range inject {
				env[n] = v
			}
			env["HOG_HOST"] = host
			envs = append(envs, env)
		}
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
