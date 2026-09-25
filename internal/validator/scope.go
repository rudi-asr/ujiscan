package validator

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ScopeValidator validates and classifies targets
type ScopeValidator struct {
	classifier *Classifier
}

// NewScopeValidator creates a new scope validator
func NewScopeValidator() *ScopeValidator {
	return &ScopeValidator{
		classifier: NewClassifier(),
	}
}

// Validate validates a target and returns a TargetProfile
func (sv *ScopeValidator) Validate(target, scopeInclude, scopeExclude string) (*TargetProfile, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}

	profile := &TargetProfile{
		ID:         uuid.New().String(),
		Target:     target,
		CreatedAt:  time.Now(),
		Properties: make(map[string]interface{}),
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Step 1: Classify target type
	classification := sv.classifier.Classify(target)
	profile.Type = classification.Type
	profile.Properties["classification_confidence"] = classification.Confidence
	profile.Properties["classification_reason"] = classification.Reason

	if classification.Type == TargetTypeUnknown {
		profile.IsValid = false
		profile.Reason = "Unable to classify target type"
		profile.Errors = append(profile.Errors, "Unknown target type")
		return profile, nil
	}

	// Step 2: Parse scope
	scope := &ScopeDefinition{
		Include: parseScope(scopeInclude),
		Exclude: parseScope(scopeExclude),
	}

	// Parse include patterns
	for _, inc := range scope.Include {
		if _, ipnet, err := net.ParseCIDR(inc); err == nil {
			scope.IncludedNets = append(scope.IncludedNets, ipnet)
		} else {
			scope.IncludedDomains = append(scope.IncludedDomains, inc)
		}
	}

	// Parse exclude patterns
	for _, exc := range scope.Exclude {
		if _, ipnet, err := net.ParseCIDR(exc); err == nil {
			scope.ExcludedNets = append(scope.ExcludedNets, ipnet)
		} else {
			scope.ExcludedDomains = append(scope.ExcludedDomains, exc)
		}
	}

	profile.Scope = scope

	// Step 3: Extract hosts from target
	hosts := sv.extractHosts(target)
	profile.Hosts = hosts

	// Step 4: Validate against policy
	if errs := sv.validatePolicy(profile); len(errs) > 0 {
		profile.Errors = append(profile.Errors, errs...)
		profile.IsValid = false
		if len(errs) > 0 {
			profile.Reason = errs[0]
		}
		return profile, nil
	}

	// Step 5: Validate against scope (if provided)
	if len(scope.Include) > 0 {
		if err := sv.validateScope(profile); err != nil {
			profile.Errors = append(profile.Errors, err.Error())
			profile.IsValid = false
			profile.Reason = err.Error()
			return profile, nil
		}
	}

	// All validations passed
	profile.IsValid = true
	return profile, nil
}

// validatePolicy checks target against security policy
func (sv *ScopeValidator) validatePolicy(profile *TargetProfile) []string {
	var errors []string

	for _, host := range profile.Hosts {
		// Check for localhost
		if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
			errors = append(errors, "Target is localhost (not allowed for engagement)")
		}

		// Check for private IPs
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsPrivate() {
				errors = append(errors, fmt.Sprintf("Target %s is private IP (not allowed)", host))
			}
		}

		// Check for other restricted patterns
		if strings.Contains(host, "192.168.") || strings.Contains(host, "10.0.") {
			errors = append(errors, fmt.Sprintf("Target %s is private IP range (not allowed)", host))
		}
	}

	return errors
}

// validateScope checks if hosts are within defined scope
func (sv *ScopeValidator) validateScope(profile *TargetProfile) error {
	if profile.Scope == nil || len(profile.Scope.Include) == 0 {
		return nil // No scope restrictions
	}

	for _, host := range profile.Hosts {
		if !sv.isInScope(host, profile.Scope) {
			return fmt.Errorf("target %s is not within defined scope", host)
		}
	}

	return nil
}

// isInScope checks if a host matches any include pattern
func (sv *ScopeValidator) isInScope(host string, scope *ScopeDefinition) bool {
	// Check against IPs
	if ip := net.ParseIP(host); ip != nil {
		for _, ipnet := range scope.IncludedNets {
			if ipnet.Contains(ip) {
				return true
			}
		}
	}

	// Check against domains
	for _, domain := range scope.IncludedDomains {
		if sv.matchDomain(host, domain) {
			return true
		}
	}

	return false
}

// matchDomain checks if host matches a domain pattern (including wildcards)
func (sv *ScopeValidator) matchDomain(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard match (*.example.com matches sub.example.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(host, "."+suffix) || host == suffix
	}

	return false
}

// extractHosts extracts hosts from target (could be URL, IP, domain, etc)
func (sv *ScopeValidator) extractHosts(target string) []string {
	target = strings.TrimSpace(target)
	var hosts []string

	// If it's a URL, extract hostname
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		if u, err := url.Parse(target); err == nil {
			if u.Hostname() != "" {
				hosts = append(hosts, u.Hostname())
			}
		}
	} else if strings.Contains(target, "/") {
		// CIDR notation - just store as-is
		hosts = append(hosts, strings.Split(target, "/")[0])
	} else {
		// Bare domain or IP
		hosts = append(hosts, target)
	}

	return hosts
}

// parseScope parses scope string (comma-separated)
func parseScope(scope string) []string {
	if scope == "" {
		return []string{}
	}

	var patterns []string
	for _, s := range strings.Split(scope, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			patterns = append(patterns, s)
		}
	}
	return patterns
}
