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
	case "sslscan":
		findings = parseSslscanOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "nuclei":
		findings = parseNucleiOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "gobuster":
		findings = parseGobusterOutput(toolOutput.Stdout, scanID, toolOutput.Target)
	case "nikto":
		findings = parseNiktoOutput(toolOutput.Stdout, scanID, toolOutput.Target)
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

// parseSslscanOutput extracts SSL/TLS vulnerabilities from sslscan output
func parseSslscanOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Weak ciphers
		if strings.Contains(strings.ToLower(line), "accept") && 
		   (strings.Contains(strings.ToLower(line), "rc4") || 
		    strings.Contains(strings.ToLower(line), "md5") ||
		    strings.Contains(strings.ToLower(line), "sslv2") ||
		    strings.Contains(strings.ToLower(line), "sslv3")) {
			findings = append(findings, models.Finding{
				FindingType: models.FindingTypeWeakSSL,
				Severity:    models.SeverityHigh,
				Title:       "Weak SSL/TLS Cipher Detected",
				Description: fmt.Sprintf("Weak or insecure cipher suite enabled on %s", target),
				Evidence:    line,
				Remediation: "Disable weak ciphers (RC4, MD5, SSLv2, SSLv3). Use TLS 1.2+ with strong ciphers only.",
				Hostname:    target,
			})
		}
		
		// Expired or self-signed certificates
		if strings.Contains(strings.ToLower(line), "expired") || 
		   strings.Contains(strings.ToLower(line), "self-signed") {
			findings = append(findings, models.Finding{
				FindingType: models.FindingTypeWeakSSL,
				Severity:    models.SeverityMedium,
				Title:       "Certificate Issue Detected",
				Description: "SSL/TLS certificate problem found",
				Evidence:    line,
				Remediation: "Replace expired or self-signed certificates with valid CA-signed certificates",
				Hostname:    target,
			})
		}
	}
	
	return findings
}

// parseNucleiOutput extracts vulnerabilities from nuclei output
func parseNucleiOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") && !strings.Contains(line, "[critical]") && 
		   !strings.Contains(line, "[high]") && !strings.Contains(line, "[medium]") && 
		   !strings.Contains(line, "[low]") && !strings.Contains(line, "[info]") {
			continue
		}
		
		// Nuclei output format: [template-id] [severity] Title [url]
		// Example: [CVE-2021-12345] [high] Apache Log4j RCE [http://target.com]
		
		var severity models.Severity
		var title, templateID, url string
		
		// Extract severity
		if strings.Contains(strings.ToLower(line), "[critical]") {
			severity = models.SeverityCritical
		} else if strings.Contains(strings.ToLower(line), "[high]") {
			severity = models.SeverityHigh
		} else if strings.Contains(strings.ToLower(line), "[medium]") {
			severity = models.SeverityMedium
		} else if strings.Contains(strings.ToLower(line), "[low]") {
			severity = models.SeverityLow
		} else {
			severity = models.SeverityInfo
		}
		
		// Try to extract template ID (first bracketed value that's not severity)
		re := regexp.MustCompile(`\[([^\]]+)\]`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			val := match[1]
			if !strings.Contains(strings.ToLower(val), "critical") &&
			   !strings.Contains(strings.ToLower(val), "high") &&
			   !strings.Contains(strings.ToLower(val), "medium") &&
			   !strings.Contains(strings.ToLower(val), "low") &&
			   !strings.Contains(strings.ToLower(val), "info") {
				templateID = val
				break
			}
		}
		
		// Extract URL (last bracketed http/https value)
		urlRe := regexp.MustCompile(`\[(https?://[^\]]+)\]`)
		if urlMatch := urlRe.FindStringSubmatch(line); len(urlMatch) > 1 {
			url = urlMatch[1]
		}
		
		// Title is everything between template and URL
		title = line
		if templateID != "" {
			title = strings.Replace(title, "["+templateID+"]", "", 1)
		}
		title = re.ReplaceAllString(title, "")
		title = strings.TrimSpace(title)
		
		if title == "" {
			title = "Vulnerability Detected"
		}
		
		finding := models.Finding{
			FindingType: models.FindingTypeOther,
			Severity:    severity,
			Title:       title,
			Description: fmt.Sprintf("Nuclei template %s matched on %s", templateID, target),
			Evidence:    line,
			Remediation: "Review vulnerability details and apply vendor patches. Consult CVE database for specific remediation.",
			Hostname:    target,
		}
		
		findings = append(findings, finding)
	}
	
	return findings
}

