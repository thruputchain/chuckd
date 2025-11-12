package hogsched

import (
	"errors"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	proto "github.com/groundhogchain/protocol-go"
)

type Spec struct {
	Name string `toml:"name"`
	Jobs []Job  `toml:"jobs"`
}

func (x Spec) HostObjects() proto.LockObjects {
	r := make(proto.LockObjects, len(x.Jobs))
	for i := range x.Jobs {
		r[i] = x.Jobs[i].Host
	}
	return r
}

func (x Spec) Scripts() []string {
	r := make([]string, len(x.Jobs))
	for i := range x.Jobs {
		r[i] = x.Jobs[i].Script
	}
	return r
}

type Job struct {
	Name   string `toml:"name"`
	Host   string `toml:"host"`
	Script string `toml:"script"`
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
	for i, s := range spec.Jobs {

		if s.Name == "" {
			return fmt.Errorf("scripts[%d]: name is empty", i)
		}
		if _, ok := seenName[s.Name]; ok {
			return fmt.Errorf("duplicate job name %q", s.Name)
		}
		seenName[s.Name] = struct{}{}

		if s.Host == "" {
			return fmt.Errorf("scripts[%d]: host is empty", i)
		}
		if _, ok := seenHost[s.Host]; ok {
			return fmt.Errorf("duplicate job host %q", s.Host)
		}
		seenHost[s.Host] = struct{}{}

		if s.Script == "" {
			return fmt.Errorf("scripts[%d] (%q): script body is empty", i, s.Name)
		}
	}
	return nil
}
