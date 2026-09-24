// Package findings provides finding extraction and verification
package findings

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Finding represents a security discovery
type Finding struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"` // vulnerability, info, misconfiguration, etc
	Severity    string      `json:"severity"` // info, low, medium, high, critical
	CVSS        float32     `json:"cvss,omitempty"`
	CWE         string      `json:"cwe,omitempty"`
	CVE         string      `json:"cve,omitempty"`
	Target      string      `json:"target"`
	Source      string      `json:"source"` // which tool found this
	Evidence    string      `json:"evidence"` // proof
	SourceData  interface{} `json:"source_data,omitempty"`
	Verified    bool        `json:"verified"`
	VerifyTools []string    `json:"verify_tools,omitempty"` // tools to verify
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FindingParser extracts findings from tool output
type FindingParser interface {
	Parse(output string, target string) []*Finding
	ToolName() string
}

// NmapParser extracts findings from nmap output
type NmapParser struct{}

func (p *NmapParser) ToolName() string {
	return "nmap"
}

func (p *NmapParser) Parse(output string, target string) []*Finding {
	findings := make([]*Finding, 0)

	// Parse open ports from nmap text output
	// Pattern: port/protocol open service
	portPattern := regexp.MustCompile(`(\d+)/(tcp|udp)\s+open\s+(\S+)`)
	matches := portPattern.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		port := match[1]
		protocol := match[2]
		service := match[3]

		finding := &Finding{
			ID:          fmt.Sprintf("nmap-port-%s-%s", port, protocol),
			Title:       fmt.Sprintf("Open Port: %s/%s (%s)", port, protocol, service),
			Description: fmt.Sprintf("Port %s/%s is open and running %s service", port, protocol, service),
			Type:        "open_port",
			Severity:    "info",
			Target:      target,
			Source:      "nmap",
			Evidence:    fmt.Sprintf("Port %s/%s open - service: %s", port, protocol, service),
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata: map[string]interface{}{
				"port":     port,
				"protocol": protocol,
				"service":  service,
			},
		}

		findings = append(findings, finding)
	}

	return findings
}

// NucleiParser extracts findings from nuclei JSONL output
type NucleiParser struct{}

func (p *NucleiParser) ToolName() string {
	return "nuclei"
}

func (p *NucleiParser) Parse(output string, target string) []*Finding {
	findings := make([]*Finding, 0)

	// Parse JSONL format - each line is a JSON finding
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract severity from nuclei output format
		// Nuclei outputs like: [severity] [template-id] message
		severityPattern := regexp.MustCompile(`\[([a-z]+)\]`)
		severityMatch := severityPattern.FindStringSubmatch(line)
		severity := "info"
		if len(severityMatch) > 1 {
			severity = severityMatch[1]
		}

		// Extract message
		msgPattern := regexp.MustCompile(`\]\s+(.+)$`)
		msgMatch := msgPattern.FindStringSubmatch(line)
		message := line
		if len(msgMatch) > 1 {
			message = msgMatch[1]
		}

		finding := &Finding{
			ID:          fmt.Sprintf("nuclei-%d", len(findings)),
			Title:       message,
			Description: message,
			Type:        "vulnerability",
			Severity:    severity,
			Target:      target,
			Source:      "nuclei",
			Evidence:    line,
			Verified:    false, // nuclei findings need verification
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		findings = append(findings, finding)
	}

	return findings
}

// CurlParser extracts findings from curl output (headers, redirects, etc)
type CurlParser struct{}

func (p *CurlParser) ToolName() string {
	return "curl"
}

