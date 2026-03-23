package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &Config{
		Port:            8081,
		PollIntervalSec: 1,
		PiHoleAddress:   "http://localhost:9999",
		DBPath:          ":memory:",
	}
	cats := &CategoryMap{}
	cats.Categories = map[string][]string{}
	cats.lookup = make(map[string]string)
	pihole := NewPiHoleClient("http://localhost:9999")

	return NewServer(cfg, db, cats, pihole)
}

func TestHandleSSE_ContentType(t *testing.T) {
	srv := newTestServer(t)

	// Use a context with timeout so the SSE handler exits quickly
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	srv.HandleSSE(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}
}

func TestHandleSSE_EventFormat(t *testing.T) {
	srv := newTestServer(t)

	// Short-lived context: allow one tick cycle
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	srv.HandleSSE(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "event: bandwidth") {
		t.Errorf("expected 'event: bandwidth' in SSE output, got: %s", body)
	}
	if !strings.Contains(body, "data: ") {
		t.Errorf("expected 'data: ' in SSE output, got: %s", body)
	}
}

func TestHandleSSE_Disconnect(t *testing.T) {
	srv := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.HandleSSE(w, req)
		close(done)
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Handler exited -- test passes
	case <-time.After(3 * time.Second):
		t.Error("HandleSSE did not exit after context cancel")
	}
}

func TestCapStatusHTML_Green(t *testing.T) {
	// 30% usage: 6GB of 20GB
	html := renderCapStatusHTML(6*1024*1024*1024, 20*1024*1024*1024)
	if !strings.Contains(html, "green") {
		t.Errorf("expected green class at 30%%, got: %s", html)
	}
	if !strings.Contains(html, "20.0 GB") {
		t.Errorf("expected '20.0 GB' in cap status, got: %s", html)
	}
}

func TestCapStatusHTML_Red(t *testing.T) {
	// 95% usage: 19GB of 20GB
	html := renderCapStatusHTML(19*1024*1024*1024, 20*1024*1024*1024)
	if !strings.Contains(html, "red") {
		t.Errorf("expected red class at 95%%, got: %s", html)
	}
}

func TestRenderAlertHTML(t *testing.T) {
	// 50% threshold
	html := renderAlertHTML(50.0)
	if !strings.Contains(html, "alert-warning") {
		t.Errorf("expected alert-warning at 50%%, got: %s", html)
	}

	// 90% threshold
	html = renderAlertHTML(90.0)
	if !strings.Contains(html, "alert-danger") {
		t.Errorf("expected alert-danger at 90%%, got: %s", html)
	}

	// 30% -- no alert
	html = renderAlertHTML(30.0)
	if html != "" {
		t.Errorf("expected empty alert at 30%%, got: %s", html)
	}
}
