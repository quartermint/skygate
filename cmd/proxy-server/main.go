package main

import (
	"context"
	"crypto/tls"
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

	// 2. Load or generate CA certificates
	// Phase 5: Use intermediate CA for MITM leaf signing (root CA key stays on Pi per D-08).
	// The intermediate CA cert+key are provisioned from the Pi during setup.
	var mitmCert *tls.Certificate
	if cfg.IntermediateCACertPath != "" && cfg.IntermediateCAKeyPath != "" {
		// Phase 5 mode: load intermediate CA for MITM
		cert, err := tls.LoadX509KeyPair(cfg.IntermediateCACertPath, cfg.IntermediateCAKeyPath)
		if err != nil {
			log.Printf("WARN: Failed to load intermediate CA: %v (falling back to root CA)", err)
		} else {
			mitmCert = &cert
			log.Printf("Intermediate CA loaded from %s", cfg.IntermediateCACertPath)
		}
	}
	if mitmCert == nil {
		// Fallback: use root/auto-generated CA (pre-Phase 5 behavior)
		cert, err := LoadOrGenerateCA(cfg.CACertPath, cfg.CAKeyPath)
		if err != nil {
			log.Fatalf("Failed to load/generate CA: %v", err)
		}
		mitmCert = cert
		log.Printf("CA certificate ready at %s", cfg.CACertPath)
	}

	// 3. Load bypass domains (hardcoded never-MITM + user YAML per D-14, D-15)
	bypassSet, err := BuildBypassSet(cfg.BypassDomainsFile)
	if err != nil {
		log.Printf("WARN: BuildBypassSet error: %v (using hardcoded only)", err)
		bypassSet = NewBypassSet(hardcodedBypassDomains)
	}
	log.Printf("Bypass set loaded: hardcoded + user domains")

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

	// 5b. Create MaxSavingsIPSet for per-device MITM decision
	// Polls the Pi's dashboard daemon /api/mode/ips endpoint (via WireGuard tunnel)
	maxSavingsIPs := NewMaxSavingsIPSet(cfg.DashboardAPIURL)
	go maxSavingsIPs.StartPolling(10 * time.Second) // Poll every 10 seconds

	// 6. Setup goproxy with intermediate CA (or fallback root CA)
	proxy := SetupProxy(mitmCert, bypassSet, maxSavingsIPs, chain, cfg.Verbose)

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