func (p *CurlParser) Parse(output string, target string) []*Finding {
	findings := make([]*Finding, 0)

	// Check for security headers
	securityHeaders := map[string]string{
		"Strict-Transport-Security": "HSTS header missing",
		"X-Content-Type-Options":    "X-Content-Type-Options missing",
		"X-Frame-Options":           "X-Frame-Options missing",
		"Content-Security-Policy":   "CSP header missing",
	}

	for header, finding := range securityHeaders {
		if !strings.Contains(output, header) {
			f := &Finding{
				ID:          fmt.Sprintf("curl-header-%s", strings.ToLower(header)),
				Title:       finding,
				Description: fmt.Sprintf("Missing HTTP security header: %s", header),
				Type:        "misconfiguration",
				Severity:    "low",
				Target:      target,
				Source:      "curl",
				Evidence:    fmt.Sprintf("Header '%s' not found in response", header),
				Verified:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			findings = append(findings, f)
		}
	}

	// Check for insecure protocols
	if strings.Contains(output, "HTTP/1.0") || strings.Contains(output, "http://") {
		finding := &Finding{
			ID:          "curl-insecure-protocol",
			Title:       "Insecure Protocol",
			Description: "Target uses unencrypted HTTP or older protocol version",
			Type:        "misconfiguration",
			Severity:    "medium",
			Target:      target,
			Source:      "curl",
			Evidence:    "HTTP/1.0 or unencrypted protocol detected",
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		findings = append(findings, finding)
	}

	return findings
}

// WhoisParser extracts findings from whois output
type WhoisParser struct{}

func (p *WhoisParser) ToolName() string {
	return "whois"
}

func (p *WhoisParser) Parse(output string, target string) []*Finding {
	findings := make([]*Finding, 0)

	// Check for domain registration info (informational)
	if strings.Contains(output, "Registrar:") || strings.Contains(output, "registrar") {
		finding := &Finding{
			ID:          "whois-registered",
			Title:       "Domain Registered",
			Description: "Domain is registered and active",
			Type:        "info",
			Severity:    "info",
			Target:      target,
			Source:      "whois",
			Evidence:    "WHOIS record found",
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		findings = append(findings, finding)
	}

	// Check for expiration info
	if strings.Contains(output, "Expir") {
		finding := &Finding{
			ID:          "whois-expiration",
			Title:       "Domain Expiration Info Available",
			Description: "Domain expiration information found in WHOIS",
			Type:        "info",
			Severity:    "info",
			Target:      target,
			Source:      "whois",
			Evidence:    "Expiration information in WHOIS record",
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		findings = append(findings, finding)
	}

	return findings
}

// DigParser extracts findings from dig DNS output
type DigParser struct{}

func (p *DigParser) ToolName() string {
	return "dig"
}

func (p *DigParser) Parse(output string, target string) []*Finding {
	findings := make([]*Finding, 0)

	// Check if DNS resolution succeeded
	if strings.Contains(output, "ANSWER SECTION") || strings.Contains(output, ";") {
		finding := &Finding{
			ID:          "dig-resolution",
			Title:       "DNS Resolution Successful",
			Description: "Domain resolves via DNS",
			Type:        "info",
			Severity:    "info",
			Target:      target,
			Source:      "dig",
			Evidence:    "DNS A/AAAA records found",
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		findings = append(findings, finding)
	}

	// Check for DNS AXFR (zone transfer) possibility
	if strings.Contains(output, "Transfer failed") {
		finding := &Finding{
			ID:          "dig-axfr-protected",
			Title:       "DNS Zone Transfer Protected",
			Description: "DNS AXFR is properly blocked",
			Type:        "info",
			Severity:    "info",
			Target:      target,
			Source:      "dig",
			Evidence:    "Zone transfer denied",
			Verified:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		findings = append(findings, finding)
	}

	return findings
}

// ParserRegistry holds all parsers
type ParserRegistry struct {
	parsers map[string]FindingParser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[string]FindingParser),
	}
}

// RegisterParser adds a parser
func (pr *ParserRegistry) RegisterParser(parser FindingParser) {
	pr.parsers[parser.ToolName()] = parser
}

// Parse extracts findings from tool output
func (pr *ParserRegistry) Parse(toolName string, output string, target string) []*Finding {
	parser, ok := pr.parsers[toolName]
	if !ok {
		return make([]*Finding, 0)
	}
	return parser.Parse(output, target)
}

// InitializeDefaultParsers adds built-in parsers
func (pr *ParserRegistry) InitializeDefaultParsers() {
	pr.RegisterParser(&NmapParser{})
	pr.RegisterParser(&NucleiParser{})
	pr.RegisterParser(&CurlParser{})
	pr.RegisterParser(&WhoisParser{})
	pr.RegisterParser(&DigParser{})
}

// Verifier checks if findings are real
type Verifier struct {
	registry ParserRegistry
}

// NewVerifier creates a new finding verifier
func NewVerifier() *Verifier {
	v := &Verifier{
		registry: *NewParserRegistry(),
	}
	v.registry.InitializeDefaultParsers()
	return v
}

// VerifySeverity maps finding characteristics to severity
func (v *Verifier) VerifySeverity(finding *Finding) {
	// CVSS-based severity
	if finding.CVSS > 0 {
		switch {
		case finding.CVSS == 0:
			finding.Severity = "info"
		case finding.CVSS < 4:
			finding.Severity = "low"
		case finding.CVSS < 7:
			finding.Severity = "medium"
		case finding.CVSS < 9:
			finding.Severity = "high"
		default:
			finding.Severity = "critical"
		}
		return
	}

	// Type-based severity defaults
	switch finding.Type {
	case "open_port":
		finding.Severity = "info"
	case "misconfiguration":
		finding.Severity = "low"
	case "vulnerability":
		finding.Severity = "high"
	case "critical_vuln":
		finding.Severity = "critical"
	default:
		finding.Severity = "medium"
	}
}

// FilterDuplicates removes duplicate findings
func (v *Verifier) FilterDuplicates(findings []*Finding) []*Finding {
	seen := make(map[string]bool)
	unique := make([]*Finding, 0)

	for _, finding := range findings {
		key := fmt.Sprintf("%s:%s:%s", finding.Title, finding.Target, finding.Source)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, finding)
		}
	}

	return unique
}

// AggregateFindings combines findings from multiple sources
func (v *Verifier) AggregateFindings(findings []*Finding) []*Finding {
	// Filter duplicates
	unique := v.FilterDuplicates(findings)

	// Verify severity for each
	for _, finding := range unique {
		v.VerifySeverity(finding)
	}

	return unique
}
