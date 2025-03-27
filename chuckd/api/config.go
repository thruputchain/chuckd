package api

import (
	_ "gopkg.in/yaml.v3"
)

const (
	LocalAgentPath           = ".chuckd"
	LocalAgentTempPath       = "chuckd"
	LocalAgentConfigFilebase = "chuckd.yaml"
)

type Config struct {
	// CacheDir        string `json:"cache_dir" yaml:"cache_dir"`
	// CacheTTLSeconds int    `json:"cache_ttl_seconds" yaml:"cache_ttl_seconds"`
}

// func (cfg Config) CacheTTL() time.Duration {
// 	return time.Second * time.Duration(cfg.CacheTTLSeconds)
// }
