package agent

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/tools"
)

// HostResult represents a discovered host with its open ports.
type HostResult struct {
	IP    string          `json:"ip"`
	Host  string          `json:"hostname,omitempty"`
	State string          `json:"state"`
	OS    string          `json:"os,omitempty"`
	Ports []ServiceResult `json:"ports"`
}

// ServiceResult represents an open port with service/version info.
type ServiceResult struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Product  string `json:"product,omitempty"`
	Version  string `json:"version,omitempty"`
}

// ReconResult is the structured output of a reconnaissance task.
type ReconResult struct {
	Target   string       `json:"target"`
	ScanType string       `json:"scan_type"`
	Hosts    []HostResult `json:"hosts"`
	Duration int          `json:"duration_ms"`
}

// ReconnaissanceAgent discovers hosts, ports, and services via nmap.
type ReconnaissanceAgent struct {
	BaseAgent
	executor *tools.Executor
}

// NewReconnaissanceAgent creates a recon agent backed by the tool executor.
func NewReconnaissanceAgent(executor *tools.Executor) *ReconnaissanceAgent {
	return &ReconnaissanceAgent{
		BaseAgent: NewBaseAgent(AgentTypeReconnaissance),
		executor:  executor,
	}
}

// Execute runs an nmap scan according to task params:
//
//	params.target    (required) — host, domain, or CIDR
//	params.scan_type (optional) — "full" (default), "quick", or "ping"
func (r *ReconnaissanceAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	if err := r.Validate(task); err != nil {
		return nil, err
	}

	target := getStringParam(task.Params, "target")
	if target == "" {
		return nil, errors.New("params.target is required")
	}
	scanType := getStringParam(task.Params, "scan_type")
	if scanType == "" {
		scanType = "full"
	}

	var args string
	switch scanType {
	case "quick":
		args = "-F"
	case "ping":
		args = "-sn"
	default:
		args = "-sV -p-"
	}

	start := time.Now()

	out, err := r.executor.Execute(ctx, "nmap", map[string]string{
		"target": target,
		"args":   args,
	})
	if err != nil {
		r.recordFailure(start)
		return nil, fmt.Errorf("recon scan failed: %w", err)
	}
	if ctx.Err() != nil {
		r.recordFailure(start)
		return nil, ctx.Err()
	}
	if out == nil || !out.Success {
		r.recordFailure(start)
		stderr := ""
		if out != nil {
			stderr = strings.TrimSpace(out.Stderr)
		}
		return nil, fmt.Errorf("nmap failed: %s", stderr)
	}

	hosts := parseNmapText(out.Stdout)

	result := &ReconResult{
		Target:   target,
		ScanType: scanType,
		Hosts:    hosts,
		Duration: int(time.Since(start).Milliseconds()),
	}

	r.recordSuccess(start)
	return result, nil
}

// Validate ensures the recon task is well-formed.
func (r *ReconnaissanceAgent) Validate(task *Task) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if task.EngagementID == "" {
		return errors.New("engagement_id required")
	}
	if getStringParam(task.Params, "target") == "" {
		return errors.New("params.target required")
	}
	return nil
}

// Stop terminates the agent.
func (r *ReconnaissanceAgent) Stop() error {
	r.Status = StatusTerminated
	return nil
}

// --- nmap text output parsing (default nmap stdout) ---

var (
	nmapReportRe = regexp.MustCompile(`(?m)^Nmap scan report for (\S+)\s*(?:\(([^)]+)\))?`)
	portLineRe   = regexp.MustCompile(`^(\d+)/(tcp|udp|sctp)\s+(\w+)\s+(\S*)?\s*(.*)$`)
	osLineRe     = regexp.MustCompile(`(?i)^(?:Running|OS details|OS CPE):\s*(.+)$`)
)

// parseNmapText converts nmap's human-readable output into HostResult
// structures. It is lenient: hosts with no data are skipped and malformed
// lines are ignored.
func parseNmapText(output string) []HostResult {
	lines := strings.Split(output, "\n")
	var hosts []HostResult
	var current *HostResult

	flush := func() {
		if current != nil {
			hosts = append(hosts, *current)
			current = nil
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		if m := nmapReportRe.FindStringSubmatch(line); m != nil {
			flush()
			// "Nmap scan report for hostname (ip)" or "for ip"
			ip := m[1]
			hostname := ""
			if !isIP(m[1]) {
				hostname = m[1]
				if m[2] != "" && isIP(m[2]) {
					ip = m[2]
				}
			}
			current = &HostResult{IP: ip, Host: hostname, State: "up", Ports: []ServiceResult{}}
			continue
		}

		if current == nil {
			continue
		}

		if m := portLineRe.FindStringSubmatch(line); m != nil {
			port, _ := strconv.Atoi(m[1])
			parts := strings.Fields(m[4] + " " + m[5])
			service, product, version := "", "", ""
			if len(parts) > 0 {
				service = parts[0]
			}
			if len(parts) > 1 {
				product = parts[1]
			}
			if len(parts) > 2 {
				version = strings.Join(parts[2:], " ")
			}
			if m[3] == "open" {
				current.Ports = append(current.Ports, ServiceResult{
					Port:     port,
					Protocol: m[2],
					State:    m[3],
					Service:  service,
					Product:  product,
					Version:  version,
				})
			}
			continue
		}

		if m := osLineRe.FindStringSubmatch(line); m != nil && current.OS == "" {
			current.OS = guessOS(m[1])
		}
	}

	flush()
	return hosts
}

// isIP reports whether s looks like an IPv4 address.
func isIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

// guessOS extracts a coarse OS family from an nmap OS line.
func guessOS(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "windows"):
		return "Windows"
	case strings.Contains(low, "linux"):
		return "Linux"
	case strings.Contains(low, "freebsd"), strings.Contains(low, "openbsd"), strings.Contains(low, "netbsd"):
		return "BSD"
	case strings.Contains(low, "mac"), strings.Contains(low, "darwin"):
		return "macOS"
	case strings.Contains(low, "solaris"):
		return "Solaris"
	default:
		return strings.TrimSpace(s)
	}
}
