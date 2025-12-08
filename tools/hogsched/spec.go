package hogsched

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/groundhogchain/chuckd/util/textutil"
	"github.com/groundhogchain/lib-go/must"
)

type Spec struct {
	Name string `toml:"name"`
	Jobs []Job  `toml:"jobs"`
}

type Job struct {
	Name    string            `toml:"name"`
	Host    map[string]string `toml:"host"`
	Script  string            `toml:"script"`
	After   string            `toml:"after"`
	Env     map[string]string `toml:"env"`
	FileIn  []string          `toml:"file_in"`
	FileOut []string          `toml:"file_out"`
}

func (x Spec) Hosts() map[string]string {
	h := map[string]string{}
	for _, job := range x.Jobs {
		maps.Copy(h, job.Host)
	}
	return h
}

// bind is host_key -> host_value
func (x *Spec) SubstituteHostVariables(ctx context.Context, assignedHostMap map[string]string) {
	for _, job := range x.Jobs {
		for hk, hv := range job.Host {
			if IsWildcard(hv) {
				assignedHost, ok := assignedHostMap[hk]
				must.Assert(ctx, ok)
				job.Host[hk] = assignedHost
			}
		}
	}
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
	for hostKey, hostValue := range x.Host {
		kv["hog_host_key"] = hostKey
		kv["hog_host"] = hostValue
		//
		rwEnv := map[string]string{}
		for envKey, envValue := range x.Env {
			rwEnv[textutil.Apply(ctx, envKey, kv)] = textutil.Apply(ctx, envValue, kv)
		}
		//
		rwJob := RewrittenJob{
			Name:      textutil.Apply(ctx, x.Name, kv),
			HostKey:   hostKey,
			HostValue: hostValue,
			Script:    textutil.Apply(ctx, x.Script, kv),
			After:     textutil.Apply(ctx, x.After, kv),
			Env:       rwEnv,
			FileIn:    make([]string, len(x.FileIn)),
			FileOut:   make([]string, len(x.FileOut)),
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
	delete(kv, "hog_host_key")
	return rw
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
	seenHostKey := make(map[string]struct{}, len(spec.Jobs))
	seenHostValue := make(map[string]struct{}, len(spec.Jobs))
	for i, job := range spec.Jobs {

		if job.Name == "" {
			return fmt.Errorf("scripts[%d]: name is empty", i)
		}
		if _, ok := seenName[job.Name]; ok {
			return fmt.Errorf("duplicate job name %q", job.Name)
		}
		seenName[job.Name] = struct{}{}

		for hostKey, hostValue := range job.Host {

			// host key syntax
			if !IsIdentifier(hostKey) {
				return fmt.Errorf("scripts[%d]: host key is not an identifier", i)
			}

			// no duplicate host keys
			if _, ok := seenHostKey[hostKey]; ok {
				return fmt.Errorf("duplicate job host key %q", hostKey)
			}
			seenHostKey[hostKey] = struct{}{}

			// host values
			switch {
			case IsHost(hostValue):
				if _, ok := seenHostValue[hostValue]; ok {
					return fmt.Errorf("duplicate job host %q", hostValue)
				}
				seenHostValue[hostValue] = struct{}{}
			case IsWildcard(hostValue):
			default:
				return fmt.Errorf("scripts[%d]: host value %q is not a valid host or wildcard", i, hostValue)
			}

		}

		if job.Script == "" {
			return fmt.Errorf("scripts[%d] (%q): script body is empty", i, job.Name)
		}
	}
	return nil
}

var (
	// RFC-1123-ish hostname: labels of letters, digits, hyphens,
	// cannot start/end with '-', max label length not strictly enforced.
	hostRe = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(?:\.(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?))*$`)

	// programming identifier
	identifierRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

func IsHost(s string) bool {
	return hostRe.MatchString(s)
}

func IsWildcard(s string) bool {
	return s == "*" || s == "" || s == "?"
}

func IsIdentifier(s string) bool {
	return identifierRe.MatchString(s)
}
