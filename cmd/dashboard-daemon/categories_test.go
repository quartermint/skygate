package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCategories_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "categories.yaml")
	content := []byte(`categories:
  Social Media:
    - facebook.com
    - instagram.com
    - twitter.com
  Streaming:
    - youtube.com
    - netflix.com
  Aviation:
    - foreflight.com
    - garmin.com
  Other: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cm, err := LoadCategories(path)
	if err != nil {
		t.Fatalf("LoadCategories returned error: %v", err)
	}

	cat := cm.Categorize("facebook.com")
	if cat != "Social Media" {
		t.Errorf("expected Social Media for facebook.com, got %s", cat)
	}
	cat = cm.Categorize("youtube.com")
	if cat != "Streaming" {
		t.Errorf("expected Streaming for youtube.com, got %s", cat)
	}
}

func TestCategorize_Known(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "categories.yaml")
	content := []byte(`categories:
  Social Media:
    - facebook.com
    - instagram.com
  Streaming:
    - youtube.com
  Other: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cm, err := LoadCategories(path)
	if err != nil {
		t.Fatalf("LoadCategories returned error: %v", err)
	}

	if cat := cm.Categorize("facebook.com"); cat != "Social Media" {
		t.Errorf("expected Social Media, got %s", cat)
	}
	if cat := cm.Categorize("instagram.com"); cat != "Social Media" {
		t.Errorf("expected Social Media, got %s", cat)
	}
}

func TestCategorize_Subdomain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "categories.yaml")
	content := []byte(`categories:
  Social Media:
    - facebook.com
  Other: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cm, err := LoadCategories(path)
	if err != nil {
		t.Fatalf("LoadCategories returned error: %v", err)
	}

	// Subdomains should match parent domain
	if cat := cm.Categorize("m.facebook.com"); cat != "Social Media" {
		t.Errorf("expected Social Media for m.facebook.com, got %s", cat)
	}
	if cat := cm.Categorize("www.facebook.com"); cat != "Social Media" {
		t.Errorf("expected Social Media for www.facebook.com, got %s", cat)
	}
	if cat := cm.Categorize("api.facebook.com"); cat != "Social Media" {
		t.Errorf("expected Social Media for api.facebook.com, got %s", cat)
	}
}

func TestCategorize_Unknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "categories.yaml")
	content := []byte(`categories:
  Social Media:
    - facebook.com
  Other: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cm, err := LoadCategories(path)
	if err != nil {
		t.Fatalf("LoadCategories returned error: %v", err)
	}

	if cat := cm.Categorize("randomsite.xyz"); cat != "Other" {
		t.Errorf("expected Other for unknown domain, got %s", cat)
	}
}
