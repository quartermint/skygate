package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the proxy server configuration loaded from YAML.
type Config struct {
	ListenAddr        string       `yaml:"listen_addr"`
	CACertPath        string       `yaml:"ca_cert_path"`
	CAKeyPath         string       `yaml:"ca_key_path"`
	CADownloadAddr    string       `yaml:"ca_download_addr"`
	DBPath            string       `yaml:"db_path"`
	BypassDomainsFile string       `yaml:"bypass_domains_file"`
	Verbose           bool         `yaml:"verbose"`
	Image             ImageConfig  `yaml:"image"`
	Minify            MinifyConfig `yaml:"minify"`
	Log               LogConfig    `yaml:"log"`
}

// ImageConfig holds image transcoding settings.
type ImageConfig struct {
	Quality         int `yaml:"quality"`
	MaxWidth        int `yaml:"max_width"`
	TimeoutMS       int `yaml:"timeout_ms"`
	MaxSizeBytes    int `yaml:"max_size_bytes"`
	ConcurrentLimit int `yaml:"concurrent_limit"`
}

// MinifyConfig holds text minification settings.
type MinifyConfig struct {
	Enabled bool `yaml:"enabled"`
	HTML    bool `yaml:"html"`
	CSS     bool `yaml:"css"`
	JS      bool `yaml:"js"`
	SVG     bool `yaml:"svg"`
	JSON    bool `yaml:"json"`
}

// LogConfig holds compression logging settings.
type LogConfig struct {
	RetentionDays  int `yaml:"retention_days"`
	BatchIntervalS int `yaml:"batch_interval_s"`
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

// bypassConfig is an internal struct for parsing the bypass domains YAML file.
type bypassConfig struct {
	BypassDomains []string `yaml:"bypass_domains"`
}

// LoadBypassDomains reads a YAML file with a bypass_domains key and returns the domain list.
// These are SNI bypass domains that skip MITM interception (cert-pinned apps).
func LoadBypassDomains(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg bypassConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg.BypassDomains, nil
}
