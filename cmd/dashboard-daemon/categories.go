package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CategoryMap holds a mapping of domains to traffic categories.
// The YAML structure maps category names to lists of domains.
type CategoryMap struct {
	Categories map[string][]string `yaml:"categories"`
	// lookup is the internal reverse map: domain -> category.
	lookup map[string]string
}

// LoadCategories reads a domain-categories YAML file and builds
// both exact-match and suffix-match lookup maps.
func LoadCategories(path string) (*CategoryMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading categories %s: %w", path, err)
	}
	var cm CategoryMap
	if err := yaml.Unmarshal(data, &cm); err != nil {
		return nil, fmt.Errorf("parsing categories %s: %w", path, err)
	}
	cm.buildLookup()
	return &cm, nil
}

// buildLookup creates the internal domain -> category reverse map.
func (cm *CategoryMap) buildLookup() {
	cm.lookup = make(map[string]string)
	for category, domains := range cm.Categories {
		for _, domain := range domains {
			cm.lookup[strings.ToLower(domain)] = category
		}
	}
}

// Categorize returns the category for a given domain.
// It checks exact match first, then walks up domain labels for subdomain matching.
// Returns "Other" if no match is found.
func (cm *CategoryMap) Categorize(domain string) string {
	domain = strings.ToLower(domain)

	// Exact match first.
	if cat, ok := cm.lookup[domain]; ok {
		return cat
	}

	// Walk up domain labels for subdomain matching.
	// e.g., "m.facebook.com" -> check "facebook.com" -> check "com"
	parts := strings.SplitN(domain, ".", 2)
	for len(parts) == 2 {
		parent := parts[1]
		if cat, ok := cm.lookup[parent]; ok {
			return cat
		}
		parts = strings.SplitN(parent, ".", 2)
	}

	return "Other"
}