// parseGobusterOutput extracts discovered directories and files from gobuster output
func parseGobusterOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "===============") || 
		   strings.HasPrefix(line, "Gobuster") || strings.Contains(line, "Progress:") {
			continue
		}
		
		// Gobuster format: /path (Status: 200) [Size: 1234]
		// Extract path and status code
		if strings.Contains(line, "(Status:") {
			parts := strings.Split(line, "(Status:")
			if len(parts) >= 2 {
				path := strings.TrimSpace(parts[0])
				statusPart := parts[1]
				
				// Extract status code
				statusRe := regexp.MustCompile(`(\d+)`)
				statusMatch := statusRe.FindStringSubmatch(statusPart)
				statusCode := ""
				if len(statusMatch) > 1 {
					statusCode = statusMatch[1]
				}
				
				severity := models.SeverityInfo
				remediation := "Review discovered paths for sensitive information exposure"
				
				// Sensitive paths
				lowerPath := strings.ToLower(path)
				if strings.Contains(lowerPath, "admin") || strings.Contains(lowerPath, "backup") ||
				   strings.Contains(lowerPath, ".git") || strings.Contains(lowerPath, ".env") ||
				   strings.Contains(lowerPath, "config") || strings.Contains(lowerPath, "sql") {
					severity = models.SeverityMedium
					remediation = "Sensitive path detected. Restrict access or remove if not needed."
				}
				
				finding := models.Finding{
					FindingType: models.FindingTypeOther,
					Severity:    severity,
					Title:       fmt.Sprintf("Directory/File Found: %s", path),
					Description: fmt.Sprintf("Discovered accessible path on %s (Status: %s)", target, statusCode),
					Evidence:    line,
					Remediation: remediation,
					Hostname:    target,
				}
				findings = append(findings, finding)
			}
		}
	}
	
	return findings
}

// parseNiktoOutput extracts web vulnerabilities from nikto output
func parseNiktoOutput(output string, scanID string, target string) []models.Finding {
	var findings []models.Finding
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "+") ||
		   strings.Contains(line, "Nikto v") || strings.Contains(line, "Target IP") ||
		   strings.Contains(line, "Target Hostname") || strings.Contains(line, "Start Time") {
			continue
		}
		
		// Nikto format: + Item description
		// Severity based on keywords
		var severity models.Severity
		lowerLine := strings.ToLower(line)
		
		if strings.Contains(lowerLine, "vulnerability") || strings.Contains(lowerLine, "injection") ||
		   strings.Contains(lowerLine, "xss") || strings.Contains(lowerLine, "sql") {
			severity = models.SeverityHigh
		} else if strings.Contains(lowerLine, "outdated") || strings.Contains(lowerLine, "version") ||
		          strings.Contains(lowerLine, "misconfiguration") {
			severity = models.SeverityMedium
		} else if strings.Contains(lowerLine, "header") || strings.Contains(lowerLine, "cookie") {
			severity = models.SeverityLow
		} else {
			severity = models.SeverityInfo
		}
		
		// Skip non-finding lines
		if !strings.HasPrefix(line, "+") && !strings.Contains(line, "OSVDB") && 
		   !strings.Contains(line, "CVE") {
			continue
		}
		
		title := strings.TrimPrefix(line, "+ ")
		if len(title) > 100 {
			title = title[:100] + "..."
		}
		
		finding := models.Finding{
			FindingType: models.FindingTypeOther,
			Severity:    severity,
			Title:       title,
			Description: fmt.Sprintf("Nikto web vulnerability scan finding on %s", target),
			Evidence:    line,
			Remediation: "Review Nikto output and apply recommended security configurations",
			Hostname:    target,
		}
		
		findings = append(findings, finding)
	}
	
	return findings
}
