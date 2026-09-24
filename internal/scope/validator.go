// Package scope provides scope validation for engagements
package scope

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/registry"
)

// ScopeValidator validates targets against engagement scope
type ScopeValidator struct {
	allowedDomains    []string           // exact domain matches
	allowedDomainWildcards []string      // wildcard patterns (*.example.com)
	allowedIPs        []*net.IPNet       // CIDR ranges
	allowedPorts      map[string][]int   // per-host port ranges
	deniedDomains     []string           // explicitly denied
	deniedIPs        []*net.IPNet        // explicitly denied ranges
	strict            bool               // strict mode: deny by default unless in whitelist
}

// ValidationResult represents scope check result
type ValidationResult struct {
	Valid       bool
	Target      string
	Reason      string
	InScope     bool
	Suggestions []string
}

// NewScopeValidator creates a new scope validator
func NewScopeValidator(strict bool) *ScopeValidator {
	return &ScopeValidator{
		allowedDomains:    make([]string, 0),
		allowedDomainWildcards: make([]string, 0),
		allowedIPs:        make([]*net.IPNet, 0),
		allowedPorts:      make(map[string][]int),
		deniedDomains:     make([]string, 0),
		deniedIPs:         make([]*net.IPNet, 0),
		strict:            strict,
	}
}

// LoadFromRegistry loads scope rules from registry
func (sv *ScopeValidator) LoadFromRegistry(reg *registry.Registry) error {
	for _, pattern := range reg.ScopeRules {
		if err := sv.AddPattern(pattern.Pattern); err != nil {
			log.Printf("Warning: Failed to add scope pattern %s: %v", pattern.Pattern, err)
		}
	}
	return nil
}

// AddPattern adds a domain or CIDR pattern to allowed scope
func (sv *ScopeValidator) AddPattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)

	// Handle CIDR notation
	if strings.Contains(pattern, "/") {
		_, ipnet, err := net.ParseCIDR(pattern)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %s", pattern)
		}
		sv.allowedIPs = append(sv.allowedIPs, ipnet)
		return nil
	}

	// Handle domain wildcards
	if strings.HasPrefix(pattern, "*.") {
		sv.allowedDomainWildcards = append(sv.allowedDomainWildcards, pattern[2:])
		return nil
	}

	// Handle explicit domains
	if !strings.Contains(pattern, ":") {
		sv.allowedDomains = append(sv.allowedDomains, strings.ToLower(pattern))
		return nil
	}

	// Handle domain:port patterns
	parts := strings.Split(pattern, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid domain:port pattern: %s", pattern)
	}
	domain := strings.ToLower(parts[0])
	
	// Parse port (could be range like 80-443)
	portRange := parts[1]
	if strings.Contains(portRange, "-") {
		// Range handling - store as-is for now
		sv.allowedPorts[domain] = append(sv.allowedPorts[domain], 0) // TODO: parse range
	} else {
		// Single port
		var port int
		_, _ = fmt.Sscanf(portRange, "%d", &port)
		sv.allowedPorts[domain] = append(sv.allowedPorts[domain], port)
	}

	return nil
}

// DenyPattern adds a domain or CIDR to denied list
func (sv *ScopeValidator) DenyPattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)

	if strings.Contains(pattern, "/") {
		_, ipnet, err := net.ParseCIDR(pattern)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %s", pattern)
		}
		sv.deniedIPs = append(sv.deniedIPs, ipnet)
		return nil
	}

	sv.deniedDomains = append(sv.deniedDomains, strings.ToLower(pattern))
	return nil
}

// ValidateTarget checks if a target is in scope
func (sv *ScopeValidator) ValidateTarget(target string) *ValidationResult {
	result := &ValidationResult{
		Target:      target,
		Suggestions: make([]string, 0),
	}

	// Parse target - could be IP, domain, URL, or IP:port
	normalizedTarget := sv.normalizeTarget(target)
	if normalizedTarget == "" {
		result.Valid = false
		result.Reason = "Invalid target format"
		return result
	}

	// Check if explicitly denied
	if sv.isDenied(normalizedTarget) {
		result.Valid = false
		result.Reason = "Target is explicitly denied"
		result.InScope = false
		return result
	}

	// Try to parse as IP
	ip := net.ParseIP(sv.extractIP(normalizedTarget))
	if ip != nil {
		return sv.validateIP(ip, result)
	}

	// Parse as domain/hostname
	domain := sv.extractDomain(normalizedTarget)
	return sv.validateDomain(domain, result)
}

// ValidateTargets checks multiple targets
func (sv *ScopeValidator) ValidateTargets(targets []string) []*ValidationResult {
	results := make([]*ValidationResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, sv.ValidateTarget(target))
	}
	return results
}

