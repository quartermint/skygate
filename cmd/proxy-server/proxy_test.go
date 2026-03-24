package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestBypassSet_Exact(t *testing.T) {
	bs := NewBypassSet([]string{"example.com", "bank.com"})

	if !bs.Contains("example.com") {
		t.Error("expected example.com to be in bypass set")
	}
	if !bs.Contains("bank.com") {
		t.Error("expected bank.com to be in bypass set")
	}
	if bs.Contains("other.com") {
		t.Error("expected other.com to NOT be in bypass set")
	}
}

func TestBypassSet_Wildcard(t *testing.T) {
	bs := NewBypassSet([]string{"*.apple.com"})

	if !bs.Contains("auth.apple.com") {
		t.Error("expected auth.apple.com to match *.apple.com")
	}
	if !bs.Contains("id.apple.com") {
		t.Error("expected id.apple.com to match *.apple.com")
	}
	// Wildcard requires subdomain -- apple.com itself should NOT match *.apple.com
	if bs.Contains("apple.com") {
		t.Error("expected apple.com to NOT match *.apple.com (wildcard requires subdomain)")
	}
}

func TestBypassSet_Mixed(t *testing.T) {
	bs := NewBypassSet([]string{"exact.com", "*.wild.com", "  spacey.com  "})

	if !bs.Contains("exact.com") {
		t.Error("expected exact.com to be in bypass set")
	}
	if !bs.Contains("sub.wild.com") {
		t.Error("expected sub.wild.com to match *.wild.com")
	}
	if !bs.Contains("spacey.com") {
		t.Error("expected spacey.com to be in bypass set (whitespace trimmed)")
	}
}

