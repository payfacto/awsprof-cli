// Package config loads the optional ~/.awsprof.yaml settings file.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds awsprof settings. All fields are optional.
type Config struct {
	// Prefixes are tried in order when resolving a short profile name.
	Prefixes []string `yaml:"prefixes"`
	// Aliases maps a shortcut to a profile name. The target may be a full
	// profile name or a short name resolved through Prefixes.
	Aliases map[string]string `yaml:"aliases"`
}

func defaults() Config {
	return Config{Prefixes: []string{"payfacto-"}}
}

// DefaultPath returns the default config path (~/.awsprof.yaml).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".awsprof.yaml"
	}
	return filepath.Join(home, ".awsprof.yaml")
}

// Load reads the config file at path. A missing file yields defaults and no
// error. A present file with no prefixes also falls back to the default prefix.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Prefixes) == 0 {
		cfg.Prefixes = defaults().Prefixes
	}
	return cfg, nil
}
