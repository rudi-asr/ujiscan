// Package findings provides finding classification
package findings

import (
	"fmt"
	"strings"
)

// Classifier categorizes findings based on characteristics
type Classifier struct {
	rules []*ClassificationRule
}

// ClassificationRule defines how to classify a finding
type ClassificationRule struct {
	ID          string
	Name        string
	Patterns    []string // patterns to match in title/description
	FindingType string   // the type to assign
	Severity    string   // default severity
	CWE         string   // associated CWE
	CVE         string   // associated CVE pattern
}

// NewClassifier creates a new finding classifier
func NewClassifier() *Classifier {
	c := &Classifier{
		rules: make([]*ClassificationRule, 0),
	}
	c.loadDefaultRules()
	return c
}

// loadDefaultRules loads built-in classification rules
func (c *Classifier) loadDefaultRules() {
	rules := []*ClassificationRule{
		// SQL Injection
		{
			ID:          "sql_injection",
			Name:        "SQL Injection",
			Patterns:    []string{"sql injection", "sqli", "sql error", "sql syntax"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-89",
		},
		// XSS
		{
			ID:          "xss",
			Name:        "Cross-Site Scripting",
			Patterns:    []string{"xss", "cross-site script", "script injection", "dom xss"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-79",
		},
		// Authentication
		{
			ID:          "weak_auth",
			Name:        "Weak Authentication",
			Patterns:    []string{"weak password", "default cred", "basic auth", "no auth"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-521",
		},
		// CSRF
		{
			ID:          "csrf",
			Name:        "Cross-Site Request Forgery",
			Patterns:    []string{"csrf", "cross-site request"},
			FindingType: "vulnerability",
			Severity:    "medium",
			CWE:         "CWE-352",
		},
		// XXE
		{
			ID:          "xxe",
			Name:        "XML External Entity",
			Patterns:    []string{"xxe", "xml external", "entity injection"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-611",
		},
		// SSRF
		{
			ID:          "ssrf",
			Name:        "Server-Side Request Forgery",
			Patterns:    []string{"ssrf", "server-side request"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-918",
		},
		// Path Traversal
		{
			ID:          "path_traversal",
			Name:        "Path Traversal",
			Patterns:    []string{"path traversal", "directory traversal", "../"},
			FindingType: "vulnerability",
			Severity:    "medium",
			CWE:         "CWE-22",
		},
		// Insecure Deserialization
		{
			ID:          "deserialization",
			Name:        "Insecure Deserialization",
			Patterns:    []string{"deserialization", "object injection", "pickle"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-502",
		},
		// Information Disclosure
		{
			ID:          "info_disclosure",
			Name:        "Information Disclosure",
			Patterns:    []string{"information disclosure", "exposed", "leaked", "sensitive data"},
			FindingType: "information_disclosure",
			Severity:    "medium",
			CWE:         "CWE-200",
		},
		// Misconfiguration
		{
			ID:          "misconfig",
			Name:        "Security Misconfiguration",
			Patterns:    []string{"misconfiguration", "misconfigured", "insecure config", "default config"},
			FindingType: "misconfiguration",
			Severity:    "medium",
			CWE:         "CWE-16",
		},
		// Broken Access Control
		{
			ID:          "authz",
			Name:        "Broken Access Control",
			Patterns:    []string{"access control", "privilege escalation", "unauthorized access", "privilege"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-284",
		},
		// Insecure Protocol
		{
			ID:          "insecure_proto",
			Name:        "Insecure Protocol",
			Patterns:    []string{"http/1.0", "unencrypted", "plaintext", "no encryption"},
			FindingType: "misconfiguration",
			Severity:    "medium",
			CWE:         "CWE-319",
		},
		// Missing Security Headers
		{
			ID:          "missing_headers",
			Name:        "Missing Security Headers",
			Patterns:    []string{"missing header", "hsts", "csp", "x-content-type"},
			FindingType: "misconfiguration",
			Severity:    "low",
			CWE:         "CWE-693",
		},
		// RCE
		{
			ID:          "rce",
			Name:        "Remote Code Execution",
			Patterns:    []string{"rce", "code execution", "command execution", "remote execution"},
			FindingType: "vulnerability",
			Severity:    "critical",
			CWE:         "CWE-94",
		},
		// File Upload
		{
			ID:          "file_upload",
			Name:        "Unrestricted File Upload",
			Patterns:    []string{"file upload", "unrestricted upload", "arbitrary file"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-434",
		},
		// Broken Crypto
		{
			ID:          "crypto",
			Name:        "Broken Cryptography",
			Patterns:    []string{"weak crypto", "broken encryption", "md5", "sha1", "weak hash"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-327",
		},
		// Open Port (info)
		{
			ID:          "open_port",
			Name:        "Open Port",
			Patterns:    []string{"open port", "port open", "listening"},
			FindingType: "open_port",
			Severity:    "info",
		},
		// Default Credentials
		{
			ID:          "default_creds",
			Name:        "Default Credentials",
			Patterns:    []string{"default", "admin", "password", "123456"},
			FindingType: "vulnerability",
			Severity:    "high",
			CWE:         "CWE-798",
		},
	}

	for _, rule := range rules {
		c.rules = append(c.rules, rule)
	}
}

// Classify analyzes a finding and assigns type/severity
func (c *Classifier) Classify(finding *Finding) {
	titleLower := strings.ToLower(finding.Title)
	descLower := strings.ToLower(finding.Description)
	searchText := titleLower + " " + descLower

	// Check each rule in order
	for _, rule := range c.rules {
		matched := false
		for _, pattern := range rule.Patterns {
			if strings.Contains(searchText, strings.ToLower(pattern)) {
				matched = true
				break
			}
		}

		if matched {
			// Apply classification
			finding.Type = rule.FindingType
			if rule.Severity != "" {
				finding.Severity = rule.Severity
			}
			if rule.CWE != "" {
				finding.CWE = rule.CWE
			}
			return // First match wins
		}
	}

	// Default classification if no rule matched
	if finding.Type == "" {
		finding.Type = "unknown"
	}
	if finding.Severity == "" {
		finding.Severity = "medium"
	}
}

// ClassifyBatch classifies multiple findings
func (c *Classifier) ClassifyBatch(findings []*Finding) {
	for _, finding := range findings {
		c.Classify(finding)
	}
}

// SeverityCounts returns count of findings by severity
type SeverityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Info     int
	Total    int
}

// CountBySeverity counts findings by severity level
func CountBySeverity(findings []*Finding) SeverityCounts {
	counts := SeverityCounts{}

	for _, finding := range findings {
		counts.Total++
		switch finding.Severity {
		case "critical":
			counts.Critical++
		case "high":
			counts.High++
		case "medium":
			counts.Medium++
		case "low":
			counts.Low++
		case "info":
			counts.Info++
		}
	}

	return counts
}

// String returns formatted severity counts
func (sc SeverityCounts) String() string {
	return fmt.Sprintf(
		"Findings: %d total | 🔴 Critical: %d | 🟠 High: %d | 🟡 Medium: %d | 🟢 Low: %d | ℹ️ Info: %d",
		sc.Total, sc.Critical, sc.High, sc.Medium, sc.Low, sc.Info,
	)
}

// GroupBySeverity groups findings by severity
func GroupBySeverity(findings []*Finding) map[string][]*Finding {
	grouped := make(map[string][]*Finding)
	severities := []string{"critical", "high", "medium", "low", "info"}

	for _, severity := range severities {
		grouped[severity] = make([]*Finding, 0)
	}

	for _, finding := range findings {
		if findings, ok := grouped[finding.Severity]; ok {
			grouped[finding.Severity] = append(findings, finding)
		}
	}

	return grouped
}

// FilterBySeverity returns findings matching severity level
func FilterBySeverity(findings []*Finding, severity string) []*Finding {
	filtered := make([]*Finding, 0)
	for _, f := range findings {
		if f.Severity == severity {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// FilterByType returns findings matching type
func FilterByType(findings []*Finding, findingType string) []*Finding {
	filtered := make([]*Finding, 0)
	for _, f := range findings {
		if f.Type == findingType {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
