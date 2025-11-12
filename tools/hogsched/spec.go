package hogsched

import (
	"errors"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
)

type Spec struct {
	Jobs []Job `toml:"jobs"`
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
