package tools

import (
	"net/url"
	"strings"
)

// CleanTarget removes protocol and path from target, returns domain/IP only
func CleanTarget(target string) string {
	// Remove trailing slashes
	target = strings.TrimSpace(target)
	target = strings.TrimSuffix(target, "/")

	// Try to parse as URL
	if strings.Contains(target, "://") {
		u, err := url.Parse(target)
		if err == nil && u.Host != "" {
			// Extract just the hostname (no port)
			host := u.Host
			if strings.Contains(host, ":") {
				host = strings.Split(host, ":")[0]
			}
			return host
		}
	}

	// If it looks like a domain/IP with port, extract just the host
	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Return as-is if already looks like IP/domain
	return target
}
