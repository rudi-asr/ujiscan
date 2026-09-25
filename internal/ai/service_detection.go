package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ServiceInfo represents a detected service on a port
type ServiceInfo struct {
	Port     int
	Protocol string // tcp/udp
	Service  string // ssh, http, https, mysql, postgresql, etc
	Version  string // optional version info
	State    string // open, closed, filtered
}

// DetectServicesFromNmap parses nmap output to identify services
func (c *Client) DetectServicesFromNmap(nmapOutput string) []ServiceInfo {
	var services []ServiceInfo

	// Parse lines like: "22/tcp   open  ssh      OpenSSH 7.4"
	lines := strings.Split(nmapOutput, "\n")
	portRegex := regexp.MustCompile(`^(\d+)/(tcp|udp)\s+(\w+)\s+(\S+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := portRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			port, _ := strconv.Atoi(matches[1])
			protocol := matches[2]
			state := matches[3]
			serviceName := matches[4]

			// Extract version if present
			version := ""
			parts := strings.Fields(line)
			if len(parts) > 4 {
				version = strings.Join(parts[4:], " ")
			}

			service := ServiceInfo{
				Port:     port,
				Protocol: protocol,
				Service:  c.normalizeServiceName(serviceName),
				Version:  version,
				State:    state,
			}

			services = append(services, service)
			fmt.Printf("[ai] Detected: port %d/%s - %s (%s)\n", port, protocol, service.Service, state)
		}
	}

	return services
}

// normalizeServiceName standardizes service names
func (c *Client) normalizeServiceName(name string) string {
	name = strings.ToLower(name)

	// Common aliases
	switch name {
	case "http", "www":
		return "http"
	case "https", "ssl":
		return "https"
	case "ssh", "openssh":
		return "ssh"
	case "ftp", "vsftpd":
		return "ftp"
	case "smtp", "sendmail":
		return "smtp"
	case "pop3", "pop":
		return "pop3"
	case "imap":
		return "imap"
	case "mysql", "mariadb":
		return "mysql"
	case "postgresql", "postgres", "psql":
		return "postgresql"
	case "mongodb", "mongod":
		return "mongodb"
	case "redis":
		return "redis"
	case "rdp", "ms-wbt-server":
		return "rdp"
	case "smb", "netbios-ssn":
		return "smb"
	case "dns", "domain":
		return "dns"
	case "ldap":
		return "ldap"
	case "snmp":
		return "snmp"
	case "ntp":
		return "ntp"
	case "vnc":
		return "vnc"
	case "telnet":
		return "telnet"
	default:
		return name
	}
}

// GetToolsForServices recommends tools based on detected services
func (c *Client) GetToolsForServices(services []ServiceInfo) []string {
	toolSet := make(map[string]bool) // Use map to avoid duplicates

	for _, svc := range services {
		tools := c.getToolsForService(svc)
		for _, tool := range tools {
			toolSet[tool] = true
		}
	}

	// Convert map to slice
	var result []string
	// Prioritize by frequency
	priority := []string{
		"httpx", "whatweb", "sslscan", "nikto", "nuclei",
		"mysql", "postgresql", "mongodb",
		"smb", "ssh", "ldap", "snmp",
	}

	for _, tool := range priority {
		if toolSet[tool] {
			result = append(result, tool)
			delete(toolSet, tool)
		}
	}

	// Add remaining tools
	for tool := range toolSet {
		if tool != "" {
			result = append(result, tool)
		}
	}

	return result
}

// getToolsForService returns tools for a specific service
func (c *Client) getToolsForService(svc ServiceInfo) []string {
	switch svc.Service {
	case "http":
		return []string{"httpx", "whatweb", "nikto", "nuclei"}
	case "https":
		return []string{"httpx", "whatweb", "sslscan", "nikto", "nuclei"}
	case "ssh":
		return []string{"sslscan"} // SSH fingerprinting
	case "ftp":
		return []string{"nmap"} // Already have from nmap
	case "smtp", "pop3", "imap":
		return []string{} // Mail services - specialized tools needed
	case "mysql", "postgresql":
		return []string{"nuclei"} // Database scanning via nuclei templates
	case "mongodb", "redis":
		return []string{"nmap"} // Already have
	case "rdp":
		return []string{} // RDP scanning specialized
	case "smb":
		return []string{"nmap"} // SMB enumeration already in nmap
	case "dns":
		return []string{"nuclei"}
	case "ldap":
		return []string{"nuclei"}
	case "snmp":
		return []string{"nmap"}
	case "vnc":
		return []string{"nmap"}
	case "telnet":
		return []string{"nmap"}
	default:
		// Generic HTTP-like services
		if strings.Contains(svc.Service, "web") {
			return []string{"httpx", "whatweb"}
		}
		return []string{}
	}
}

// DetectOS attempts to identify operating system from tool output
func (c *Client) DetectOS(toolOutput map[string]string) string {
	nmapOutput, ok := toolOutput["nmap"]
	if !ok {
		return "Unknown"
	}

	// nmap typically shows "OS details: Linux 4.x.x" or similar
	osRegex := regexp.MustCompile(`OS details?: ([^\n]+)`)
	matches := osRegex.FindStringSubmatch(nmapOutput)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback patterns
	if strings.Contains(nmapOutput, "Linux") {
		return "Linux"
	}
	if strings.Contains(nmapOutput, "Windows") {
		return "Windows"
	}
	if strings.Contains(nmapOutput, "BSD") {
		return "BSD"
	}
	if strings.Contains(nmapOutput, "Mac OS") || strings.Contains(nmapOutput, "Darwin") {
		return "macOS"
	}

	return "Unknown"
}

// SummarizeServices creates a human-readable summary of detected services
func (c *Client) SummarizeServices(services []ServiceInfo) string {
	if len(services) == 0 {
		return "No services detected"
	}

	summary := fmt.Sprintf("Detected %d service(s):\n", len(services))

	// Group by service type
	serviceGroups := make(map[string][]ServiceInfo)
	for _, svc := range services {
		serviceGroups[svc.Service] = append(serviceGroups[svc.Service], svc)
	}

	for svcName, svcs := range serviceGroups {
		summary += fmt.Sprintf("  • %s: ", svcName)
		ports := make([]string, len(svcs))
		for i, svc := range svcs {
			ports[i] = fmt.Sprintf("%d/%s", svc.Port, svc.Protocol)
		}
		summary += strings.Join(ports, ", ") + "\n"
	}

	return summary
}
