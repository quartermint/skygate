// Package main implements the SkyGate tunnel health monitor.
//
// The tunnel monitor checks WireGuard handshake recency and manages
// routing fallback when the tunnel is unreachable. When the tunnel drops,
// traffic falls back to direct routing via Starlink. DNS filtering (Pi-hole)
// continues regardless of tunnel state.
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

const defaultConfigPath = "/data/skygate/tunnel-monitor.yaml"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to tunnel monitor YAML config")
	flag.Parse()

	log.SetPrefix("[skygate-tunnel] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded config: interface=%s, timeout=%ds, check_interval=%ds",
		cfg.Interface, cfg.HandshakeTimeoutS, cfg.CheckIntervalS)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	monitor := NewMonitor(cfg.FailCount, cfg.RecoverCount)
	fbCfg := FallbackConfig{
		Fwmark:   cfg.Fwmark,
		Table:    cfg.Table,
		Priority: cfg.Priority,
	}
	maxAge := time.Duration(cfg.HandshakeTimeoutS) * time.Second

	// Initial check
	runCheck(cfg.Interface, maxAge, monitor, fbCfg)

	ticker := time.NewTicker(time.Duration(cfg.CheckIntervalS) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Tunnel monitor stopped")
			return
		case <-ticker.C:
			runCheck(cfg.Interface, maxAge, monitor, fbCfg)
		}
	}
}

func runCheck(iface string, maxAge time.Duration, mon *Monitor, fbCfg FallbackConfig) {
	output, err := GetHandshakeOutput(iface)
	if err != nil {
		log.Printf("WARN: failed to get handshake output: %v", err)
		// Treat exec failure as unhealthy
		if changed := mon.Update(false); changed {
			handleStateChange(mon.State, fbCfg)
		}
		return
	}

	healthy, age, err := CheckHandshake(output, maxAge)
	if err != nil {
		log.Printf("WARN: handshake check error: %v", err)
		if changed := mon.Update(false); changed {
			handleStateChange(mon.State, fbCfg)
		}
		return
	}

	if changed := mon.Update(healthy); changed {
		handleStateChange(mon.State, fbCfg)
	}

	if healthy {
		log.Printf("tunnel healthy (handshake age: %s)", age.Round(time.Second))
	} else {
		log.Printf("tunnel unhealthy (handshake age: %s, state: %s, fail: %d/%d, recover: %d/%d)",
			age.Round(time.Second), mon.State, mon.FailCount, mon.MaxFail, mon.RecoverCount, mon.MaxRecover)
	}
}

func handleStateChange(state TunnelState, fbCfg FallbackConfig) {
	switch state {
	case StateDegraded:
		log.Printf("STATE CHANGE -> DEGRADED: removing tunnel routing rule (fallback to direct)")
		if err := ExecIPRule(FormatDelRule(fbCfg)); err != nil {
			log.Printf("ERROR: failed to remove tunnel rule: %v", err)
		}
	case StateHealthy:
		log.Printf("STATE CHANGE -> HEALTHY: restoring tunnel routing rule")
		if err := ExecIPRule(FormatAddRule(fbCfg)); err != nil {
			log.Printf("ERROR: failed to restore tunnel rule: %v", err)
		}
	}
}
