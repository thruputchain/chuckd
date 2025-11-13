package libtools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type DotHogtools struct {
	Lockd  string `toml:"lockd"`
	JobDir string `toml:"job_dir"`
}

func FindDotHogtools(ctx context.Context) (*DotHogtools, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, ".hogtools")

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var cfg DotHogtools
	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &cfg, nil
}

func DeriveDotHogtools(ctx context.Context) *DotHogtools {
	dot, _ := FindDotHogtools(ctx)
	if dot == nil {
		dot = &DotHogtools{}
	}
	if envLockd := os.Getenv("HOG_LOCKD"); envLockd != "" {
		dot.Lockd = envLockd
	}
	if envJobDir := os.Getenv("HOG_JOB_DIR"); envJobDir != "" {
		dot.JobDir = envJobDir
	}
	return dot
}
