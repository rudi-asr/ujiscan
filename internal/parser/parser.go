package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/google/uuid"
)

// Parser interface for parsing tool outputs
type Parser interface {
	Parse(output string, toolName string, scanID string) []models.Finding
}

// ParseToolOutput dispatches to appropriate parser based on tool name
func ParseToolOutput(toolOutput *models.ToolOutput, scanID string) []models.Finding {
	if !toolOutput.Success || toolOutput.Stdout == "" {
		return []models.Finding{}
	}

	var findings []models.Finding

	switch toolOutput.ToolName {
	case "nmap":
		findings = parseNmapOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "subfinder":
		findings = parseSubfinderOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "dig":
		findings = parseDigOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "httpx":
		findings = parseHttpxOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "whatweb":
		findings = parseWhatwebOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	}

	// Add tool name to all findings
	for i := range findings {
		findings[i].ToolName = toolOutput.ToolName
		findings[i].ScanID = scanID
		findings[i].Timestamp = time.Now()
		if findings[i].ID == "" {
			findings[i].ID = uuid.New().String()
		}
	}

	return findings
}

// parseNmapOutput extracts open ports and services from nmap output
func parseNmapOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	// Parse JSON nmap output (when using -oJ flag)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse each line as JSON
		var portData struct {
			Host  string `json:"host"`
			Ports []struct {
				Port     int    `json:"port"`
				Protocol string `json:"protocol"`
				State    string `json:"state"`
				Service  struct {
					Name    string `json:"name"`
					Product string `json:"product"`
					Version string `json:"version"`
				} `json:"service"`
			} `json:"ports"`
		}

		if err := json.Unmarshal([]byte(line), &portData); err == nil && len(portData.Ports) > 0 {
			for _, port := range portData.Ports {
				if port.State == "open" {
					severity := models.SeverityMedium
					if isCommonPort(port.Port) {
						severity = models.SeverityLow
					}

					finding := models.Finding{
						FindingType: models.FindingTypePort,
						Severity:    severity,
						Title:       fmt.Sprintf("Open Port: %d/%s", port.Port, port.Protocol),
						Description: fmt.Sprintf("Port %d is open on target %s", port.Port, target),
						Evidence:    fmt.Sprintf("Service: %s, Product: %s, Version: %s", port.Service.Name, port.Service.Product, port.Service.Version),
						Remediation: "Review firewall rules and ensure only necessary ports are open",
						Port:        port.Port,
						Protocol:    port.Protocol,
						Service:     port.Service.Name,
						Version:     port.Service.Version,
						IPAddress:   portData.Host,
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	// Fallback to regex parsing if JSON fails
	if len(findings) == 0 {
		findings = parseNmapPlainText(output, scanID, target)
	}

	return findings
}

// parseNmapPlainText parses plain text nmap output
func parseNmapPlainText(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	// Match: PORT STATE SERVICE
	portRegex := regexp.MustCompile(`(\d+)/(\w+)\s+(\w+)\s+(.*)`)
	
	for _, line := range strings.Split(output, "\n") {
		matches := portRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			port, _ := strconv.Atoi(matches[1])
			protocol := matches[2]
			state := matches[3]
			service := matches[4]

			if state == "open" {
				finding := models.Finding{
					FindingType: models.FindingTypePort,
					Severity:    models.SeverityMedium,
					Title:       fmt.Sprintf("Open Port: %d/%s", port, protocol),
					Description: fmt.Sprintf("Port %d is open with service: %s", port, service),
					Evidence:    line,
					Remediation: "Review and secure unnecessary open ports",
					Port:        port,
					Protocol:    protocol,
					Service:     service,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// parseSubfinderOutput extracts subdomains from subfinder output
func parseSubfinderOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	// Each line is either a JSON object or domain
	lines := strings.Split(output, "\n")
	uniqueDomains := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try JSON first
		var data struct {
			Host string `json:"host"`
		}
		if err := json.Unmarshal([]byte(line), &data); err == nil && data.Host != "" {
			if !uniqueDomains[data.Host] {
				uniqueDomains[data.Host] = true
				findings = append(findings, models.Finding{
					FindingType: models.FindingTypeSubdomain,
					Severity:    models.SeverityInfo,
					Title:       fmt.Sprintf("Subdomain Found: %s", data.Host),
					Description: fmt.Sprintf("Discovered subdomain of %s", target),
					Evidence:    data.Host,
					Remediation: "Review subdomain purpose and security configuration",
					Hostname:    data.Host,
				})
			}
		} else if strings.Contains(line, ".") && !strings.Contains(line, "\"") {
			// Plain domain
			if !uniqueDomains[line] {
				uniqueDomains[line] = true
				findings = append(findings, models.Finding{
					FindingType: models.FindingTypeSubdomain,
					Severity:    models.SeverityInfo,
					Title:       fmt.Sprintf("Subdomain Found: %s", line),
					Description: fmt.Sprintf("Discovered subdomain of %s", target),
					Evidence:    line,
					Remediation: "Review subdomain purpose and security configuration",
					Hostname:    line,
				})
			}
		}
	}

	return findings
}

// parseDigOutput extracts DNS records from dig output
func parseDigOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	// Parse ANSWER SECTION
	lines := strings.Split(output, "\n")
	inAnswerSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "ANSWER SECTION") {
			inAnswerSection = true
			continue
		}
		if strings.HasPrefix(line, ";") || line == "" {
			continue
		}
		if inAnswerSection && strings.Contains(line, "SECTION") {
			break
		}

		if inAnswerSection {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				domain := parts[0]
				recordType := parts[3]
				recordValue := strings.Join(parts[4:], " ")

				finding := models.Finding{
					FindingType: models.FindingTypeDNSRecord,
					Severity:    models.SeverityInfo,
					Title:       fmt.Sprintf("DNS Record: %s %s", domain, recordType),
					Description: fmt.Sprintf("%s record for %s", recordType, domain),
					Evidence:    fmt.Sprintf("%s -> %s", recordType, recordValue),
					Remediation: "Review DNS configuration for security best practices",
					Hostname:    domain,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// parseHttpxOutput extracts HTTP details from httpx output
func parseHttpxOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try JSON parsing
		var data struct {
			URL        string `json:"url"`
			StatusCode int    `json:"status_code"`
			Server     string `json:"server"`
			Title      string `json:"title"`
			Webserver  string `json:"webserver"`
		}

		if err := json.Unmarshal([]byte(line), &data); err == nil && data.StatusCode > 0 {
			severity := models.SeverityInfo
			if data.StatusCode == 401 || data.StatusCode == 403 {
				severity = models.SeverityMedium
			}

			finding := models.Finding{
				FindingType: models.FindingTypeHTTPHeader,
				Severity:    severity,
				Title:       fmt.Sprintf("HTTP Service: %s (Status: %d)", data.URL, data.StatusCode),
				Description: fmt.Sprintf("HTTP service running on %s", data.URL),
				Evidence:    fmt.Sprintf("Server: %s, Title: %s", data.Server, data.Title),
				Remediation: "Review HTTP security headers and service configuration",
				Hostname:    data.URL,
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

// parseWhatwebOutput extracts technology stack from whatweb output
func parseWhatwebOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WhatWeb") {
			continue
		}

		// Parse whatweb format: URL [HTTP Status] [Details]
		if strings.Contains(line, "[") {
			parts := strings.Split(line, "[")
			if len(parts) > 1 {
				details := strings.Join(parts[1:], "[")
				
				finding := models.Finding{
					FindingType: models.FindingTypeTechStack,
					Severity:    models.SeverityInfo,
					Title:       fmt.Sprintf("Technology Detected: %s", details),
					Description: fmt.Sprintf("Technology stack detected on %s", target),
					Evidence:    line,
					Remediation: "Review exposed technology versions for known vulnerabilities",
					Hostname:    target,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// isCommonPort returns true if port is a common, expected open port
func isCommonPort(port int) bool {
	commonPorts := []int{22, 80, 443, 3306, 5432, 6379, 8080, 8443, 5000, 3000}
	for _, p := range commonPorts {
		if port == p {
			return true
		}
	}
	return false
}
