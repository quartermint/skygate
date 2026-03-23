// Package main implements the SkyGate dashboard daemon.
//
// The dashboard daemon is the core engine of the usage dashboard: it polls
// nftables per-MAC counters every 5 seconds, queries Pi-hole for DNS stats,
// calculates bandwidth savings, streams events to the dashboard via SSE,
// and serves REST endpoints for settings and captive portal acceptance.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultConfigPath = "/data/skygate/dashboard.yaml"
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to dashboard YAML config")
	flag.Parse()

	log.SetPrefix("[skygate-dashboard] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Load config
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded: port=%d, poll=%ds, pihole=%s", cfg.Port, cfg.PollIntervalSec, cfg.PiHoleAddress)

	// Initialize SQLite database
	db, err := NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	log.Printf("Database initialized at %s", cfg.DBPath)

	// Load domain categories
	cats, err := LoadCategories(cfg.CategoriesFile)
	if err != nil {
		log.Printf("WARN: Failed to load categories: %v (using empty map)", err)
		cats = &CategoryMap{
			Categories: map[string][]string{},
		}
		cats.lookup = make(map[string]string)
	}

	// Initialize Pi-hole client
	pihole := NewPiHoleClient(cfg.PiHoleAddress)
	if cfg.PiHolePassword != "" {
		if err := pihole.Authenticate(cfg.PiHolePassword); err != nil {
			log.Printf("WARN: Pi-hole auth failed: %v (some features may be limited)", err)
		} else {
			log.Println("Pi-hole authentication successful")
		}
	}

	// Create server
	srv := NewServer(cfg, db, cats, pihole)

	// Register HTTP routes
	mux := http.NewServeMux()

	// SSE endpoint
	mux.HandleFunc("/api/events", srv.HandleSSE)

	// REST API
	mux.HandleFunc("/api/stats/devices", srv.HandleGetDevices)
	mux.HandleFunc("/api/stats/domains", srv.HandleGetDomains)
	mux.HandleFunc("/api/stats/savings", srv.HandleGetSavings)
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.HandleGetSettings(w, r)
		case http.MethodPut:
			srv.HandlePutSettings(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Captive portal
	mux.HandleFunc("/api/captive/accept", srv.HandleCaptiveAccept)

	// Static file serving (dashboard HTML, JS, CSS)
	if cfg.StaticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(cfg.StaticDir)))
	}

	// HTTP server
	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 0, // SSE requires no write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		httpSrv.Shutdown(shutdownCtx)
	}()

	// Start data collection polling loop
	go srv.StartPolling(ctx)

	log.Printf("Dashboard daemon starting on :%d", cfg.Port)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
	log.Println("Dashboard daemon stopped")
}
