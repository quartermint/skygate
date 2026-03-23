package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds tunnel monitor configuration loaded from YAML.
type Config struct {
	Interface          string `yaml:"interface"`
	CheckIntervalS     int    `yaml:"check_interval_s"`
	HandshakeTimeoutS  int    `yaml:"handshake_timeout_s"`
	RecoveryThresholdS int    `yaml:"recovery_threshold_s"`
	FailCount          int    `yaml:"fail_count"`
	RecoverCount       int    `yaml:"recover_count"`
	Fwmark             string `yaml:"fwmark"`
	Table              int    `yaml:"table"`
	Priority           int    `yaml:"priority"`
}

// LoadConfig reads and parses a YAML config file.
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
