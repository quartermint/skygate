// Package main implements the SkyGate bypass daemon.
//
// The bypass daemon periodically resolves aviation app domains (ForeFlight,
// Garmin Pilot, weather APIs, etc.) and populates nftables native sets with
// the resolved IPs. This ensures safety-critical aviation traffic routes
// directly to Starlink, bypassing any proxy or DNS blocking.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the bypass domain configuration loaded from YAML.
type Config struct {
	BypassDomains []string `yaml:"bypass_domains"`
}

// loadConfig reads and parses the bypass domain YAML configuration file.
func loadConfig(path string) (*Config, error) {
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

// resolveDomains resolves a list of domain names to their IP addresses.
// Wildcard prefixes (*.) are stripped before resolution. Domains that fail
// to resolve are skipped gracefully. Returns a deduplicated list of IPs.
func resolveDomains(domains []string) ([]string, error) {
	seen := make(map[string]bool)
	var ips []string

	for _, domain := range domains {
		// Strip wildcard prefix if present
		d := domain
		if strings.HasPrefix(d, "*.") {
			d = d[2:]
		}

		addrs, err := net.LookupHost(d)
		if err != nil {
			// Graceful skip — domain may not resolve (e.g., typo, offline)
			log.Printf("warning: could not resolve %s: %v", d, err)
			continue
		}

		for _, addr := range addrs {
			if !seen[addr] {
				seen[addr] = true
				ips = append(ips, addr)
			}
		}
	}

	return ips, nil
}

func main() {
	configPath := flag.String("config", "/data/skygate/bypass-domains.yaml", "path to bypass domains YAML config")
	interval := flag.Duration("interval", 60*time.Second, "DNS resolution interval")
	flag.Parse()

	log.SetPrefix("skygate-bypass: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("loaded %d bypass domains from %s", len(cfg.BypassDomains), *configPath)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		cancel()
	}()

	// Main loop: resolve domains and update nftables sets
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Run immediately on startup
	runCycle(cfg)

	for {
		select {
		case <-ctx.Done():
			log.Println("shutdown complete")
			return
		case <-ticker.C:
			runCycle(cfg)
		}
	}
}

// runCycle performs one resolution + nftables update cycle.
func runCycle(cfg *Config) {
	ips, err := resolveDomains(cfg.BypassDomains)
	if err != nil {
		log.Printf("error resolving domains: %v", err)
		return
	}
	log.Printf("resolved %d unique IPs from %d domains", len(ips), len(cfg.BypassDomains))

	if err := updateNftSet(ips); err != nil {
		log.Printf("error updating nftables set: %v", err)
	}
}
