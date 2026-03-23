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

const defaultConfigPath = "/etc/skygate/proxy.yaml"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to proxy YAML config")
	flag.Parse()

	log.SetPrefix("[skygate-proxy] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 1. Load config
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded: listen=%s, image_quality=%d, max_width=%d",
		cfg.ListenAddr, cfg.Image.Quality, cfg.Image.MaxWidth)

	// 2. Load or generate CA certificate (D-11)
	caCert, err := LoadOrGenerateCA(cfg.CACertPath, cfg.CAKeyPath)
	if err != nil {
		log.Fatalf("Failed to load/generate CA: %v", err)
	}
	log.Printf("CA certificate ready at %s", cfg.CACertPath)

	// 3. Load bypass domains (D-09)
	bypassDomains, err := LoadBypassDomains(cfg.BypassDomainsFile)
	if err != nil {
		log.Printf("WARN: Failed to load bypass domains: %v (using empty list)", err)
		bypassDomains = []string{}
	}
	bypassSet := NewBypassSet(bypassDomains)
	log.Printf("Loaded %d bypass domains", len(bypassDomains))

	// 4. Initialize SQLite database (D-10)
	db, err := NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	log.Printf("Database initialized at %s", cfg.DBPath)

	// 5. Create processing components
	transcoder := NewTranscoder(cfg.Image)
	minifier := NewMinifier(cfg.Minify)
	chain := NewHandlerChain(transcoder, minifier, db, cfg.Verbose)

	// 6. Setup goproxy
	proxy := SetupProxy(caCert, bypassSet, chain, cfg.Verbose)

	// 7. CA cert download endpoint (D-12)
	// Serve CA cert on CADownloadAddr for Phase 5 captive portal to fetch
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ca.crt", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-x509-ca-cert")
			w.Header().Set("Content-Disposition", "attachment; filename=\"skygate-ca.crt\"")
			http.ServeFile(w, r, cfg.CACertPath)
		})
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","service":"skygate-proxy"}`)
		})
		caServer := &http.Server{
			Addr:         cfg.CADownloadAddr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		log.Printf("CA download endpoint listening on %s", cfg.CADownloadAddr)
		if err := caServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("CA download server error: %v", err)
		}
	}()

	// 8. Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// 9. Start proxy server
	proxyServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Proxy needs unlimited write time for large responses
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("Proxy server listening on %s", cfg.ListenAddr)
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		proxyServer.Shutdown(shutdownCtx)
	}()

	if err := proxyServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Proxy server error: %v", err)
	}
	log.Println("Proxy server stopped")
}
