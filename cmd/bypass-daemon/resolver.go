package main

import (
	"log"
	"net"
	"strings"
)

// ResolveDomains takes a list of domain strings (possibly with "*." prefix),
// resolves each to IPv4 addresses, and returns a deduplicated list.
// Unresolvable domains are logged and skipped (not errors).
func ResolveDomains(domains []string) []string {
	seen := make(map[string]bool)
	var ips []string

	for _, domain := range domains {
		// Strip wildcard prefix -- resolve base domain
		d := strings.TrimPrefix(domain, "*.")

		addrs, err := net.LookupHost(d)
		if err != nil {
			log.Printf("WARN: could not resolve %s: %v", d, err)
			continue
		}

		for _, addr := range addrs {
			// Only IPv4
			if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
				if !seen[addr] {
					seen[addr] = true
					ips = append(ips, addr)
				}
			}
		}
	}

	return ips
}