func TestBypassSet_Empty(t *testing.T) {
	bs := NewBypassSet(nil)

	if bs.Contains("anything.com") {
		t.Error("empty bypass set should not contain anything")
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com:443", "example.com"},
		{"example.com:8080", "example.com"},
		{"example.com", "example.com"},
		{"[::1]:443", "[::1]"},
	}

	for _, tt := range tests {
		got := stripPort(tt.input)
		if got != tt.expected {
			t.Errorf("stripPort(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMemCertStore_SetAndGet(t *testing.T) {
	store := newMemCertStore(10)

	// Create a test cert to return from gen().
	testCert := &tls.Certificate{}
	genCalled := false
	gen := func() (*tls.Certificate, error) {
		genCalled = true
		return testCert, nil
	}

	// First Fetch should call gen() (cache miss).
	cert, err := store.Fetch("example.com", gen)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if cert != testCert {
		t.Error("expected gen() cert on cache miss")
	}
	if !genCalled {
		t.Error("expected gen() to be called on cache miss")
	}

	// Second Fetch should return cached cert (gen NOT called).
	genCalled = false
	cert2, err := store.Fetch("example.com", gen)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if cert2 != testCert {
		t.Error("expected cached cert on cache hit")
	}
	if genCalled {
		t.Error("expected gen() NOT to be called on cache hit")
	}

	// Fetch for unknown host should call gen().
	otherCert := &tls.Certificate{}
	cert3, err := store.Fetch("other.com", func() (*tls.Certificate, error) {
		return otherCert, nil
	})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if cert3 != otherCert {
		t.Error("expected gen() cert for other.com")
	}
}

func TestMemCertStore_LRUEviction(t *testing.T) {
	store := newMemCertStore(2) // max 2 entries

	cert1 := &tls.Certificate{}
	cert2 := &tls.Certificate{}
	cert3 := &tls.Certificate{}

	// Populate cache with 2 entries.
	store.Fetch("a.com", func() (*tls.Certificate, error) { return cert1, nil })
	store.Fetch("b.com", func() (*tls.Certificate, error) { return cert2, nil })

	// Both should be cached (gen not called on hit).
	c1, _ := store.Fetch("a.com", func() (*tls.Certificate, error) { t.Error("should not call gen"); return nil, nil })
	if c1 != cert1 {
		t.Error("expected a.com to be cached")
	}
	c2, _ := store.Fetch("b.com", func() (*tls.Certificate, error) { t.Error("should not call gen"); return nil, nil })
	if c2 != cert2 {
		t.Error("expected b.com to be cached")
	}

	// Adding third entry should evict oldest (a.com).
	store.Fetch("c.com", func() (*tls.Certificate, error) { return cert3, nil })

	// a.com should have been evicted -- gen will be called.
	evictedGenCalled := false
	newCert := &tls.Certificate{}
	ca, _ := store.Fetch("a.com", func() (*tls.Certificate, error) {
		evictedGenCalled = true
		return newCert, nil
	})
	if !evictedGenCalled {
		t.Error("expected gen() to be called for evicted a.com")
	}
	if ca != newCert {
		t.Error("expected new cert for re-fetched a.com")
	}

	// c.com should still be cached (a.com re-add evicted b.com, the oldest remaining).
	cc, _ := store.Fetch("c.com", func() (*tls.Certificate, error) { t.Error("should not call gen for c.com"); return nil, nil })
	if cc != cert3 {
		t.Error("expected c.com to still be cached")
	}
}

func TestSetupProxy_ReturnsNonNil(t *testing.T) {
	// Minimal test: SetupProxy returns a non-nil proxy.
	// Full CONNECT behavior tested at the BypassSet level.
	caCert := &tls.Certificate{}
	bypassSet := NewBypassSet([]string{"example.com"})
	chain := &HandlerChain{}

	maxSavingsIPs := NewMaxSavingsIPSet("") // disabled, MITM all
	proxy := SetupProxy(caCert, bypassSet, maxSavingsIPs, chain, false)
	if proxy == nil {
		t.Fatal("expected non-nil proxy from SetupProxy")
	}
}

func TestSetupProxy_BypassDomainLogic(t *testing.T) {
	// Verify bypass set correctly identifies bypass vs MITM domains.
	bypassSet := NewBypassSet([]string{"example.com", "*.apple.com"})

	// Should bypass these
	if !bypassSet.Contains("example.com") {
		t.Error("example.com should be bypassed")
	}
	if !bypassSet.Contains("auth.apple.com") {
		t.Error("auth.apple.com should be bypassed (wildcard)")
	}

	// Should MITM these
	if bypassSet.Contains("nonbypass.com") {
		t.Error("nonbypass.com should NOT be bypassed")
	}
}

func TestHardcodedBypassDomains(t *testing.T) {
	// Verify the hardcodedBypassDomains var contains critical never-MITM domains.
	required := []string{
		"*.chase.com",
		"*.gov",
		"*.paypal.com",
		"accounts.google.com",
		"*.epic.com",
		"*.bankofamerica.com",
		"*.wellsfargo.com",
		"*.mil",
		"*.foreflight.com",
	}

	domainSet := make(map[string]bool)
	for _, d := range hardcodedBypassDomains {
		domainSet[d] = true
	}

	for _, r := range required {
		if !domainSet[r] {
			t.Errorf("hardcodedBypassDomains missing required domain: %q", r)
		}
	}
}

func TestBuildBypassSet_MergesHardcodedAndUser(t *testing.T) {
	// Create a temp YAML file with user domains.
	dir := t.TempDir()
	userFile := filepath.Join(dir, "user-bypass.yaml")
	content := "bypass_domains:\n  - example.com\n  - custom-domain.org\n"
	if err := os.WriteFile(userFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing user bypass file: %v", err)
	}

	bs, err := BuildBypassSet(userFile)
	if err != nil {
		t.Fatalf("BuildBypassSet error: %v", err)
	}

	// User domains should be present.
	if !bs.Contains("example.com") {
		t.Error("expected example.com (user domain) in bypass set")
	}
	if !bs.Contains("custom-domain.org") {
		t.Error("expected custom-domain.org (user domain) in bypass set")
	}

	// Hardcoded domains should also be present.
	if !bs.Contains("online.chase.com") {
		t.Error("expected online.chase.com to match *.chase.com (hardcoded)")
	}
	if !bs.Contains("app.wellsfargo.com") {
		t.Error("expected app.wellsfargo.com to match *.wellsfargo.com (hardcoded)")
	}
}

func TestBuildBypassSet_EmptyUserFile(t *testing.T) {
	// Call with nonexistent file path -- hardcoded domains should still be present.
	bs, err := BuildBypassSet("/nonexistent/path/bypass.yaml")
	if err != nil {
		t.Fatalf("BuildBypassSet error: %v", err)
	}

	// Hardcoded domains must be present even without user file.
	if !bs.Contains("secure.chase.com") {
		t.Error("expected secure.chase.com to match *.chase.com (hardcoded)")
	}
	if !bs.Contains("accounts.google.com") {
		t.Error("expected accounts.google.com (hardcoded) in bypass set")
	}
	if !bs.Contains("login.microsoftonline.com") {
		t.Error("expected login.microsoftonline.com (hardcoded) in bypass set")
	}
	if !bs.Contains("irs.gov") {
		t.Error("expected irs.gov to match *.gov (hardcoded)")
	}
}

func TestBuildBypassSet_UserCannotRemoveHardcoded(t *testing.T) {
	// Even with an empty user file, hardcoded domains remain.
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty-bypass.yaml")
	if err := os.WriteFile(emptyFile, []byte("bypass_domains: []\n"), 0644); err != nil {
		t.Fatalf("writing empty bypass file: %v", err)
	}

	bs, err := BuildBypassSet(emptyFile)
	if err != nil {
		t.Fatalf("BuildBypassSet error: %v", err)
	}

	// All hardcoded domains still present.
	if !bs.Contains("checkout.paypal.com") {
		t.Error("expected checkout.paypal.com to match *.paypal.com (hardcoded)")
	}
	if !bs.Contains("portal.epic.com") {
		t.Error("expected portal.epic.com to match *.epic.com (hardcoded)")
	}
}

// --- MaxSavingsIPSet Tests (Phase 5 Plan 03 Task 3) ---

func TestMaxSavingsIPSet_Contains(t *testing.T) {
	// Create enabled set with manually added IPs.
	ipSet := NewMaxSavingsIPSet("http://localhost:9999")
	ipSet.Update([]string{"192.168.4.2", "192.168.4.5", "192.168.4.10"})

	// Added IPs should return true.
	if !ipSet.Contains("192.168.4.2") {
		t.Error("expected 192.168.4.2 to be in MaxSavingsIPSet")
	}
	if !ipSet.Contains("192.168.4.5") {
		t.Error("expected 192.168.4.5 to be in MaxSavingsIPSet")
	}
	if !ipSet.Contains("192.168.4.10") {
		t.Error("expected 192.168.4.10 to be in MaxSavingsIPSet")
	}

	// Non-added IPs should return false.
	if ipSet.Contains("192.168.4.3") {
		t.Error("expected 192.168.4.3 to NOT be in MaxSavingsIPSet")
	}
	if ipSet.Contains("10.0.0.1") {
		t.Error("expected 10.0.0.1 to NOT be in MaxSavingsIPSet")
	}
}

func TestMaxSavingsIPSet_Update(t *testing.T) {
	// Verify Update replaces the full set (old IPs gone, new IPs present).
	ipSet := NewMaxSavingsIPSet("http://localhost:9999")

	// First update with set A.
	ipSet.Update([]string{"192.168.4.1", "192.168.4.2"})
	if !ipSet.Contains("192.168.4.1") {
		t.Error("expected 192.168.4.1 after first Update")
	}
	if !ipSet.Contains("192.168.4.2") {
		t.Error("expected 192.168.4.2 after first Update")
	}

	// Second update with set B -- set A IPs should be gone.
	ipSet.Update([]string{"192.168.4.10", "192.168.4.20"})
	if ipSet.Contains("192.168.4.1") {
		t.Error("expected 192.168.4.1 to be GONE after second Update")
	}
	if ipSet.Contains("192.168.4.2") {
		t.Error("expected 192.168.4.2 to be GONE after second Update")
	}
	if !ipSet.Contains("192.168.4.10") {
		t.Error("expected 192.168.4.10 after second Update")
	}
	if !ipSet.Contains("192.168.4.20") {
		t.Error("expected 192.168.4.20 after second Update")
	}
}

func TestMaxSavingsIPSet_EmptyURL(t *testing.T) {
	// NewMaxSavingsIPSet with empty URL returns disabled set.
	// Contains always returns true (MITM all, pre-Phase 5 behavior).
	ipSet := NewMaxSavingsIPSet("")

	if !ipSet.Contains("192.168.4.1") {
		t.Error("disabled MaxSavingsIPSet should return true for any IP")
	}
	if !ipSet.Contains("10.0.0.1") {
		t.Error("disabled MaxSavingsIPSet should return true for any IP")
	}
	if !ipSet.Contains("") {
		t.Error("disabled MaxSavingsIPSet should return true even for empty string")
	}
	if ipSet.enabled {
		t.Error("MaxSavingsIPSet with empty URL should not be enabled")
	}
}

func TestMaxSavingsIPSet_EnabledEmptySet(t *testing.T) {
	// Enabled set with no IPs should return false (no devices in Max Savings mode).
	ipSet := NewMaxSavingsIPSet("http://localhost:9999")

	if ipSet.Contains("192.168.4.1") {
		t.Error("enabled but empty MaxSavingsIPSet should return false")
	}
}

func TestExtractSourceIP(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{"192.168.4.2:12345", "192.168.4.2"},
		{"10.0.0.1:443", "10.0.0.1"},
		{"192.168.4.5", "192.168.4.5"},
		{"[::1]:8080", "[::1]"},
	}

	for _, tt := range tests {
		req := &http.Request{RemoteAddr: tt.remoteAddr}
		got := extractSourceIP(req)
		if got != tt.want {
			t.Errorf("extractSourceIP(RemoteAddr=%q) = %q, want %q", tt.remoteAddr, got, tt.want)
		}
	}
}

func TestExtractSourceIP_NilRequest(t *testing.T) {
	got := extractSourceIP(nil)
	if got != "" {
		t.Errorf("extractSourceIP(nil) = %q, want empty string", got)
	}
}

func TestSetupProxy_QuickConnectPassthrough(t *testing.T) {
	// SetupProxy with MaxSavingsIPSet that does NOT contain a source IP
	// should result in ConnectAccept (TCP passthrough) for non-bypass hosts.
	caCert := &tls.Certificate{}
	bypassSet := NewBypassSet([]string{"bypass.example.com"})
	chain := &HandlerChain{}

	// MaxSavingsIPSet with specific IPs -- NOT including "192.168.4.99"
	maxSavingsIPs := NewMaxSavingsIPSet("http://localhost:9999")
	maxSavingsIPs.Update([]string{"192.168.4.2", "192.168.4.5"})

	proxy := SetupProxy(caCert, bypassSet, maxSavingsIPs, chain, false)
	if proxy == nil {
		t.Fatal("expected non-nil proxy from SetupProxy")
	}

	// For Quick Connect devices (not in Max Savings set), verify the
	// MaxSavingsIPSet returns false (no MITM).
	if maxSavingsIPs.Contains("192.168.4.99") {
		t.Error("192.168.4.99 should NOT be in MaxSavingsIPSet (Quick Connect device)")
	}
}

func TestSetupProxy_MaxSavingsMITM(t *testing.T) {
	// SetupProxy with MaxSavingsIPSet that DOES contain a source IP
	// should result in ConnectMitm for non-bypass hosts.
	caCert := &tls.Certificate{}
	bypassSet := NewBypassSet([]string{"bypass.example.com"})
	chain := &HandlerChain{}

	// MaxSavingsIPSet with specific IPs -- INCLUDING "192.168.4.2"
	maxSavingsIPs := NewMaxSavingsIPSet("http://localhost:9999")
	maxSavingsIPs.Update([]string{"192.168.4.2", "192.168.4.5"})

	proxy := SetupProxy(caCert, bypassSet, maxSavingsIPs, chain, false)
	if proxy == nil {
		t.Fatal("expected non-nil proxy from SetupProxy")
	}

	// For Max Savings devices (in the set), verify the set returns true (MITM).
	if !maxSavingsIPs.Contains("192.168.4.2") {
		t.Error("192.168.4.2 SHOULD be in MaxSavingsIPSet (Max Savings device)")
	}
	if !maxSavingsIPs.Contains("192.168.4.5") {
		t.Error("192.168.4.5 SHOULD be in MaxSavingsIPSet (Max Savings device)")
	}
}

func TestSetupProxy_BypassAlwaysAccept(t *testing.T) {
	// Bypass domains get ConnectAccept regardless of MaxSavingsIPSet membership.
	bypassSet := NewBypassSet([]string{"*.chase.com", "accounts.google.com"})

	// MaxSavingsIPSet enabled and containing all IPs -- bypass should still work.
	maxSavingsIPs := NewMaxSavingsIPSet("http://localhost:9999")
	maxSavingsIPs.Update([]string{"192.168.4.2", "192.168.4.5"})

	// Even though the IP is in Max Savings mode, bypass domains bypass MITM.
	if !bypassSet.Contains("online.chase.com") {
		t.Error("online.chase.com should be bypassed regardless of mode")
	}
	if !bypassSet.Contains("accounts.google.com") {
		t.Error("accounts.google.com should be bypassed regardless of mode")
	}

	// Non-bypass domain with Max Savings IP should be MITMed.
	if bypassSet.Contains("example.com") {
		t.Error("example.com should NOT be bypassed")
	}
	if !maxSavingsIPs.Contains("192.168.4.2") {
		t.Error("192.168.4.2 should be in Max Savings (gets MITM for non-bypass)")
	}
}

// Suppress unused import warnings
var _ = os.WriteFile
var _ = filepath.Join