// validateIP checks if an IP is in scope
func (sv *ScopeValidator) validateIP(ip net.IP, result *ValidationResult) *ValidationResult {
	// Check against allowed CIDR ranges
	for _, ipnet := range sv.allowedIPs {
		if ipnet.Contains(ip) {
			result.Valid = true
			result.InScope = true
			result.Reason = fmt.Sprintf("IP %s is in allowed range %s", ip, ipnet)
			return result
		}
	}

	// If strict mode and not in whitelist, deny
	if sv.strict {
		result.Valid = false
		result.InScope = false
		result.Reason = fmt.Sprintf("IP %s not in any allowed range (strict mode)", ip)
		result.Suggestions = append(result.Suggestions, "Add this IP range to scope if authorized")
		return result
	}

	// Non-strict: allow by default
	result.Valid = true
	result.InScope = true
	result.Reason = fmt.Sprintf("IP %s allowed (non-strict mode)", ip)
	return result
}

// validateDomain checks if a domain is in scope
func (sv *ScopeValidator) validateDomain(domain string, result *ValidationResult) *ValidationResult {
	domain = strings.ToLower(domain)

	// Check exact domain match
	for _, allowed := range sv.allowedDomains {
		if domain == allowed {
			result.Valid = true
			result.InScope = true
			result.Reason = fmt.Sprintf("Domain %s is explicitly allowed", domain)
			return result
		}
	}

	// Check wildcard match
	for _, wildcard := range sv.allowedDomainWildcards {
		if strings.HasSuffix(domain, wildcard) {
			result.Valid = true
			result.InScope = true
			result.Reason = fmt.Sprintf("Domain %s matches allowed wildcard *.%s", domain, wildcard)
			return result
		}
	}

	// If strict mode and not in whitelist, deny
	if sv.strict {
		result.Valid = false
		result.InScope = false
		result.Reason = fmt.Sprintf("Domain %s not in allowed list (strict mode)", domain)
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("Add %s to scope if authorized", domain))
		return result
	}

	// Non-strict: allow by default
	result.Valid = true
	result.InScope = true
	result.Reason = fmt.Sprintf("Domain %s allowed (non-strict mode)", domain)
	return result
}

// isDenied checks explicit deny list
func (sv *ScopeValidator) isDenied(target string) bool {
	target = strings.ToLower(target)

	// Check IP denials
	if ip := net.ParseIP(sv.extractIP(target)); ip != nil {
		for _, denied := range sv.deniedIPs {
			if denied.Contains(ip) {
				return true
			}
		}
	}

	// Check domain denials
	domain := sv.extractDomain(target)
	for _, denied := range sv.deniedDomains {
		if domain == denied {
			return true
		}
	}

	return false
}

// normalizeTarget removes protocol and port for validation
func (sv *ScopeValidator) normalizeTarget(target string) string {
	// Remove protocol
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "ftp://")

	// Remove path
	if idx := strings.Index(target, "/"); idx != -1 {
		target = target[:idx]
	}

	return strings.TrimSpace(target)
}

// extractIP extracts IP from target (handles host:port)
func (sv *ScopeValidator) extractIP(target string) string {
	if !strings.Contains(target, ":") {
		return target
	}

	// Could be host:port or [ipv6]:port
	if strings.HasPrefix(target, "[") {
		// IPv6 format
		if idx := strings.Index(target, "]"); idx != -1 {
			return target[1:idx]
		}
	}

	// IPv4:port format
	if idx := strings.LastIndex(target, ":"); idx != -1 {
		host := target[:idx]
		// Verify it's an IP
		if net.ParseIP(host) != nil {
			return host
		}
	}

	return target
}

// extractDomain extracts domain from target
func (sv *ScopeValidator) extractDomain(target string) string {
	// Remove port if present
	if idx := strings.LastIndex(target, ":"); idx != -1 {
		// Make sure it's not IPv6
		if !strings.Contains(target[:idx], ":") {
			target = target[:idx]
		}
	}
	return strings.ToLower(target)
}

// Summary returns a summary of scope rules
func (sv *ScopeValidator) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Scope Configuration (strict=%v)\n", sv.strict))
	sb.WriteString(fmt.Sprintf("  Allowed domains: %v\n", sv.allowedDomains))
	sb.WriteString(fmt.Sprintf("  Allowed wildcards: %v\n", sv.allowedDomainWildcards))
	sb.WriteString(fmt.Sprintf("  Allowed CIDR ranges: %d\n", len(sv.allowedIPs)))
	sb.WriteString(fmt.Sprintf("  Denied domains: %v\n", sv.deniedDomains))
	sb.WriteString(fmt.Sprintf("  Denied CIDR ranges: %d\n", len(sv.deniedIPs)))
	return sb.String()
}
