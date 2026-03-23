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
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultConfigPath = "/data/skygate/bypass-domains.yaml"
	defaultInterval   = 60 * time.Second
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to bypass domains YAML config")
	interval := flag.Duration("interval", defaultInterval, "DNS re-resolution interval")
	flag.Parse()

	log.SetPrefix("[skygate-bypass] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded %d bypass domains from %s", len(cfg.BypassDomains), *configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Initial resolution
	runCycle(cfg.BypassDomains)

	// Periodic re-resolution
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Bypass daemon stopped")
			return
		case <-ticker.C:
			runCycle(cfg.BypassDomains)
		}
	}
}

func runCycle(domains []string) {
	ips := ResolveDomains(domains)
	log.Printf("Resolved %d unique IPs from %d domains", len(ips), len(domains))
	if err := UpdateNftSet(ips); err != nil {
		log.Printf("ERROR: failed to update nft set: %v", err)
	}
}
