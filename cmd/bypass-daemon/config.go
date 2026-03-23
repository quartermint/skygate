package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the bypass daemon configuration loaded from YAML.
type Config struct {
	BypassDomains []string `yaml:"bypass_domains"`
}

// LoadConfig reads and parses a YAML config file.
// Returns an empty Config (not error) for valid but empty files.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}
