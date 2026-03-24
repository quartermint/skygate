package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the dashboard daemon configuration loaded from YAML.
type Config struct {
	Port            int    `yaml:"port"`
	PollIntervalSec int    `yaml:"poll_interval_sec"`
	PiHoleAddress   string `yaml:"pihole_address"`
	PiHolePassword  string `yaml:"pihole_password"`
	DBPath          string `yaml:"db_path"`
	CategoriesFile  string `yaml:"categories_file"`
	StaticDir       string `yaml:"static_dir"`
	CACertPath     string `yaml:"ca_cert_path"`
	CertBypassFile string `yaml:"cert_bypass_file"`
}

// LoadConfig reads and parses a YAML config file.
// Zero-value fields are replaced with sensible defaults.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

// applyDefaults fills in zero-value fields with production defaults.
func applyDefaults(cfg *Config) {
	if cfg.Port == 0 {
		cfg.Port = 8081
	}
	if cfg.PollIntervalSec == 0 {
		cfg.PollIntervalSec = 5
	}
	if cfg.PiHoleAddress == "" {
		cfg.PiHoleAddress = "http://localhost:8080"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "/data/skygate/dashboard.db"
	}
	if cfg.CategoriesFile == "" {
		cfg.CategoriesFile = "/data/skygate/domain-categories.yaml"
	}
	if cfg.StaticDir == "" {
		cfg.StaticDir = "/opt/skygate/static"
	}
	if cfg.CACertPath == "" {
		cfg.CACertPath = "/data/skygate/ca/root-ca.crt"
	}
	if cfg.CertBypassFile == "" {
		cfg.CertBypassFile = "/data/skygate/cert-bypass-domains.yaml"
	}
}
