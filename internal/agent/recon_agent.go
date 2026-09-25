package agent

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
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

	start := time.Now()

	var out *models.ToolOutput
	var err error
	switch scanType {
	case "quick":
		out, err = r.executor.QuickNmapScan(target)
	case "ping":
		out, err = r.executor.NmapPingDiscovery(target)
	default:
		out, err = r.executor.ScanWithNmap(target)
	}

	if err != nil {
		r.recordFailure(start)
		return nil, fmt.Errorf("recon scan failed: %w", err)
	}
	if ctx.Err() != nil {
		r.recordFailure(start)
		return nil, ctx.Err()
	}
	if out.ExitCode != 0 {
		r.recordFailure(start)
		return nil, fmt.Errorf("nmap exited %d: %s", out.ExitCode, strings.TrimSpace(out.Stderr))
	}

	hosts := parseNmapXML(out.Stdout)

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

// --- nmap XML parsing (nmap -oX output) ---

type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status  nmapStatus   `xml:"status"`
	Address nmapAddress  `xml:"address"`
	Host    nmapHostname `xml:"hostnames>hostname"`
	OS      nmapOS       `xml:"os>osmatch"`
	Ports   []nmapPort   `xml:"ports>port"`
}

type nmapStatus struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr string `xml:"addr,attr"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapOS struct {
	Name string `xml:"name,attr"`
}

type nmapPort struct {
	PortID   int           `xml:"portid,attr"`
	Protocol string        `xml:"protocol,attr"`
	State    nmapPortState `xml:"state"`
	Service  nmapService   `xml:"service"`
}

type nmapPortState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

// parseNmapXML converts nmap XML output into HostResult structures.
// It is lenient: hosts with no data are skipped, malformed XML yields an
// empty result rather than an error (the raw stdout is still in the task log).
func parseNmapXML(xmlData string) []HostResult {
	var run nmapRun
	if err := xml.Unmarshal([]byte(xmlData), &run); err != nil {
		return []HostResult{}
	}

	hosts := make([]HostResult, 0, len(run.Hosts))
	for _, h := range run.Hosts {
		hr := HostResult{
			IP:    h.Address.Addr,
			Host:  h.Host.Name,
			State: h.Status.State,
			OS:    h.OS.Name,
			Ports: make([]ServiceResult, 0, len(h.Ports)),
		}
		if hr.IP == "" {
			continue
		}
		for _, p := range h.Ports {
			if p.State.State != "open" {
				continue
			}
			hr.Ports = append(hr.Ports, ServiceResult{
				Port:     p.PortID,
				Protocol: p.Protocol,
				State:    p.State.State,
				Service:  p.Service.Name,
				Product:  p.Service.Product,
				Version:  p.Service.Version,
			})
		}
		hosts = append(hosts, hr)
	}

	return hosts
}
