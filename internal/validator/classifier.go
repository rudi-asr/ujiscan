package validator

import (
	"net/url"
	"strings"
)

// Classifier detects target type based on input
type Classifier struct{}

// NewClassifier creates a new target classifier
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify determines the type of target
func (c *Classifier) Classify(target string) *TargetClassification {
	target = strings.TrimSpace(target)

	// Try parsing as URL
	if u, err := url.Parse(target); err == nil && u.Scheme != "" {
		return c.classifyURL(u)
	}

	// Try parsing as IP/CIDR
	if classification := c.classifyIP(target); classification != nil {
		return classification
	}

	// Try parsing as domain
	if classification := c.classifyDomain(target); classification != nil {
		return classification
	}

	// Unknown
	return &TargetClassification{
		Type:       TargetTypeUnknown,
		Confidence: 0.0,
		Reason:     "Could not classify target type",
	}
}

// classifyURL detects URL-based targets (WEB applications)
func (c *Classifier) classifyURL(u *url.URL) *TargetClassification {
	scheme := strings.ToLower(u.Scheme)

	// HTTP/HTTPS → Web application
	if scheme == "http" || scheme == "https" {
		indicators := []string{"http-based protocol"}
		
		// Check for common web ports
		host := u.Host
		if strings.Contains(host, ":") {
			parts := strings.Split(host, ":")
			port := parts[1]
			if port == "80" || port == "443" || port == "8080" || port == "8443" || port == "3000" {
				indicators = append(indicators, "common web port detected")
			}
		}

		return &TargetClassification{
			Type:       TargetTypeWeb,
			Confidence: 0.95,
			Reason:     "HTTP(S) URL detected",
			Indicators: indicators,
		}
	}

	return &TargetClassification{
		Type:       TargetTypeUnknown,
		Confidence: 0.1,
		Reason:     "Unknown URL scheme: " + scheme,
	}
}

// classifyIP detects IP-based targets (NETWORK infrastructure)
func (c *Classifier) classifyIP(target string) *TargetClassification {
	// Try parsing single IP
	if strings.Count(target, ".") == 3 && !strings.Contains(target, "/") {
		parts := strings.Split(target, ".")
		if len(parts) == 4 {
			return &TargetClassification{
				Type:       TargetTypeNetwork,
				Confidence: 0.9,
				Reason:     "Single IP address detected",
				Indicators: []string{"IPv4 format"},
			}
		}
	}

	// Try parsing CIDR
	if strings.Contains(target, "/") {
		parts := strings.Split(target, "/")
		if len(parts) == 2 {
			return &TargetClassification{
				Type:       TargetTypeNetwork,
				Confidence: 0.95,
				Reason:     "CIDR range detected",
				Indicators: []string{"CIDR notation", parts[1]},
			}
		}
	}

	return nil
}

// classifyDomain detects domain-based targets
func (c *Classifier) classifyDomain(target string) *TargetClassification {
	// Basic domain detection: contains dot, no spaces, valid chars
	if !strings.Contains(target, " ") && strings.Contains(target, ".") {
		// Check for wildcard (cloud/multi-tenant)
		if strings.HasPrefix(target, "*.") {
			return &TargetClassification{
				Type:       TargetTypeCloud,
				Confidence: 0.85,
				Reason:     "Wildcard domain detected (cloud infrastructure)",
				Indicators: []string{"wildcard pattern", "multi-tenant"},
			}
		}

		// Regular domain
		return &TargetClassification{
			Type:       TargetTypeWeb,
			Confidence: 0.8,
			Reason:     "Domain name detected",
			Indicators: []string{"domain format"},
		}
	}

	return nil
}
